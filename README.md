# Lagoon-sync

Lagoon-sync is cli tool written in Go that fundamentally provides the functionality to synchronise data between Lagoon environments. Lagoon-sync is part of the [Lagoon cli](https://github.com/amazeeio/lagoon-cli) toolset and works closely with its parent project.

Lagoon-sync offers:
* Sync commands for databases such as `mariadb`, `postgres` and `mongodb`
* Php/node-based framework support such as Drupal, Laravel or Node.js
* Standard file transfer support with `files` syncer
* Has built-in default configuration values for syncing out-the-box
* Provides an easy way to override sync configuration via `.lagoon.yml` or `.lagoon-sync.yml` files
* Offers `--dry-run` flag to see what commands would be executed before running a transfer
* `--no-interaction` can be used to auto-run all processes without prompt - useful for CI/builds
* `config` command shows the configuration of the current environment
* There is a `--show-debug` flag to output more verbose logging for debugging
* Lagoon-sync uses `rsync` for the transfer of data, and will automatically detect and install `rsync` if it is not available on target environments
* Secure cross-platform self-updating with `selfUpdate` command


# Installing

You can run `lagoon-sync` as a single binary by downloading from `https://github.com/uselagoon/lagoon-sync/releases/latest`.

MacOS: `lagoon-sync_*.*.*_darwin_amd64`
Linux (3 variants available): `lagoon-sync_*.*.*_linux_386`
Windows: `lagoon-sync_*.*.*_windows_amd64.exe`

To install via bash:


## macOS (with M1 processors)

    DOWNLOAD_PATH=$(curl -sL "https://api.github.com/repos/uselagoon/lagoon-sync/releases/latest" | grep "browser_download_url" | cut -d \" -f 4 | grep darwin_arm64) && wget -O /usr/local/bin/lagoon-sync $DOWNLOAD_PATH && chmod a+x /usr/local/bin/lagoon-sync

## macOS (with Intel processors)

    DOWNLOAD_PATH=$(curl -sL "https://api.github.com/repos/uselagoon/lagoon-sync/releases/latest" | grep "browser_download_url" | cut -d \" -f 4 | grep darwin_amd64) && wget -O /usr/local/bin/lagoon-sync $DOWNLOAD_PATH && chmod a+x /usr/local/bin/lagoon-sync

## Linux (386)

    DOWNLOAD_PATH=$(curl -sL "https://api.github.com/repos/uselagoon/lagoon-sync/releases/latest" | grep "browser_download_url" | cut -d \" -f 4 | grep linux_386) && wget -O /usr/local/bin/lagoon-sync $DOWNLOAD_PATH && chmod a+x /usr/local/bin/lagoon-sync

## Linux (amd64)

    DOWNLOAD_PATH=$(curl -sL "https://api.github.com/repos/uselagoon/lagoon-sync/releases/latest" | grep "browser_download_url" | cut -d \" -f 4 | grep linux_amd64) && wget -O /usr/local/bin/lagoon-sync $DOWNLOAD_PATH && chmod a+x /usr/local/bin/lagoon-sync

## Linux (arm64)

    DOWNLOAD_PATH=$(curl -sL "https://api.github.com/repos/uselagoon/lagoon-sync/releases/latest" | grep "browser_download_url" | cut -d \" -f 4 | grep linux_arm64) && wget -O /usr/local/bin/lagoon-sync $DOWNLOAD_PATH && chmod a+x /usr/local/bin/lagoon-sync


# Usage

Lagoon-sync has the following core commands:

```
$ lagoon-sync
lagoon-sync is a tool for syncing resources between environments in Lagoon hosted applications. This includes files, databases, and configurations.

Usage:
  lagoon-sync [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  config      Print the config that is being used by lagoon-sync
  help        Help about any command
  selfUpdate  Update this tool to the latest version
  sync        Sync a resource type
  version     Print the version number of lagoon-sync

Flags:
      --config string   config file (default is .lagoon.yaml) (default "./.lagoon.yml")
  -h, --help            help for lagoon-sync
      --show-debug      Shows debug information
  -t, --toggle          Help message for toggle
  -v, --version         version for lagoon-sync

Use "lagoon-sync [command] --help" for more information about a command.
```

## sync

Sync transfers are executed with `$lagoon-sync sync <syncer>` and require at least a syncer type `[mariadb|files|mongodb|postgres|drupalconfig]`, a valid project name `-p` and source environment `-e`. By default, if you do not provide an optional target environment `-t` then `local` is used.

```
lagoon-sync sync

Usage:
  lagoon-sync sync [mariadb|files|mongodb|postgres|etc.] [flags]

Flags:
  -c, --configuration-file string        File containing sync configuration.
      --dry-run                          Don't run the commands, just preview what will be run
  -h, --help                             help for sync
      --no-interaction                   Disallow interaction
  -p, --project-name string              The Lagoon project name of the remote system
  -r, --rsync-args string                Pass through arguments to change the behaviour of rsync (default "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX -r")
  -s, --service-name string              The service name (default is 'cli'
  -e, --source-environment-name string   The Lagoon environment name of the source system
  -i, --ssh-key string                   Specify path to a specific SSH key to use for authentication
  -t, --target-environment-name string   The target environment name (defaults to local)
      --verbose                          Run ssh commands in verbose (useful for debugging)

Global Flags:
      --config string   config file (default is .lagoon.yaml) (default "./.lagoon.yml")
      --show-debug      Shows debug information
```

## config

The `config` command will output all current configuration information it can find on the environment. This is used, for example, to gather prerequisite data which can be used to determine how `lagoon-sync` should proceed with a transfer. For example, when running the tool on a environment that doesn't have rsync, then the syncer will know to install a static copy of rsync on that machine for us. This is because rsync requires that you need to have it available on both environments in order to transfer.

This can be run with:

`$ lagoon-sync config`

## Example syncs

As with all sync commands, if you run into issues you can run `--show-debug` to see extra log information. There is also the `config` command which is useful to see what configuration files are active.

### Mariadb sync from remote source -> local environment
An example sync between a `mariadb` database from a remote source environment to your local instance may go as follows:

Running `$ lagoon-sync sync mariadb -p amazeelabsv4-com -e dev --dry-run` would dry-run a process that takes a database dump, runs a data transfer and then finally syncs the local database with the latest dump.

### Mariadb sync from remote source -> remote target environment
To transfer between remote environments you can pass in a target argument such as:

`$ lagoon-sync sync mariadb -p amazeelabsv4-com -e prod -t dev --dry-run`

### Mariadb sync from remote source to local file (*Dump only*)

It's also possible to simply generate a backup from one of the remote servers by using the options
`--skip-target-cleanup=true`, which doesn't delete temporary transfer files, and `--skip-target-import=true` which
skips actually importing the database locally.

`$ lagoon-sync sync mariadb -p amazeelabsv4-com -e prod -t dev --skip-target-cleanup=true --skip-target-import=true`

You will then see the transfer-resource name listed in the output.

This command would attempt to sync mariadb databases from `prod` to `dev` environments.

## Configuring lagoon-sync

Lagoon-sync configuration can be managed via yaml-formatted configuration files. The paths to these config files can be defined either by the `--config` argument, or by environment variables (`LAGOON_SYNC_PATH` or `LAGOON_SYNC_DEFAULTS_PATH`).

The order of configuration precedence is as follows:

1. `--config` argument (e.g. `lagoon-sync [command] --config ./.custom-lagoon-sync-config.yaml`).
2.  `.lagoon.yaml` files (i.e. in project root, or `lagoon` directory). If an `.lagoon.yml` is available within the project, then this file will be used as the active configuration file by default.
3. `LAGOON_SYNC_PATH` or `LAGOON_SYNC_DEFAULTS_PATH` environment variables.
4. Finally, if no config file can be found the default configuration will be used a safely written to a new '.lagoon.yml`

There are some configuration examples in the `examples` directory of this repo.

To double check which config file is loaded you can also run the `lagoon-sync config` command.

### Example sync config overrides
```
lagoon-sync:
  mariadb:
    config:
      hostname: "${MARIADB_HOST:-mariadb}"
      username: "${MARIADB_USERNAME:-drupal}"
      password: "${MARIADB_PASSWORD:-drupal}"
      port: "${MARIADB_PORT:-3306}"
      database: "${MARIADB_DATABASE:-drupal}"
  files:
    config:
      sync-directory: "/app/web/sites/default/files"
  drupalconfig:
    config:
      syncpath: "./config/sync"
```

# Useful things
## Updating lagoon-sync

It's possible to safely perform a cross-platform update of your lagoon-sync binary by running the `$ lagoon-sync selfUpdate` command. This will look for the latest release, then download the corresponding checksum and signature of the executable on GitHub, and verify its integrity and authenticity before it performs the update. The binary used to perform the update will then replace itself (if successful) to the new version. If an error occurs then the update will roll back to the previous stable version.

```
$ lagoon-sync selfUpdate

Downloading binary from https://github.com/uselagoon/lagoon-sync/releases/download/v0.4.4/lagoon-sync_0.4.4_linux_386
Checksum for linux_386: 61a55bd793d5745b6196ffd5bb87263aba85629f55ee0eaf53c771a0720adefd
Good signature from "amazeeio"
Applying update...
Successfully updated binary at: /usr/bin/lagoon-sync
```

You can check version with `$ lagoon-sync --version`

## Installing binary from script - Drupal example

This example will run a script that will install a Linux lagoon-sync binary and default configuration file for a Drupal project.

```
wget -q -O - https://gist.githubusercontent.com/timclifford/cec9fe3ddf8d0805e4801d132dfce682/raw/a9979ff24290a500f53df09723774216603de6b5/lagoon-sync-drupal-install.sh | bash
```

# Contributing

Setting up locally:

* `make all`                          Installs missing dependencies, runs tests and build locally.
* `make build`                        Compiles binary based on current go env.
* `make local-build-linux`            Compile linix binary.
* `make local-build-darwin`           Compile macOS (darwin) binary.
* `make check-current-tag-version`    Check the current version.
* `make clean`                        Remove all build files and assets.

## Releases

We are using [goreleaser](https://github.com/goreleaser/goreleaser) for the official build, release and publish steps that will be run from a GitHub Action on a pushed tag event.

Locally, we can run `make release-test` to check if our changes will build. If compiling was successful we can commit our changes and then run `make release-[patch|minor|major]` to tag with next release number and it will push up to GitHub. A GitHub action will then be triggered which will publish the official release using goreleaser.

Prior to that, we can locally test our release to ensure that it will successfully build with `make release-test`. If compiling was successful we can commit our changes and then run `make release-[patch|minor|major]` to tag with next release number and it will push up to GitHub. A GitHub action will then be triggered which will publish the official release using goreleaser.
