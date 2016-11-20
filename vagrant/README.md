# How to setup your test environment with Vagrant

The instruction below will help you bringing up a virtual lab where with VMs
sharing their own private network.
This assumes you are somewhat familiar with `vagrant`.
This has been tested under OSX but it should work find on Linux too.
This instructions are for DHCPv4 only, DHCPv6 will follow soon.

## Install dependencies

First, install `chef-dk` from https://downloads.chef.io/chef-dk/ 
On OSX you can use `brew`:

```
$ brew cask install Caskroom/cask/chefdk
```

Install `vagrant-berkshelf` plugin:

```
$ vagrant plugin install vagrant-berkshelf
$ cd ${PROJECT_ROOT}/vagrant/
$ berk install
$ berks vendor chef/cookbooks
```

## Start VMs

To start all the vms:

```
$ cd ${PROJECT_ROOT}/vagrant/
$ vagrant up
```

This will bring up the following VMs:

* `dhcpserver`: a VM running ISC `dhcpd` (both v4 and v6);
* `dhcprelay`: a VM running ISC `dhcrelay`;
* `dhcpclient`: a VM you can use to run `dhclient`, or `perfdhcp`;
* `dhcplb`: a VM running the `dhcplb` itself;


You can ssh into VMs using `vagrant ssh ${vm_name}`.

### `dhcpserver` VM

A VM that runs ISC `dhcpd` configured with a subnet. Something like:

```
subnet 192.168.50.0 netmask 255.255.255.0 {} 
subnet 192.168.51.0 netmask 255.255.255.0 {range 192.168.51.220 192.168.51.230;}
```

### `dhcplb` VM

A VM that runs `dhcplb` configured to redirect traffic to the VM above.

### `dhcprelay` VM

A VM that runs `dhcrelay` configured to relay traffic to the VM above.

### `dhcpclient` VM

A VM that contains dhclient and [ISC KEA's
perfdhcp](https://kea.isc.org/wiki/DhcpBenchmarking) utility.

## Development cycle

### Useful commands

On `dhcpclient`:

```
# perfdhcp -R 1 -4 -r 1200 -p 30 -t 1 -i 192.168.51.104
# dhclient -d -1 -v -pf /run/dhclient.eth1.pid -lf /var/lib/dhcp/dhclient.eth1.leases eth1
```

On `dhcprelay`:

```
# initctl list
# initctl (stop|start|restart) isc-dhcp-relay
# /usr/sbin/dhcrelay -d -4 -i eth1 -i eth2 192.168.50.104
```

On `dhcpserver`:

```
# /etc/init.d/isc-dhcp-server restart
```

On `dhcplb`:
