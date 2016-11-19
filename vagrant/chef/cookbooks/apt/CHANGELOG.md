# apt Cookbook CHANGELOG

This file is used to list changes made in each version of the apt cookbook.

## 5.0.0 (2016-10-14)

- Remove search logic from the cacher client cookbook and rework attribute structure. See the attributes file and readme for new structure. Determining what servers to cache against is better handled in a wrapper cookbook where you can define the exact search syntax yourself
- Corrected readme examples for the cacher client setup
- Depend on the latest compat_resource
- Define matchers for ChefSpec
- Testing updates to better test the various recipes and providers in the cookbook on Travis

## 4.0.2 (2016-08-13)

- The cookbook requires Chef 12.1+ not 12.0\. Update docs
- Test on Chef 12.1 to ensure compatibility
- Restore compatibility with Chef < 12.4

## 4.0.1 (2016-06-21)

- Fix bug that prevented adding the cookbook to non Debian/Ubuntu nodes without chef run failures

## 4.0.0 (2016-06-02)

This cookbook now requires Chef 12\. If you require Chef 11 compatibility you will need to pin to the 3.X cookbook version

- The apt-get update logic in the default recipe has been converted to apt_update custom resource and compat_resource cookbook has been added for backwards compatibility with all Chef 12.X releases. In addition this resource is now included in core chef-client and the cookbook will use the built-in resource if available
- Added support for the unattended-upgrade RandomSleep configuration feature
- Added support for the unattended-upgrade Unattended-Upgrade::Origins-Pattern configuration feature
- Added Chefspec matchers for apt_update
- Fixed apt_repository documentation to correctly reflect the deb_src property

## 3.0.0 (2016-03-01)

- Removed Chef 10 compatibility code. This cookbook requires Chef 11 or greater now
- The default recipe will no longer create /etc/apt/ and other directories on non-Debian based systems
- Updated the autoremove command in the default recipe to run in non-interactive mode
- Added CentOS 7 to Test Kitchenwith tests to ensure we don't create any files on RHEL or other non-Debian hosts
- Updated Chefspec to 4.X format
- Properly mock the existence of apt for the Chefspec runs so they don't just skip over the resources
- Fixed lwrp test kitchen tests to pass
- Resolved or disabled all Rubocop warnings
- Enabled testing in Travis CI
- Removed Apt Cacher NG support for Ubuntu 10.04 and Debian 6.X as they are both deprecated
- Fixed + signs in packages names with the preference LWRP being rejected

## v2.9.2

- # 168 Adding guard to package resource.

## v2.9.1

- Adding package apt-transport-https to default.rb

## v2.9.0

- Add `sensitive` flag for apt_repositories
- Enable installation of recommended or suggested packages
- Tidy up `apt-get update` logic
- Fixing not_if guard on ruby_block[validate-key #{key}]

## v2.8.2 (2015-08-24)

- Fix removal of apt_preferences

## v2.8.1 (2015-08-18)

- Handle keyservers as URLs and bare hostnames

## v2.8.0 (2015-08-18)

- Access keyservers on port 80
- Adds key_proxy as LWRP attribute for apt_repository
- Fix wildcard glob preferences files
- Fix text output verification for non en_US locales
- Quote repo URLs to deal with spaces

## v2.7.0 (2015-03-23)

- Support Debian 8.0
- Filename verification for LWRPs
- Support SSL enabled apt repositories

## v2.6.1 (2014-12-29)

- Remove old preference files without .pref extension from previous versions

## v2.6.0 (2014-09-09)

- Always update on first run - check
- Adding ppa support for apt_repository

## v2.5.3 (2014-08-14)

- # 87 - Improve default settings, account for non-linux platforms

## v2.5.2 (2014-08-14)

- Fully restore 2.3.10 behaviour

## v2.5.1 (2014-08-14)

- fix breakage introduced in apt 2.5.0

## v2.5.0 (2014-08-12)

- Add unattended-upgrades recipe
- Only update the cache for the created repository
- Added ChefSpec matchers and default_action for resources
- Avoid cloning resource attributes
- Minor documentation updates

## v2.4.0 (2014-05-15)

- [COOK-4534]: Add option to update apt cache at compile time

## v2.3.10 (2014-04-23)

- [COOK-4512] Bugfix: Use empty PATH if PATH is nil

## v2.3.8 (2014-02-14)

### Bug

- **[COOK-4287](https://tickets.opscode.com/browse/COOK-4287)** - Cleanup the Kitchen

## v2.3.6

- [COOK-4154] - Add chefspec matchers.rb file to apt cookbook
- [COOK-4102] - Only index created repository

## v2.3.6

- [COOK-4154] - Add chefspec matchers.rb file to apt cookbook
- [COOK-4102] - Only index created repository

## v2.3.4

No change. Version bump for toolchain sanity

## v2.3.2

- [COOK-3905] apt-get-update-periodic: configuration for the update period
- Updating style for rubocops
- Updating test-kitchen harness

## v2.3.0

### Bug

- **[COOK-3812](https://tickets.opscode.com/browse/COOK-3812)** - Add a way to bypass the apt existence check

### Improvement

- **[COOK-3567](https://tickets.opscode.com/browse/COOK-3567)** - Allow users to bypass apt-cache via attributes

## v2.2.1

### Improvement

- **[COOK-664](https://tickets.opscode.com/browse/COOK-664)** - Check platform before running apt-specific commands

## v2.2.0

### Bug

- **[COOK-3707](https://tickets.opscode.com/browse/COOK-3707)** - multiple nics confuse apt::cacher-client

## v2.1.2

### Improvement

- **[COOK-3551](https://tickets.opscode.com/browse/COOK-3551)** - Allow user to set up a trusted APT repository

## v2.1.1

### Bug

- **[COOK-1856](https://tickets.opscode.com/browse/COOK-1856)** - Match GPG keys without case sensitivity

## v2.1.0

- [COOK-3426]: cacher-ng fails with restrict_environment set to true
- [COOK-2859]: cacher-client executes out of order
- [COOK-3052]: Long GPG keys are downloaded on every run
- [COOK-1856]: apt cookbook should match keys without case sensitivity
- [COOK-3255]: Attribute name incorrect in README
- [COOK-3225]: Call use_inline_resources only if defined
- [COOK-3386]: Cache dir for apt-cacher-ng
- [COOK-3291]: apt_repository: enable usage of a keyserver on port 80
- Greatly expanded test coverage with ChefSpec and Test-Kitchen

## v2.0.0

### Bug

- [COOK-2258]: apt: LWRP results in error under why-run mode in apt 1.9.0 cookbook

## v1.10.0

### Improvement

- [COOK-2885]: Improvements for apt cache server search

### Bug

- [COOK-2441]: Apt recipe broken in new chef version
- [COOK-2660]: Create Debian 6.0 "squeeze" specific template for
- apt-cacher-ng

## v1.9.2

- [COOK-2631] - Create Ubuntu 10.04 specific template for apt-cacher-ng

## v1.9.0

- [COOK-2185] - Proxy for apt-key
- [COOK-2338] - Support pinning by glob() or regexp

## v1.8.4

- [COOK-2171] - Update README to clarify required Chef version: 10.18.0
- or higher.

## v1.8.2

- [COOK-2112] - need [] around "arch" in sources.list entries
- [COOK-2171] - fixes a regression in the notification

## v1.8.0

- [COOK-2143] - Allow for a custom cacher-ng port
- [COOK-2171] - On `apt_repository.run_action(:add)` the source file
- is not created.
- [COOK-2184] - apt::cacher-ng, use `cacher_port` attribute in
- acng.conf

## v1.7.0

- [COOK-2082] - add "arch" parameter to apt_repository LWRP

## v1.6.0

- [COOK-1893] - `apt_preference` use "`package_name`" resource instead of "name"
- [COOK-1894] - change filename for sources.list.d files
- [COOK-1914] - Wrong dir permissions for /etc/apt/preferences.d/
- [COOK-1942] - README.md has wrong name for the keyserver attribute
- [COOK-2019] - create 01proxy before any other apt-get updates get executed

## v1.5.2

- [COOK-1682] - use template instead of file resource in apt::cacher-client
- [COOK-1875] - cacher-client should be Environment-aware

## V1.5.0

- [COOK-1500] - Avoid triggering apt-get update
- [COOK-1548] - Add execute commands for autoclean and autoremove
- [COOK-1591] - Setting up the apt proxy should leave https
- connections direct
- [COOK-1596] - execute[apt-get-update-periodic] never runs
- [COOK-1762] - create /etc/apt/preferences.d directory
- [COOK-1776] - apt key check isn't idempotent

## v1.4.8

- Adds test-kitchen support
- [COOK-1435] - repository lwrp is not idempotent with http key

## v1.4.6

- [COOK-1530] - apt_repository isn't aware of update-success-stamp
- file (also reverts COOK-1382 patch).

## v1.4.4

- [COOK-1229] - Allow cacher IP to be set manually in non-Chef Solo
- environments
- [COOK-1530] - Immediately update apt-cache when sources.list file is dropped off

## v1.4.2

- [COOK-1155] - LWRP for apt pinning

## v1.4.0

- [COOK-889] - overwrite existing repo source files
- [COOK-921] - optionally use cookbook_file or remote_file for key
- [COOK-1032] - fixes problem with apt repository key installation
