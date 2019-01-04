# Copyright (c) Facebook, Inc. and its affiliates.
#
# This source code is licensed under the MIT license found in the
# LICENSE file in the root directory of this source tree.

apt_repository 'kea-repo' do
  uri          'ppa:xdeccardx/isc-kea'
end

# this contains perfdhcp utility
package 'kea-admin' do
  action :install
end
