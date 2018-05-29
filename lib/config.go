/**
 * Copyright (c) 2016-present, Facebook, Inc.
 * All rights reserved.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree. An additional grant
 * of patent rights can be found in the PATENTS file in the same directory.
 */

package dhcplb

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
	"time"
)

// ConfigProvider is an interface which provides methods to fetch the
// HostSourcer, parse extra configuration, provide additional load balancing
// implementations and how to handle dhcp requests in server mode
type ConfigProvider interface {
	NewHostSourcer(
		sourcerType, args string, version int) (DHCPServerSourcer, error)
	ParseExtras(extras json.RawMessage) (interface{}, error)
	NewDHCPBalancingAlgorithm(version int) (DHCPBalancingAlgorithm, error)
	NewHandler(interface{}) (Handler, error)
}

// Config represents the server configuration.
type Config struct {
	Version              int
	Addr                 *net.UDPAddr
	Algorithm            DHCPBalancingAlgorithm
	ServerUpdateInterval time.Duration
	PacketBufSize        int
	Handler              Handler
	HostSourcer          DHCPServerSourcer
	FreeConnTimeout      time.Duration
	RCRatio              uint32
	Overrides            map[string]Override
	Extras               interface{}
	CacheSize           int
	CacheRate           int
	Rate                int
}

// Override represents the dhcp server or the group of dhcp servers (tier) we
// want to send packets to.
type Override struct {
	// note that Host override takes precedence over Tier
	Host       string `json:"host"`
	Tier       string `json:"tier"`
	Expiration string `json:"expiration"`
}

// Overrides is a struct that holds v4 and v6 list of overrides.
// The keys of the map are mac addresses.
type Overrides struct {
	V4 map[string]Override `json:"v4"`
	V6 map[string]Override `json:"v6"`
}

// LoadConfig will take the path of the json file, the path of the override json
// file, an integer version and a ConfigProvider and will return a pointer to
// a Config object.
func LoadConfig(path, overridesPath string, version int, provider ConfigProvider) (*Config, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	overridesFile := []byte{}
	// path length of 0 means we aren't using overrides
	if len(overridesPath) != 0 {
		err = nil
		if overridesFile, err = ioutil.ReadFile(overridesPath); err != nil {
			return nil, err
		}
	}
	return ParseConfig(file, overridesFile, version, provider)
}

// ParseConfig will take JSON config files, a version and a ConfigProvider,
// and return a pointer to a Config struct
func ParseConfig(jsonConfig, jsonOverrides []byte, version int, provider ConfigProvider) (*Config, error) {
	var combined combinedconfigSpec
	if err := json.Unmarshal(jsonConfig, &combined); err != nil {
		glog.Errorf("Failed to parse JSON: %s", err)
		return nil, err
	}
	var spec configSpec
	if version == 4 {
		spec = combined.V4
	} else if version == 6 {
		spec = combined.V6
	}

	var overrides map[string]Override
	if len(jsonOverrides) == 0 {
		overrides = make(map[string]Override)
	} else {
		var err error
		overrides, err = parseOverrides(jsonOverrides, version)
		if err != nil {
			glog.Errorf("Failed to load overrides: %s", err)
			return nil, err
		}
	}
	glog.Infof("Loaded %d override(s)", len(overrides))
	return newConfig(&spec, overrides, provider)
}

// WatchConfig will keep watching for changes to both config and override json
// files. It uses fsnotify library (it uses inotify in Linux), and call
// LoadConfig when it an inotify event signals the modification of the json
// files.
// It returns a configBroadcaster which the a goroutine in the main will use
// to reload the configuration when it changes.
func WatchConfig(
	configPath, overridesPath string, version int, provider ConfigProvider,
) (*ConfigBroadcaster, chan error, error) {
	configChan := make(chan *Config)
	errChan := make(chan error)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	// strings containing the real path of a config files, if they are symlinks
	var realConfigPath string
	var realOverridesPath string

	err = watcher.Add(filepath.Dir(configPath))
	if err != nil {
		return nil, nil, err
	}
	realConfigPath, err = filepath.EvalSymlinks(configPath)
	if err == nil {
		// configPath is a symlink, also watch the pointee
		err = watcher.Add(realConfigPath)
	}

	// setup watcher on overrides file if present
	if len(overridesPath) > 0 {
		err = watcher.Add(filepath.Dir(overridesPath))
		if err != nil {
			glog.Errorf("Failed to start fsnotify on overrides config file: %s", err)
			return nil, nil, err
		}
		realOverridesPath, err = filepath.EvalSymlinks(overridesPath)
		if err == nil {
			// overridesPath is a symlink, also watch the pointee
			err = watcher.Add(realOverridesPath)
		}
	}

	// watch for fsnotify events
	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				// ignore Remove events
				if ev.Op&fsnotify.Remove == fsnotify.Remove {
					continue
				}
				// only care about symlinks and target of symlinks
				if ev.Name == overridesPath || ev.Name == configPath ||
					ev.Name == realOverridesPath || ev.Name == realConfigPath {
					glog.Infof("Configuration file changed (%s), reloading", ev)
					config, err := LoadConfig(
						configPath, overridesPath, version, provider)
					if err != nil {
						errChan <- err
						panic(err) // fail hard
					}
					configChan <- config
				}
			case err := <-watcher.Errors:
				glog.Errorf("fsnotify error: %s", err)
			}
		}
	}()

	return NewConfigBroadcaster(configChan), errChan, nil
}

// configSpec holds the raw json configuration.
type configSpec struct {
	Path                 string
	Version              int             `json:"version"`
	ListenAddr           string          `json:"listen_addr"`
	Port                 int             `json:"port"`
	AlgorithmName        string          `json:"algorithm"`
	UpdateServerInterval int             `json:"update_server_interval"`
	PacketBufSize        int             `json:"packet_buf_size"`
	HostSourcer          string          `json:"host_sourcer"`
	FreeConnTimeout      int             `json:"free_conn_timeout"`
	RCRatio              uint32          `json:"rc_ratio"`
	Extras               json.RawMessage `json:"extras"`
	CacheSize           int             `json:"throttle_cache_size"`
	CacheRate           int             `json:"throttle_cache_rate"`
	Rate                int             `json:"throttle_rate"`
}

type combinedconfigSpec struct {
	V4 configSpec `json:"v4"`
	V6 configSpec `json:"v6"`
}

func (c *configSpec) sourcer(provider ConfigProvider) (DHCPServerSourcer, error) {
	// Load the DHCPServerSourcer implementation
	sourcerInfo := strings.Split(c.HostSourcer, ":")
	sourcerType := sourcerInfo[0]
	stable := sourcerInfo[1]
	rc := ""
	if strings.Index(sourcerInfo[1], ",") > -1 {
		sourcerArgs := strings.Split(sourcerInfo[1], ",")
		stable = sourcerArgs[0]
		rc = sourcerArgs[1]
	}
	switch sourcerType {

	default:
		return provider.NewHostSourcer(sourcerType, sourcerInfo[1], c.Version)

	case "file":
		sourcer, err := NewFileSourcer(stable, rc, c.Version)
		if err != nil {
			glog.Fatalf("Can't load FileSourcer")
		}
		return sourcer, err
	}
}

func (c *configSpec) algorithm(provider ConfigProvider) (DHCPBalancingAlgorithm, error) {
	// Balancing algorithms coming with the dhcplb source code
	modulo := new(modulo)
	rr := new(roundRobin)
	algorithms := map[string]DHCPBalancingAlgorithm{
		modulo.Name(): modulo,
		rr.Name():     rr,
	}
	// load other non default algorithms from the ConfigProvider
	providedAlgo, err := provider.NewDHCPBalancingAlgorithm(c.Version)
	if err != nil {
		glog.Fatalf("Provided load balancing implementation error: %s", err)
	}
	if providedAlgo != nil {
		// TODO: check that the name is not used, if not then fatal.
		algorithms[providedAlgo.Name()] = providedAlgo
	}
	lb, ok := algorithms[c.AlgorithmName]
	if !ok {
		supported := []string{}
		for k := range algorithms {
			supported = append(supported, k)
		}
		glog.Fatalf(
			"'%s' is not a supported balancing algorithm. "+
				"Supported balancing algorithms are: %v",
			c.AlgorithmName, supported)
		return nil, fmt.Errorf(
			"'%s' is not a supported balancing algorithm", c.AlgorithmName)
	}
	lb.SetRCRatio(c.RCRatio)
	return lb, nil
}

func newConfig(spec *configSpec, overrides map[string]Override, provider ConfigProvider) (*Config, error) {
	if spec.Version != 4 && spec.Version != 6 {
		return nil, fmt.Errorf("Supported version: 4, 6 - not %d", spec.Version)
	}

	targetIP := net.ParseIP(spec.ListenAddr)
	if targetIP == nil {
		return nil, fmt.Errorf("Unable to parse IP %s", targetIP)
	}
	addr := &net.UDPAddr{
		IP:   targetIP,
		Port: spec.Port,
		Zone: "",
	}

	algo, err := spec.algorithm(provider)
	if err != nil {
		return nil, err
	}
	sourcer, err := spec.sourcer(provider)
	if err != nil {
		return nil, err
	}

	// extras
	extras, err := provider.ParseExtras(spec.Extras)
	if err != nil {
		return nil, err
	}
	handler, err := provider.NewHandler(extras)
	if err != nil {
		return nil, err
	}

	return &Config{
		Version:   spec.Version,
		Addr:      addr,
		Algorithm: algo,
		ServerUpdateInterval: time.Duration(
			spec.UpdateServerInterval) * time.Second,
		PacketBufSize:   spec.PacketBufSize,
		Handler:         handler,
		HostSourcer:     sourcer,
		FreeConnTimeout: time.Duration(spec.FreeConnTimeout) * time.Second,
		RCRatio:         spec.RCRatio,
		Overrides:       overrides,
		Extras:          extras,
		CacheSize:      spec.CacheSize,
		CacheRate:      spec.CacheRate,
		Rate:           spec.Rate,
	}, nil
}

func parseOverrides(file []byte, version int) (map[string]Override, error) {
	overrides := Overrides{}
	err := json.Unmarshal(file, &overrides)
	if err != nil {
		glog.Errorf("Failed to parse JSON: %s", err)
		return nil, err
	}
	if version == 4 {
		return overrides.V4, nil
	} else if version == 6 {
		return overrides.V6, nil
	}
	return nil, fmt.Errorf("Unsupported version %d, must be 4|6", version)
}

// ConfigBroadcaster is a convenience struct that broadcasts its input channel
// to a list of receivers.
type ConfigBroadcaster struct {
	input     <-chan *Config
	receivers []chan<- *Config
}

// NewConfigBroadcaster returns an instance of ConfigBroadcaster
func NewConfigBroadcaster(input <-chan *Config) *ConfigBroadcaster {
	bcast := &ConfigBroadcaster{
		input: input,
	}
	go bcast.listen()
	return bcast
}

func (c *ConfigBroadcaster) listen() {
	for {
		config := <-c.input
		for _, receiver := range c.receivers {
			receiver <- config
		}
	}
}

// NewReceiver allows the caller to register to receive new Config messages
// when the server config changes. This is typically used by a main go routine
// to reload the server configuration.
func (c *ConfigBroadcaster) NewReceiver() <-chan *Config {
	channel := make(chan *Config, 1)
	c.receivers = append(c.receivers, channel)
	return channel
}
