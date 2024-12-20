# Lagoon-sync

Lagoon-sync is cli tool written in Go that fundamentally provides the functionality to synchronise data between Lagoon environments. Lagoon-sync is part of the [Lagoon cli](https://github.com/amazeeio/lagoon-cli) toolset and works closely with its parent project.

Lagoon-sync offers:
* Sync commands for databases such as `mariadb`, `postgres` and `mongodb`
* Standard file transfer support with `files` syncer
* Has built-in default configuration values for syncing out-the-box
* Provides an easy way to override sync configuration via `.lagoon-sync.yml` files
* Offers `--dry-run` flag to see what commands would be executed before running a transfer
* `--no-interaction` can be used to auto-run all processes without prompt - useful for CI/builds
* `config` command shows the configuration of the current environment
* There is a `--show-debug` flag to output more verbose logging for debugging
* Secure cross-platform self-updating with `selfUpdate` command


# Getting started with lagoon-sync

Here we'll describe the typical use case for lagoon-sync. While it's able to do quite a number of things, we're going to 
speak about the standard use case - that is, how do we use lagoon-sync to sync our databases and our files between environments.

We'll focus on a very simple example, setting up `lagoon-sync` for a Laravel project.

This tutorial assumes that you've already [lagoonized](https://docs.lagoon.sh/lagoonizing/) your project, and that you have a `docker-compose.yml` file that
describes the services that you're going to need.

## Where does `lagoon-sync` actually run?

This is a common question we get, because it can be kind of confusing. Where exactly are you supposed to run `lagoon-sync`?

Well, it will always run _inside a container_ - it's not a tool like the lagoon cli that just runs from anywhere.
`lagoon-sync` is essentially a wrapper around commands like `mysqldump`, `rsync`, `mongodump`, etc.
In fact, everything that `lagoon-sync` does, you could do manually if you `ssh`ed into your running containers and typed out the various commands.

There is no special, secret sauce. It's more like a collection of neat `bash` scripts than anything else. And so, like with `bash` scripts,
it needs to run in the actual containers.

This means that `lagoon-sync` needs to be available insider your container - typically, we find the easiest way of doing this is
including it in your `cli` dockerfile, if you have one.


## .lagoon-sync.yml

You can run `lagoon-sync` in a myriad ways. But here is the simplest, most straight forward, that should work in most cases.
We encourage this pattern.

You can add a `.lagoon-sync.yml` file to the root of your application's source code, alongside your `.lagoon.yml` file.

This `.lagoon-sync.yml` will describe all of the syncs that you might want `lagoon-sync` to do. Each option for syncing will
appear as a separate item under the `lagoon-sync:` key.

Let's look at an [example](./examples/tutorial/.lagoon-sync.yml):

```
lagoon-sync:
  mariadb:
    type: mariadb
    config:
      hostname: "${MARIADB_HOST:-mariadb}"
      username: "${MARIADB_USERNAME:-lagoon}"
      password: "${MARIADB_PASSWORD:-lagoon}"
      port:     "${MARIADB_PORT:-3306}"
      database: "${MARIADB_DATABASE:-lagoon}"
  cli:
    type: files
    config:
      sync-directory: "/app/storage/"
```

This lagoon sync config above describes two synchers - `mariadb` for the database, and `cli` for the files.
With this in my `.lagoon-sync.yml`, inside my `cli` container, I can run the command
`lagoon-sync sync cli -p myprojectname -e sourceenvironment` and `lagoon-sync` will rsync all the files in `sourceenvironment`'s
`/app/storage/` directory into my local environment.

The same will be the case, except it will sync the database, if I ran `lagoon-sync sync mariadb -p myprojectname -e sourceenvironment`.

Note, that in the example above, `mariadb` and `cli` are simply aliases, we could have renamed them to `mydatabase` and `filestorage` like so:
```
lagoon-sync:
  mydatabase:
    type: mariadb
    config:
<..snip../>
  filestorage:
    type: files
    config:
      sync-directory: "/app/storage/"
```

and run the syncs with `lagoon-sync sync mydatabase -p mypr...` and `lagoon-sync sync filestorage -p mypr...`
The nested keys `cli` and `mariadb` are simply names - it's the `type:` key that tells `lagoon-sync` what it's actually syncing.

You can define as many of these synchers as you need - if you have multiple databases, for instance, or, more likely, if you
have multiple files/directories you'd like to sync separately.

## How should I be generating a `.lagoon-sync.yml`?

This is the part of the process that seems to trip most people up, so we've made it fairly simple.

If you'd like to generate a `.lagoon-sync.yml`, you can use `lagoon-sync`'s built in functions `generate` and `interactive-config`.

The `generate` command tries to take your lagoonized `docker-compose.yml` file and generate a `.lagoon-sync.yml` file based
off what it finds in the service definition.

The [example](./examples/tutorial/.lagoon-sync.yml) above was actually generated by [this docker-compose file](./examples/tutorial/docker-compose.yml).

In order to generate a file, simply run `lagoon-sync generate ./docker-compose.yml -o .lagoon-sync.yml` and it should,
hopefully, generate a reasonable lagoon sync config.

If you'd like to be more hands-on you can run `lagoon-sync interactive-config -o .lagoon-sync.yml` and you'll be presented with a
menu that you can use to generate a sync config.

## What's all this about clusters?

If your project is on anything except the amazeeio cluster, which are the defaults
and you're running lagoon-sync from a local container, you may have to set these variables
you can grab this information from running the lagoon cli's `lagoon config list`
this will output the ssh endpoints and ports you need.

Typically, though, this information is also available in the environment variables
LAGOON_CONFIG_SSH_HOST and LAGOON_CONFIG_SSH_PORT

These, for instance, are the amazeeio defaults - and should do for most people.
If you're on your own cluster, these are the same values that will be in your `.lagoon.yml`

* `ssh: ssh.lagoon.amazeeio.cloud:32222`
* `api: https://api.lagoon.amazeeio.cloud/graphql`

## Please note

At the moment, the generator and wizard only support the most commonly used cases - files, mariadb, and postgres.
Mongodb is actually a far more complex beast, and we'll add some more support for it in the future.


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
  completion         generate the autocompletion script for the specified shell
  config             Print the config that is being used by lagoon-sync
  generate           Generate a lagoon-sync configuration stanza from a docker-compose file
  help               Help about any command
  interactive-config Generate a lagoon-sync configuration stanza interactively
  selfUpdate         Update this tool to the latest version
  sync               Sync a resource type
  version            Print the version number of lagoon-sync

Flags:
      --config string   Path to the file used to set lagoon-sync configuration
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

### Custom synchers

It's possible to extend lagoon-sync to define your own sync processes. As lagoon-sync is essentially a 
script runner that runs commands on target and source systems, as well as transferring data between the two systems,
it's possible to define commands that generate the transfer resource and consume it on the target.

For instance, if you have [mtk](https://github.com/skpr/mtk) set up on the target machine, it should be possible to
define a custom syncher that makes use of mtk to generate a sanitized DB dump on the source, and then use mysql to
import it on the target.

This is done by defining three things:
* The transfer resource name (what file is going to be synced across the network) - in this case let's call it "/tmp/dump.sql"
* The command(s) to run on the source
* The command(s) to run target

```
lagoon-sync:
  mtkdump:
    transfer-resource: "/tmp/dump.sql"
    source:
      commands:
        - "mtk-dump > {{ .transferResource }}"
    target:
      commands:
        - "mysql -h${MARIADB_HOST:-mariadb} -u${MARIADB_USERNAME:-drupal} -p${MARIADB_PASSWORD:-drupal} -P${MARIADB_PORT:-3306} ${MARIADB_DATABASE:-drupal} < {{ .transfer-resource }}"
```

This can then be called by running the following:
```
lagoon-sync sync mtkdump -p <SOURCE_PROJECT> -e <SOURCE_ENVIRONMENT>
```

### Custom configuration files
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
