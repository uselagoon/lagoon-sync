# Lagoon-sync

Lagoon-sync is part of the Lagoon cli toolset and, indeed, works closely with its parent project.

## Usage

### Syncing a mariadb database

`lagoon-sync sync --remote-project-name=amazeelabsv4-com --remote-environment-name=dev`


## Config files

Config files that can be used in order of priority:
- .lagoon-sync-defaults _(no yaml ext neeeded)_
- .lagoon-sync _(no yaml ext neeeded)_
- .lagoon.yml _Main config file - path can be given as an argument with `--config`, default is `.lagoon.yml`_ 

If either `LAGOON_SYNC_PATH` or `LAGOON_SYNC_DEFAULTS_PATH` env vars are set then it will use those paths instead of the main config file - e.g.

```export LAGOON_SYNC_DEFAULTS_PATH="/lagoon/.lagoon-sync-defaults"```
```export LAGOON_SYNC_PATH="/lagoon/.lagoon-sync"```

To see which config file is active and other configuration settings you can run the `go run main.go config` command which return this data as json.


### Example source-env overrides
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
      hostname: "$MARIADB_HOSTNAME"
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

## Contributing

`make all`       Installs missing dependencies, runs tests and build locally.
`make build`     Compiles binary based on current go env.
`make clean`     Remove all build files and assets.

## Releases

We are using goreleaser for the official build, release and publish steps that will be ran from a github action on a pushed tag.

Locally, we can run `make release-test` to check if our changes will build. If compiling was successful we can commit our changes and then run `make release-[patch|minor|major]` to tag with next release number and it will push up to GitHub. A GitHub action will then be triggered which will publish the official release using goreleaser.