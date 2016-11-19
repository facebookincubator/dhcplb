unless defined? Chef::Resource::AptUpdate
  require 'chef_compat/copied_from_chef/chef/dsl/declare_resource'
  require 'chef/mixin/shell_out'

  class AptUpdate < ChefCompat::Resource
    include ChefCompat::CopiedFromChef::Chef::DSL::DeclareResource
    include Chef::Mixin::ShellOut

    resource_name :apt_update

    provides :apt_update, os: 'linux'
    property :frequency, Integer, default: 86_400

    default_action :periodic
    allowed_actions :update, :periodic

    APT_CONF_DIR = '/etc/apt/apt.conf.d'.freeze
    STAMP_DIR = '/var/lib/apt/periodic'.freeze

    action :periodic do
      unless apt_up_to_date? # ~FC023
        converge_by 'update new lists of packages' do
          do_update
        end
      end
    end

    action :update do
      converge_by 'force update new lists of packages' do
        do_update
      end
    end

    # Determines whether we need to run `apt-get update`
    #
    # @return [Boolean]
    def apt_up_to_date?
      ::File.exist?("#{STAMP_DIR}/update-success-stamp") &&
        ::File.mtime("#{STAMP_DIR}/update-success-stamp") > Time.now - new_resource.frequency
    end

    def do_update
      [STAMP_DIR, APT_CONF_DIR].each do |d|
        build_resource(:directory, d, caller[0]) do
          recursive true
        end.run_action(:create)
      end

      build_resource(:file, "#{APT_CONF_DIR}/15update-stamp", caller[0]) do
        content "APT::Update::Post-Invoke-Success {\"touch #{STAMP_DIR}/update-success-stamp 2>/dev/null || true\";};"
      end.run_action(:create_if_missing)

      shell_out!('apt-get -q update')
    end
  end
end
