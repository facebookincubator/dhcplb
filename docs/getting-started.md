Getting Started
===============

Out of the box `dhcplb` supports loading DHCP server lists from text files and logging to stderr with `glog`.  
All configuration files supplied to `dhcplb` (config, overrides and DHCP server files) are watched for changes using [`fsnotify`](https://github.com/fsnotify/fsnotify) and hot-reloaded without restarting the server.
Configuration is provided to the program via a JSON file

```javascript
{
  "v4": {
    "version": 4, // DHCP operation mode
    "listen_addr": "0.0.0.0", // address to bind the receiving socket to
    "port": 67, // port to listen on
    "packet_buf_size": 1024, // size of buffer to allocate for incoming packet
    "update_server_interval": 30, // how often to refresh server list (in seconds)
    "free_conn_timeout": 30, // how long to wait after removal before closing a connection to a server (in seconds)
    "algorithm": "xid", // balancing algorithm, supported are xid and rr (client hash and roundrobin)
    "host_sourcer": "file:hosts-v4.txt", // load DHCP server list from hosts-v4.txt
    "rc_ratio": 0, // what percentage of requests should go to RC servers
    "throttle_cache_size": 1024, // cache size for number of throttling objects for unique clients
    "throttle_cache_rate": 128, // rate value for throttling cache invalidation (per second)
    "throttle_rate": 256 // rate value for request per second
  },
  ... (same options for "v6") ...
```

Overrides
---------

`dhcplb` supports configurable overrides for individual machines. A MAC address
can be configured to point to a specific DHCP server IP or to a "tier" (group)
of servers.
Overrides are defined in a JSON file and the path is passed to `dhcplb` as the
command-line arg `-overrides`.

```javascript
{
  "v4": {
    "12:34:56:78:90:ab": {
        "host": "173.252.90.132"
    },
    "fe:dc:ba:09:87:65": {
        "tier": "myGroup"
    }
  },
  "v6": {
  }
}
```

With this overrides file, DHCPv4 requests coming from MAC `12:34:56:78:90:ab`
will be sent to the DHCP server at `173.252.90.132`, and requests from MAC
`fe:dc:ba:09:87:65` will be sent to the tier of servers `myGroup` (a server will
be picked according to the balancing algorithm's selection from the list of
servers returned by the `GetServersFromTier(tier string)` function of the
`DHCPServerSourcer` being used).
Overrides may be associated with an expiration timestamp in the form
"YYYY/MM/DD HH:MM TIMEZONE_OFFSET", where TIMEZONE_OFFSET is
the timezone offset with respect to UTC. `dhcplb` will convert the timestamp
in the local timezone and ignore expired overrides.

```javascript
{
  "v4": {
    "12:34:56:78:90:ab": {
        "host": "173.252.90.132",
        "expiration": "2017/05/06 14:00 +0000"
    },
    "fe:dc:ba:09:87:65": {
        "tier": "myGroup"
    }
  },
  "v6": {
  }
}
```


Throttling
----------

`dhcplb` keeps track of the request rate per second for each backend DHCP
server.
It can be set through `throttle_rate` configuration parameter.
Requests exceeding this limit will be logged and dropped. For 0 or negative
values no throttling will be done, and no cache will be created.

An LRU cache is used to keep track of rate information for each backend DHCP
server.
Cache size can be set through `throttle_cache_size`. To prevent fast cache
invalidation from malicious clients, `dhcplb` also keeps track of the number of
new clients being added to the cache (per second). This behavior can be set
through `throttle_cache_rate` configuration parameter. For 0 or negative values
no cache rate limiting will be done.


A/B testing
-----------

`dhcplb` supports sending a percentage of requests to servers marked as RC and
the rest to Stable servers.
This percentage is configurable via the `rc_ratio` JSON option.
Using the A/B testing functionality requires providing two lists of servers,
this can be done via the built in filesourcer by specifying the `host_sourcer`
option as `"file:<stable_path>,<rc_path>"`

Extending DHCPLB
----------------

It's possible to extend `dhcplb` to modify the way it fetches the list of
DHCP servers, or have a different logging implementation, or add different
balancing algorithms.
At the moment this is a bit complex but we will work on ways to make it easier.

### Adding a new balancing algorithm.

Adding a new algorithm can be done by implementing something that matches
the `DHCPBalancingAlgorithm` interface:

```go
type DHCPBalancingAlgorithm interface {
	selectServerFromList(list []*DHCPServer, message *DHCPMessage) (*DHCPServer, error)
	selectRatioBasedDhcpServer(message *DHCPMessage) (*DHCPServer, error)
	updateStableServerList(list []*DHCPServer) error
	updateRCServerList(list []*DHCPServer) error
	setRCRatio(ratio uint32)
  Name() string
}
```

Then add it to the `algorithms` map in the `configSpec.algorithm` function, in
the `config.go` file.
Do that if you want to share the algorithm with the community.

If, however, you need to implement something that you can't share, because, for
example, it's internal and specific to your infra, you can write something that
implements the `ConfigProvider` interface, in particular the
`NewDHCPBalancingAlgorithm` function.

### Adding more configuration options.

More configuration options can be added to the config JSON file using the
`ConfigProvider` interface:

```go
type ConfigProvider interface {
	NewHostSourcer(sourcerType, args string, version int) (DHCPServerSourcer, error)
	ParseExtras(extras json.RawMessage) (interface{}, error)
  NewDHCPBalancingAlgorithm(version int) (DHCPBalancingAlgorithm, error)
}
```

The `NewHostSourcer` function is passed values from the `host_sourcer` config option
with the `sourcerType` being the part of the string before the `:` and `args` the
remaining portion. ex: `file:hosts-v4.txt,hosts-v4-rc.txt` will have `sourcerType="file"`
and `args="hosts-v4.txt,hosts-v4-rc-txt"`.
The default `Config` loader is able to instantiate a `FileSourcer` by itself, so
`NewHostSourcer` can simply return `nil, nil` unless you are using a custom sourcer
implementation.

Any struct can be returned from the `ParseExtras` function and used elsewhere in
the code via the `Extras` member of a `Config` struct.

As mentioned in the section before `NewDHCPBalancingAlgorithm` can be used
to return your own specific load balancing implementation.

### Write your own logic to source list of DHCP servers

If you want to change the way `dhcplb` sources the list of DHCP servers (for
example you want to source them from a backend system like a database) you can
have something implementing the `DHCPServerSourcer` interface:

```go
type DHCPServerSourcer interface {
  GetStableServers() ([]*DHCPServer, error)
  GetRCServers() ([]*DHCPServer, error)
  // get servers from a specific named group (this is used with overrides)
  GetServersFromTier(tier string) ([]*DHCPServer, error)
}
```

Then implement your own `ConfigProvider` interface and make it return a
`DHCPServerSourcer`. Then in the main you can replace `NewDefaultConfigProvider`
with your own `ConfigProvider` implementation.

Usage
-----
```
$ ./dhcplb -h
Usage of ./dhcplb:
  -alsologtostderr
      log to standard error as well as files
  -config string
      Path to JSON config file
  -log_backtrace_at value
      when logging hits line file:N, emit a stack trace (default :0)
  -log_dir string
      If non-empty, write log files in this directory
  -logtostderr
      log to standard error instead of files
  -overrides string
      Path to JSON overrides file
  -pprof int
      Port to run pprof HTTP server on
  -stderrthreshold value
      logs at or above this threshold go to stderr
  -v value
      log level for V logs
  -version int
      Run in v4/v6 mode (default 4)
  -vmodule value
      comma-separated list of pattern=N settings for file-filtered logging
```
