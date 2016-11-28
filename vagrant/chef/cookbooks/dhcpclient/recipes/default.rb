apt_repository 'kea-repo' do
  uri          'ppa:xdeccardx/isc-kea'
end

# this contains perfdhcp utility
package 'kea-admin' do
  action :install
end
