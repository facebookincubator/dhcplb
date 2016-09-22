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
	"bufio"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

// FileSourcer holds various information about json the config files, list of
// stable and rc servers, the fsnotify Watcher and stuff needed for
// synchronization.
type FileSourcer struct {
	stablePath    string
	rcPath        string
	version       int
	watcher       *fsnotify.Watcher
	lock          sync.RWMutex
	stableServers []*DHCPServer
	rcServers     []*DHCPServer
}

// NewFileSourcer returns a new FileSourcer, stablePath and rcPath are the paths
// of the text files containing list of servers. If rcPath is empty it will be
// ignored, stablePath must be not null, version is the protocol version and
// should be either 4 or 6.
func NewFileSourcer(stablePath, rcPath string, version int) (*FileSourcer, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		glog.Fatal(err)
	}
	err = watcher.Add(stablePath)
	if err != nil {
		glog.Fatalf("Error watching stable: %s", err)
	}
	// RC is optional, only add to fsnotify and read if rcPath is present
	if len(rcPath) > 0 {
		err = watcher.Add(rcPath)
		if err != nil {
			glog.Fatalf("Error watching rc: %s", err)
		}
	}
	sourcer, err := &FileSourcer{
		stablePath: stablePath,
		rcPath:     rcPath,
		version:    version,
		watcher:    watcher,
	}, nil
	sourcer.lock.Lock()
	sourcer.stableServers, err = sourcer.GetServersFromTier(stablePath)
	if err != nil {
		glog.Errorf("Failed to load stable servers: %s", err)
	}
	if len(rcPath) > 0 {
		sourcer.rcServers, err = sourcer.GetServersFromTier(rcPath)
		if err != nil {
			glog.Errorf("Failed to load RC servers: %s", err)
		}
	}
	sourcer.lock.Unlock()
	go sourcer.watchFsnotifyEvents()
	return sourcer, err
}

// GetServersFromTier returns a list of DHCPServer from a file
func (fs *FileSourcer) GetServersFromTier(path string) ([]*DHCPServer, error) {
	inputFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()
	scanner := bufio.NewScanner(inputFile)

	var servers []*DHCPServer
	for scanner.Scan() {
		tokens := strings.Split(scanner.Text(), ":")
		var port int64
		if len(tokens) == 1 {
			port = 67
		} else {
			var errPort error
			port, errPort = strconv.ParseInt(tokens[1], 10, 32)
			if errPort != nil {
				glog.Errorf("Can't convert port %s to int", tokens[1])
				continue
			}
		}
		hostname := tokens[0]
		ip := net.ParseIP(hostname)
		if ip == nil {
			ips, err := net.LookupHost(hostname)
			if err != nil {
				glog.Errorf("Can't resolve IPv4 for %s", hostname)
				continue
			}
			for i := range ips {
				addr := net.ParseIP(ips[i])
				if addr != nil {
					if fs.version == 4 && addr.To4() != nil {
						ip = addr
						break
					}
					if fs.version == 6 && addr.To16() != nil {
						ip = addr
						break
					}
				}
			}
		}
		server := NewDHCPServer(hostname, ip, int(port))
		servers = append(servers, server)
	}
	return servers, nil
}

func (fs *FileSourcer) watchFsnotifyEvents() {
	for {
		select {
		case ev := <-fs.watcher.Events:
			if ev.Op&fsnotify.Write != 0 {
				glog.Infof("Event: %s File changed, reloading host list", ev)
				fs.lock.Lock()
				var err error
				fs.stableServers, err = fs.GetServersFromTier(fs.stablePath)
				if err != nil {
					glog.Errorf("Failed to load stable servers: %s", err)
				}
				if len(fs.rcPath) > 0 {
					fs.rcServers, err = fs.GetServersFromTier(fs.rcPath)
					if err != nil {
						glog.Errorf("Failed to RC stable servers: %s", err)
					}
				}
				fs.lock.Unlock()
			}
		case err := <-fs.watcher.Errors:
			glog.Error("Error: ", err)
		}
	}
}

// GetStableServers returns a list of stable dhcp servers
func (fs *FileSourcer) GetStableServers() ([]*DHCPServer, error) {
	return fs.stableServers, nil
}

// GetRCServers returns a list of rc dhcp servers
func (fs *FileSourcer) GetRCServers() ([]*DHCPServer, error) {
	return fs.rcServers, nil
}
