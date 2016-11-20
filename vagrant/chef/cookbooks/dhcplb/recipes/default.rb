node.default['go']['version'] = '1.7'
node.default['go']['packages'] = ['github.com/facebookincubator/dhcplb']

directory '/home/vagrant/go' do
  owner 'vagrant'
  group 'vagrant'
  recursive true
end

cookbook_file '/home/vagrant/dhcplb.config.json' do
  source 'dhcplb.config.json'
end

template '/home/vagrant/dhcp-servers-v4.cfg' do
  source 'dhcp-servers-v4.cfg.erb'
  # notifies :restart, 'service[isc-dhcp-server]'
end

# TODO: configuration file: 1) json + 2) list of servers
#
# Configure service via https://github.com/poise/poise-service
