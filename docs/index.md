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


## Commands

* `lagoon-sync sync [mariadb|files|mongodb|postgres] [flags]` - Sync resources between remote and local environments.