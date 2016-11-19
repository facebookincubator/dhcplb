#
# Cookbook Name:: apt
# Resource:: repository
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

actions :add, :remove
default_action :add

state_attrs :arch,
            :cache_rebuild,
            :components,
            :cookbook,
            :deb_src,
            :distribution,
            :key,
            :keyserver,
            :key_proxy,
            :repo_name,
            :trusted,
            :uri,
            :sensitive

# name of the repo, used for source.list filename
attribute :repo_name, kind_of: String, name_attribute: true, regex: [/^([a-z]|[A-Z]|[0-9]|_|-|\.)+$/]
attribute :uri, kind_of: String
attribute :distribution, kind_of: String
attribute :components, kind_of: Array, default: []
attribute :arch, kind_of: String, default: nil
attribute :trusted, kind_of: [TrueClass, FalseClass], default: false
# whether or not to add the repository as a source repo as well
attribute :deb_src, default: false
attribute :keyserver, kind_of: String, default: nil
attribute :key, kind_of: String, default: nil
attribute :key_proxy, kind_of: String, default: node['apt']['key_proxy']
attribute :cookbook, kind_of: String, default: nil
# trigger cache rebuild
# If not you can trigger in the recipe itself after checking the status of resource.updated{_by_last_action}?
attribute :cache_rebuild, kind_of: [TrueClass, FalseClass], default: true
# Hide content of the source file, don't show output for commands being run, etc.
attribute :sensitive, kind_of: [TrueClass, FalseClass], default: false
