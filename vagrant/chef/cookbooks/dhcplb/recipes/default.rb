node.default['go']['version'] = '1.7'
node.default['go']['packages'] = ['github.com/facebookincubator/dhcplb']

directory '/home/vagrant/go' do
  owner 'vagrant'
  group 'vagrant'
  recursive true
end

# TODO: configuration file: 1) json + 2) list of servers
