
# config

The `config` command will output all current configuration information it can find on the environment. This is used, for example, to gather prerequisite data which can be used to determine how `lagoon-sync` should proceed with a transfer. For example, when running the tool on a environment that doesn't have rsync, then the syncer will know to install a static copy of rsync on that machine for us. This is because rsync requires that you need to have it available on both environments in order to transfer.

This can be run with:

`$ lagoon-sync config`

# Configuring lagoon-sync

Lagoon-sync configuration can be managed via yaml-formatted configuration files. The paths to these config files can be defined either by the `--config` argument, or by environment variables (`LAGOON_SYNC_PATH` or `LAGOON_SYNC_DEFAULTS_PATH`).

The order of configuration precedence is as follows:

1. `--config` argument (e.g. `lagoon-sync [command] --config ./.custom-lagoon-sync-config.yaml`).
2. `.lagoon-sync.yml` typically contains a separate lagoon-sync configuration. Although this can, if required, be merged into the `.lagoon.yml` file
3. `.lagoon.yaml` files (i.e. in project root, or `lagoon` directory). If an `.lagoon.yml` is available within the project, then this file will be used as the active configuration file by default.
4. `LAGOON_SYNC_PATH` or `LAGOON_SYNC_DEFAULTS_PATH` environment variables.
5. Finally, if no config file can be found the default configuration will be used a safely written to a new '.lagoon.yml`

There are some configuration examples in the `examples` directory of this repo.

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

To recap, the configuration files that can be used by default, in order of priority when available are:
* /lagoon/.lagoon-sync-defaults
* /lagoon/.lagoon-sync
* .lagoon.yml


## Custom configuration files
If you don't want your configuration file inside `/lagoon` and want to give it another name then you can define a custom file and tell sync to use that by providing the file path. This can be done with `--config` flag such as:Config files that can be used in order of priority:
- .lagoon-sync-defaults _(no yaml ext neeeded)_
- .lagoon-sync _(no yaml ext neeeded)_
- .lagoon.yml _Main config file - path can be given as an argument with `--config`, default is `.lagoon.yml`_
  å
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
