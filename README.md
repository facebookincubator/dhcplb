# What is dhcplb?

`dhcplb` is Facebook's implementation of a DHCP v4/v6 relayer with load
balancing capabilities.
Facebook currently uses it in production, and it's deployed at global scale
across all of our data centers.

# Why did you do that?

Facebook uses DHCP to provide network configuration to bare-metal machines at
provisioning phase and to assign IPs to out-of-band interfaces.  

`dhcplb` was created because the previous infrastructure surrounding DHCP led
to very unbalanced load across the DHCP servers in a region when simply using
Anycast+ECMP alone (for example 1 server out of 10 would receive >65% of
requests).

Facebook's DHCP infrastructure was [presented at SRECon15 Ireland](https://www.usenix.org/conference/srecon15europe/program/presentation/failla).

# Why not use an existing load balancer?

* All the relayer implementations available on the internet lack the load
balancing functionality.
* Having control of the code gives you the the ability to:
  * perform A/B testing on new builds of our DHCP server
  * implement override mechanism
  * implement anything additional you need

# How do you use `dhcplb` at Facebook?

This picture shows how we have deployed `dhcplb` in our production
infrastructure:

![DHCPLB deployed at Facebook](/docs/dhcplb-fb-deployment.jpg)

TORs (Top of Rack switch) at Facebook run DHCP relayers, these relayers are
responsible for relaying broadcast DHCP traffic (DISCOVERY and SOLICIT
messages) originating within their racks to anycast VIPs, one DHCPv4 and one
for DHCPv6.

In a Cisco switch the configuration would look like this:

```
ip helper-address 10.127.255.67
ipv6 dhcp relay destination 2401:db00:eef0:a67::
```

We have a bunch of `dhcplb` [Tupperware](https://blog.docker.com/2014/07/dockercon-video-containerized-deployment-at-facebook/) instances in every region listening on
those VIPs.
They are responsible for received traffic relayed by TORs agents and load
balancing them amongst the actual KEA dhcp servers distributed across clusters
in that same region.

The configuration for `dhcplb` consists of 3 files:

* json config file: contains the main configuration for the server as explained in the [Getting Started](docs/getting-started.md) section
* host lists file: contains a list of dhcp servers, one per line, those are the servers `dhcplb` will try to balance on
* overrides file: a file containing per mac overrides. See the [Getting Started](docs/getting-started.md) section.

# What does it support?

`dhcplb` is an implementation of a DHCP relay agent, (mostly) implementing the
following RFCs:

* [RFC 2131](https://tools.ietf.org/html/rfc2131) (DHCPv4)
* [RFC 3315](https://tools.ietf.org/html/rfc3315) (DHCPv6)

Note that currently `dhcplb` does not support relaying broadcasted DHCPv4
DISCOVERY packets or DHCPv6 SOLICIT packets sent to `ff02::1:2` multicast
address. We don't need this in our production environment but adding that
support should be trivial though. PR are welcome!

# How does it work?

When operating in v4 mode `dhcplb` will relay relayed messages coming from other
relayers (in our production network those are rack switches), the response from
dhcp servers will be relayed back to the rack switches:

```
dhcp client <---> rsw relayer ---> dhcplb ---> dhcp server
                      ^                             |
                      |                             |
                      +-----------------------------+
```

In DHCPv6 mode `dhcplb` will operate normally, responses by the dhcp server
will traverse the load balancer.

# Requirements

`dhcplb` relies on the following libraries that you can get using the `go get`
command:

```
$ go get github.com/fsnotify/fsnotify
$ go get github.com/golang/glog
$ go get github.com/krolaw/dhcp4
$ go get github.com/facebookgo/ensure
```

# Installation

To install `dhcplb` in your `$GOPATH` simply run:

```
$ go get github.com/facebookincubator/dhcplb
```

This will fetch the source code and write it into
`$GOPATH/src/github.com/facebookincubator/dhcplb`, compile the binary and put
it in `$GOPATH/bin/dhcplb`.

# Cloning

If you wish to clone the repo you can do the following:


```
$ mkdir -p $GOPATH/src/github.com/facebookincubator
$ cd $_
$ git clone https://github.com/facebookincubator/dhcplb
$ go install github.com/facebookincubator/dhcplb
```

# Run unit tests

You can run tests with:

```
$ cd $GOPATH/src/github.com/facebookincubator/dhcplb/lib
$ go test
```

# Getting Started and extending `dhcplb`

`dhcplb` can be run out of the box after compilation.

To start immediately, you can run
`sudo dhcplb -config config.json -version 6`.
That will start the server in v6 mode using the default configuration.

Should you need to integrate `dhcplb` with your infrastructure please
see [Getting Started](docs/getting-started.md).

# TODOs / future improvements

TODOs and improvements are tracked [here](https://github.com/facebookincubator/dhcplb/issues?q=is%3Aissue+is%3Aopen+label%3Aenhancement)

PRs are welcome!

# Who wrote it?

`dhcplb` started in April 2016 during a 3 days hackathon in the Facebook
Dublin office, the hackathon project proved the feasibility of the tool.
In June we were joined by Vinnie Magro (@vmagro) for a 3 months internship in
which he worked with two production engineers on turning the hack into a
production ready system.
`dhcplb` has been deployed globally and currently balances all production DHCP
traffic efficiently across our KEA DHCP servers.

Hackathon project members:

* Angelo Failla ([@pallotron](https://github.com/pallotron)), Production Engineer
* Roman Gushchin ([@rgushchin](https://github.com/rgushchin)), Production Engineer
* Mateusz Kaczanowski ([@mkaczanowski](https://github.com/mkaczanowski)), Production Engineer
* Jake Bunce, Network Engineer

Internship project members:

* Vinnie Magro ([@vmagro](https://github.com/vmagro)), Production Engineer intern
* Angelo Failla (@pallotron), Intern mentor, Production Engineer
* Mateusz Kaczanowski (@mkaczanowski), Production Engineer
