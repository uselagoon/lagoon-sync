# Lagoon-sync

Lagoon-sync is cli tool written in Go that fundamentally provides the functionality to synchronise data between Lagoon environments. Lagoon-sync is part of the [Lagoon cli](https://github.com/amazeeio/lagoon-cli) toolset and, indeed, works closely with its parent project.

Lagoon-sync offers:
* Sync commands for databases such as `mariadb`, `postgres` and `mongodb`
* Any php/node-based framework support such as Drupal, Laravel or Node.js
* Standard file transfer support with `files` syncer
* Has built in default configuration values for syncing out-the-box
* Provides an easy way to override sync configuration via `.lagoon.yml` or `.lagoon-sync.yml` files
* Offers `--dry-run` flag to see what commands would be executed before running a transfer
* `--no-interaction` can be used to auto-run all processes without prompt - useful for CI/builds 
* `config` command shows the configuration of the current environment
* There is a `--show-debug` flag to output more verbose logging for debugging
* Lagoon-sync uses `rsync` for the transfer of data and will automatically detect and install `rsync` if it is not available on target environments
* Secure cross-platform self-updating with `selfUpdate` command


# Installing

You can run `lagoon-sync` as a single binary by downloading from `https://github.com/amazeeio/lagoon-sync/releases/`.
* MacOS: `lagoon-sync_*.*.*_darwin_amd64`
* Linux: `lagoon-sync_*.*.*_linux_386`
* Windows: `lagoon-sync_*.*.*_windows_amd64.exe`

To install via bash:

```
wget -O /usr/bin/lagoon-sync https://github.com/amazeeio/lagoon-sync/releases/download/v0.4.4/lagoon-sync_0.4.4_linux_386 && chmod +x /usr/bin/lagoon-sync
```

# Usage

Lagoon-sync has the following core commands:

```
$ lagoon-sync
lagoon-sync is a tool for syncing resources between environments in Lagoon hosted applications. 
This includes files, databases, and configurations.

Usage:
  lagoon-sync [command]

Available Commands:
  config      Print the config that is being used by lagoon-sync
  help        Help about any command
  selfUpdate  Update this tool to the latest version
  sync        Sync a resource type
  version     Print the version number of lagoon-sync

Flags:
      --config string   config file (default is .lagoon.yaml)
  -h, --help            help for lagoon-sync
      --show-debug      Shows debug information
  -t, --toggle          Help message for toggle
  -v, --version         version for lagoon-sync

Use "lagoon-sync [command] --help" for more information about a command.
```

## sync

Sync transfers are executed with `$lagoon-sync sync <syncer>` and requires at least a syncer type `[mariadb|files|mongodb|postgres|drupalconfig]`, a valid project name `-p` and source environment `-e`. By default, if you do not provide an optional target environment `-t` then `local` is used.

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
  -s, --service-name string              The service name (default is 'cli'
  -e, --source-environment-name string   The Lagoon environment name of the source system
  -t, --target-environment-name string   The target environment name (defaults to local)
      --verbose                          Run ssh commands in verbose (useful for debugging)

Global Flags:
      --config string   config file (default is .lagoon.yaml) (default "./.lagoon.yml")
      --show-debug      Shows debug information
```

## config

The `config` command will output all current configuation information it can find on the environment. This is used for example to gather prerequisite data which can be used to determine how `lagoon-sync` should proceed with a transfer. For example, when running the tool on a environment that doesn't have rsync, then the syncer will know to install a static copy of rsync on that machine for us. This is because rsync requires that you need to have it available on both environments in order to transfer.

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

This command would attempt to sync mariadb databases from `prod` to `dev` environments.

## Configuring lagoon-sync

It is possible to configure the data consumed by lagoon-sync via adding a `lagoon-sync:` key to an existing `.lagoon.yml` file or via a configuration file such as (`.lagoon-sync`). See the `.lagoon.yml` and `.example-lagoon-sync` in the root of this repo for examples.

If a `.lagoon.yml` is available within the project, then this file will be used as the active configuration file to attempt to gather configuration data from by default.

Next, if a `.lagoon-sync` or `.lagoon-sync-defaults` file is added to the `/lagoon` directory then these will be used as the active configuration file. Running the sync with `--show-debug` you are able to see the configuration that will be run prior to running the process:

```
$ lagoon-sync sync mariadb -p mysite-com -e dev --show-debug

2021/01/22 11:34:10 (DEBUG) Using config file: /lagoon/.lagoon-sync
2021/01/22 11:34:10 (DEBUG) Config that will be used for sync:
 {
  "Config": {
    "DbHostname": "$MARIADB_HOST",
    "DbUsername": "$MARIADB_USERNAME",
    "DbPassword": "$MARIADB_PASSWORD",
    "DbPort": "$MARIADB_PORT",
    "DbDatabase": "$MARIADB_DATABASE",
    ...
```

To recap, the configuration files that can be used by default, in order of priority when available are:
* /lagoon/.lagoon-sync-defaults
* /lagoon/.lagoon-sync
* .lagoon.yml

### Custom configuration files
If you don't want your configuration file inside `/lagoon` and want to give it another name then you can define a custom file and tell sync to use that by providing the file path. This can be done with `--config` flag such as:

```
$ lagoon-sync sync mariadb -p mysite-com -e dev --config=/app/.lagoon-sync --show-debug

2021/01/22 11:43:50 (DEBUG) Using config file: /app/.lagoon-sync
```

You can also use an environment variable to set the config sync path with either `LAGOON_SYNC_PATH` or `LAGOON_SYNC_DEFAULTS_PATH`.

```
$ LAGOON_SYNC_PATH=/app/.lagoon-sync lagoon-sync sync mariadb -p mysite-com -e dev --show-debug

2021/01/22 11:46:42 (DEBUG) LAGOON_SYNC_PATH env var found: /app/.lagoon-sync
2021/01/22 11:46:42 (DEBUG) Using config file: /app/.lagoon-sync
```

To double check which config file is active you can also run the `$ lagoon-sync config` command.

### Example sync config overrides
```
lagoon-sync:
  mariadb:
    config:
      hostname: "$MARIADB_HOST"
      username: "$MARIADB_USERNAME"
      password: "$MARIADB_PASSWORD"
      port: "$MARIADB_PORT"
      database: "$MARIADB_DATABASE"
  files:
    config:
      sync-directory: "/app/web/sites/default/files"
  drupalconfig:
    config:
      syncpath: "./config/sync"
```

# Useful things
## Updating lagoon-sync

It's possible to safely perform a cross-platform update of your lagoon-sync binary by running the `$ lagoon-sync selfUpdate` command. This will look for the latest release, then download the corresponding checksum and signature of the executable on GitHub, and verify its interity and authenticity before it performs the update. The binary used to perform the update will then replace itself (if succcssful) to the new version. If an error occurs then the update will roll-back to the previous stable version.

```
$ lagoon-sync selfUpdate

Downloading binary from https://github.com/amazeeio/lagoon-sync/releases/download/v0.4.4/lagoon-sync_0.4.4_linux_386
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

Prior to that, we can locally test our release to ensure that it will successfully build with `make release-test`. If compiling was successful we can commit our changes and then run `make release-[patch|minor|major]` to tag with next release number and it will push up to GitHub. A GitHub action will then be triggered which will publish the official release using goreleaser.
