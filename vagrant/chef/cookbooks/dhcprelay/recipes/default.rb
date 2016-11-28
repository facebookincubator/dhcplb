package 'isc-dhcp-relay' do
  action :install
end

template '/etc/default/isc-dhcp-relay' do
  source 'etc_default_isc-dhcp-relay.erb'
  owner 'root'
  group 'root'
  mode '0644'
  notifies :restart, 'service[isc-dhcp-relay]'
end

service 'isc-dhcp-relay' do
    action [:enable, :start]
end
