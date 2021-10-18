# Lagoon-sync

Lagoon-sync is cli tool written in Go that fundamentally provides the functionality to synchronise data between Lagoon environments. Lagoon-sync is part of the Lagoon cli toolset and, indeed, works closely with its parent project.

Lagoon-sync offers:
- Sync commands for databases such as `mariadb`, `postgres` and `mongodb`
- Any php/node-based framework support such as Drupal, Laravel or Node.js
- Standard file transfer support with `files` syncer
- Has built in default configuration values for syncing out-the-box
- Provides an easy way to override sync configuration via `.lagoon.yml` or `.lagoon-sync.yml` files
- Offers `--dry-run` flag to see what commands would be executed before running a transfer
- `config` command shows configuration on current environment
- There is a `--show-debug` flag to output more verbose logging for debugging
- Lagoon-sync uses `rsync` for the transfer of data and will automatically detect and install `rsync` if it is not available on target environments
- Self-updatingg with `selfUpdate` command


# Installing

You can run `lagoon-sync` as a single binary by downloading from `https://github.com/amazeeio/lagoon-sync/releases/latest`.

MacOS: `lagoon-sync_*.*.*_darwin_amd64`
Linux (3 variants available): `lagoon-sync_*.*.*_linux_386`
Windows: `lagoon-sync_*.*.*_windows_amd64.exe`

To install via bash:

```
# macOS
curl https://github.com/amazeeio/lagoon-sync/releases/download/v0.4.7/lagoon-sync_0.4.7_darwin_amd64 -Lo /usr/local/bin/lagoon-sync && chmod a+x $_

# Linux
curl https://github.com/amazeeio/lagoon-sync/releases/download/v0.4.7/lagoon-sync_0.4.7_linux_386 -Lo /usr/bin/lagoon-sync && chmod +x $_
```



# Usage

Lagoon-sync has the following core commands:

```
$ lagoon-sync
lagoon-sync is a tool for syncing resources between environments in Lagoon hosted applications. This includes files, databases, and configurations.

Usage:
  lagoon-sync [command]

Available Commands:
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

Sync transfers are ran with `sync` and requires at least a syncer type `[mariadb|files|mongodb|postgres|etc.]`, a valid project name `-p` and source environment `-e`. By default, if you do not provide an optional target environment `-t` then `local` is used.


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

The `config` command will output all current cconfiguation information it can find on the environment. This is used for example to gather prerequisite data which can be used to determine how `lagoon-sync` should proceed with a transfer. For example, when running the tool on a environment that doesn't have rsync, then the syncer will know to install a copy of rsync on that machine for us. This is because rsync requires that you need to have it available on both locations in order to transfer.

This can be ran with:

`lagoon-sync config`

## Example syncs

As with all sync commands, if you run into issues you can run `--show-debug` to see extra log information. There is also the `config` command which is useful to see what configuration files are active.

### Mariadb sync from remote source -> local environment
An example sync between a `mariadb` database from a remote source environment to your local instance may go as follows:

Running `lagoon-sync sync mariadb -p amazeelabsv4-com -e dev --dry-run` would dry-run a process that takes a database dump, runs a data transfer and then finally syncs the local database with the latest dump.

### Mariadb sync from remote source -> remote target environment
To transfer between remote environments you can pass in a target argument such as:

`lagoon-sync sync mariadb -p amazeelabsv4-com -e prod -t dev --dry-run`

This command would attempt to sync mariadb databases from `prod` to `dev` environments.

## Configuring lagoon-sync

It is possible to configure the data consumed by lagoon-sync via adding `lagoon-sync` to an existing `.lagoon.yml` file or via a configuration file such as (`.lagoon-sync`)


Config files that can be used in order of priority:
- .lagoon-sync-defaults _(no yaml ext neeeded)_
- .lagoon-sync _(no yaml ext neeeded)_
- .lagoon.yml _Main config file - path can be given as an argument with `--config`, default is `.lagoon.yml`_

If either `LAGOON_SYNC_PATH` or `LAGOON_SYNC_DEFAULTS_PATH` env vars are set then it will use those paths instead of the main config file - e.g.

```export LAGOON_SYNC_DEFAULTS_PATH="/lagoon/.lagoon-sync-defaults"```
```export LAGOON_SYNC_PATH="/lagoon/.lagoon-sync"```

To see which config file is active and other configuration settings you can run the `config` command.

### Example sync config overrides
```
lagoon-sync:
  postgres:
    config:
      hostname: "$POSTGRES_HOST"
      username: "$POSTGRES_USERNAME"
      password: "$POSTGRES_PASSWORD"
      port: "5432"
      database: "$POSTGRES_DATABASE"
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

It's possible to safely update your lagoon-sync binrary by running the `selfUpdate` command.

```
$ lagoon-sync selfUpdate

Downloading binary from https://github.com/amazeeio/lagoon-sync/releases/download/v0.4.4/lagoon-sync_0.4.4_linux_386
Checksum for linux_386: 61a55bd793d5745b6196ffd5bb87263aba85629f55ee0eaf53c771a0720adefd
Good signature from "amazeeio"
Applying update...
Successfully updated binary at: /usr/bin/lagoon-sync
```

You can check version with `lagoon-sync --version`

## Installing binary from script - Drupal example

Runs a script that will install a linux lagoon-sync binary and config file for a Drupal project.

```
wget -q -O - https://gist.githubusercontent.com/timclifford/cec9fe3ddf8d0805e4801d132dfce682/raw/a9979ff24290a500f53df09723774216603de6b5/lagoon-sync-drupal-install.sh | bash
```

# Contributing

Setting up locally:

`make all`       Installs missing dependencies, runs tests and build locally.
`make build`     Compiles binary based on current go env.
`make clean`     Remove all build files and assets.
## Releases

We are using goreleaser for the official build, release and publish steps that will be ran from a github action on a pushed tag.

Locally, we can run `make release-test` to check if our changes will build. If compiling was successful we can commit our changes and then run `make release-[patch|minor|major]` to tag with next release number and it will push up to GitHub. A GitHub action will then be triggered which will publish the official release using goreleaser.
