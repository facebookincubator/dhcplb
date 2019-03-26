/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"

	dhcplb "github.com/facebookincubator/dhcplb/lib"
	"github.com/golang/glog"
)

// Program parameters
var (
	version       = flag.Int("version", 4, "Run in v4/v6 mode")
	configPath    = flag.String("config", "", "Path to JSON config file")
	overridesPath = flag.String("overrides", "", "Path to JSON overrides file")
	pprofPort     = flag.Int("pprof", 0, "Port to run pprof HTTP server on")
	serverMode    = flag.Bool("server", false, "Run in server mode. The default is relay mode.")
)

func main() {
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")

	if *configPath == "" {
		glog.Fatal("Config file is necessary")
		return
	}

	if *pprofPort != 0 {
		go func() {
			glog.Infof("Started pprof server on port %d", *pprofPort)
			err := http.ListenAndServe(fmt.Sprintf(":%d", *pprofPort), nil)
			if err != nil {
				glog.Fatal("Error starting pprof server: ", err)
			}
		}()
	}

	logger := NewGlogLogger()

	// load initial config
	provider := NewDefaultConfigProvider()
	config, err := dhcplb.LoadConfig(
		*configPath, *overridesPath, *version, provider)
	if err != nil {
		glog.Fatalf("Failed to load config: %s", err)
		return
	}

	// start watching config
	configChan, err := dhcplb.WatchConfig(
		*configPath, *overridesPath, *version, provider)
	if err != nil {
		glog.Fatalf("Failed to watch config: %s", err)
		return
	}

	server, err := dhcplb.NewServer(config, *serverMode, logger)
	if err != nil {
		glog.Fatal(err)
		return
	}

	// update server config whenever file changes
	go func() {
		for {
			select {
			case config := <-configChan:
				glog.Info("Config changed")
				server.SetConfig(config)
			}
		}
	}()

	glog.Infof("Starting dhcplb in v%d mode", *version)
	glog.Fatal(server.ListenAndServe())
}
