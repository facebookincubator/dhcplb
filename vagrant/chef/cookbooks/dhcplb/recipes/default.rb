node.default['go']['version'] = '1.7'
node.default['go']['packages'] = ['github.com/facebookincubator/dhcplb']

poise_service 'dhcplb' do
  command '/opt/go/bin/dhcplb -version 4 -config /home/vagrant/dhcplb.config.json'
end

directory '/home/vagrant/go' do
  owner 'vagrant'
  group 'vagrant'
  recursive true
end

cookbook_file '/home/vagrant/dhcplb.config.json' do
  source 'dhcplb.config.json'
  notifies :restart, 'service[dhcplb]'
end

template '/home/vagrant/dhcp-servers-v4.cfg' do
  source 'dhcp-servers-v4.cfg.erb'
  # dhcplb will auto load files that change. no need to notify.
end

# Configure service via https://github.com/poise/poise-service
