# How to setup your test environment with Vagrant

The instruction below will help you bringing up a VM lab running in a private
network.

This assumes you are somewhat familiar with `vagrant`.

Note: this is manual work for now, but we are working on `chef-solo` cookbooks.
Note: this instructions are for DHCPv4 only, DHCPv6 will follow soon.

First, install `chef-dk` from https://downloads.chef.io/chef-dk/ 
On OSX you can use `brew`:

```
$ brew cask install Caskroom/cask/chefdk
```

Install `vagran-berkshelf` plugin:

```
$ vagrant plugin install vagrant-berkshelf
$ cd ${PROJECT_ROOT}/vagrant/
$ berk install
$ berks vendor chef/cookbooks
```

Then start all the vms:

```
$ cd ${PROJECT_ROOT}/vagrant/
$ vagrant up
```

This will bring up the following VMs:

* `dhcpserver`: a VM running ISC `dhcpd` (both v4 and v6);
* `dhcprelay`: a VM running ISC `dhcrelay`;
* `dhcpclient`: a VM you can use to run `dhclient`, or `perfdhcp`;
* `dhcplb`: a VM running the `dhcplb` itself;

## Configure the VMs

### dhcpserver

Append the subnet to `/etc/dhcp/dhcpd.conf`:

```
$ vagrant ssh dhcpserver
vagrant@dhcpserver:~$ sudo su -
root@dhcpserver:~# echo subnet 192.168.50.0 netmask 255.255.255.0 {} >> /etc/dhcp/dhcpd.conf
root@dhcpserver:~# echo subnet 192.168.51.0 netmask 255.255.255.0 {range 192.168.51.220 192.168.51.230;} >> /etc/dhcp/dhcpd.conf
```

Make sure server is listening on the correct interface:

```
root@dhcpserver:~# vi /etc/default/isc-dhcp-server
INTERFACES="eth1"
```

Restart server:

```
root@dhcpserver:~# /etc/init.d/isc-dhcp-server restart
```

Logs are in `/var/log/syslog`

### dhcrelay

Make sure it's listening to both intefaces and points to the IP of the `dhcplb`
VM:

```
$ vagrant ssh dhcprelay
vagrant@dhcprelay:~$ sudo su -
root@dhcprelay:~# vi /etc/default/isc-dhcp-relay
# IP of the dhcplb VM
SERVERS="192.168.50.104"
INTERFACES="eth1 eth2"
```

Start the daemon:

```
root@dhcprelay:~# initctl start isc-dhcp-relay
isc-dhcp-relay start/running, process 3071
```

Logs in `/var/log/syslog`.

### dhcplb

```
```

### dhclient

