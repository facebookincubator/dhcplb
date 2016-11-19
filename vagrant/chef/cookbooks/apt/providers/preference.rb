#
# Cookbook Name:: apt
# Provider:: preference
#
# Copyright 2010-2016, Chef Software, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

use_inline_resources

def whyrun_supported?
  true
end

# Build preferences.d file contents
def build_pref(package_name, pin, pin_priority)
  "Package: #{package_name}\nPin: #{pin}\nPin-Priority: #{pin_priority}\n"
end

def safe_name(name)
  name.tr('.', '_').gsub('*', 'wildcard')
end

action :add do
  preference = build_pref(
    new_resource.glob || new_resource.package_name,
    new_resource.pin,
    new_resource.pin_priority
  )

  directory '/etc/apt/preferences.d' do
    owner 'root'
    group 'root'
    mode '0755'
    recursive true
    action :create
  end

  name = safe_name(new_resource.name)

  file "/etc/apt/preferences.d/#{new_resource.name}.pref" do
    action :delete
    if ::File.exist?("/etc/apt/preferences.d/#{new_resource.name}.pref")
      Chef::Log.warn "Replacing #{new_resource.name}.pref with #{name}.pref in /etc/apt/preferences.d/"
    end
    only_if { name != new_resource.name }
  end

  file "/etc/apt/preferences.d/#{new_resource.name}" do
    action :delete
    if ::File.exist?("/etc/apt/preferences.d/#{new_resource.name}")
      Chef::Log.warn "Replacing #{new_resource.name} with #{new_resource.name}.pref in /etc/apt/preferences.d/"
    end
  end

  file "/etc/apt/preferences.d/#{name}.pref" do
    owner 'root'
    group 'root'
    mode '0644'
    content preference
    action :create
  end
end

action :remove do
  name = safe_name(new_resource.name)
  if ::File.exist?("/etc/apt/preferences.d/#{name}.pref")
    Chef::Log.info "Un-pinning #{name} from /etc/apt/preferences.d/"
    file "/etc/apt/preferences.d/#{name}.pref" do
      action :delete
    end
  end
end
