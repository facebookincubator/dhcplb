# Copyright (c) 2016-present, Facebook, Inc.
# All rights reserved.
#
# This source code is licensed under the BSD-style license found in the
# LICENSE file in the root directory of this source tree. An additional grant
# of patent rights can be found in the PATENTS file in the same directory.

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
