# Lagoon-sync

[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/10765/badge)](https://www.bestpractices.dev/projects/10765)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/uselagoon/lagoon-sync/badge)](https://securityscorecards.dev/viewer/?uri=github.com/uselagoon/lagoon-sync)
[![coverage](https://raw.githubusercontent.com/uselagoon/lagoon-sync/badges/.badges/main/coverage.svg)](https://github.com/uselagoon/lagoon-sync/actions/workflows/coverage.yaml)

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


# Documentation

See this document for a brief tutorial on getting started with `lagoon-sync`. Other topics covered in the documentation are

* [Installation options](./documentation/INSTALLATION.md)
* [More details about custom configurations](./documentation/CONFIG.md)
* [Lagoon-sync usage examples](./documentation/EXAMPLE_SYNCS.md)
* [Writing custom synchers](./documentation/CUSTOM.md)
* [Contributing](./documentation/CONTRIBUTING.md)
* [Odds and ends](./documentation/MISC.md)



# Tutorial - Getting started with lagoon-sync

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


