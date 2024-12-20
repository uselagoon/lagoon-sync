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

## Usage

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