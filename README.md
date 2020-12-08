# Lagoon-sync

Lagoon-sync is part of the Lagoon cli toolset and, indeed, works closely with its parent project.

## Usage

### Syncing a mariadb database

`lagoon-sync sync --remote-project-name=amazeelabsv4-com --remote-environment-name=dev`git


## Contributing

`make all`       Installs missing dependencies, runs tests and build locally.
`make build`     Compiles binary based on current go env.
`make clean`     Remove all build files and assets.

## Releases

We are using goreleaser for the official build, release and publish steps that will be ran from a github action on a pushed tag.

Locally, we can run `make release-test` to check if our changes will build. If compiling was successful we can commit our changes and then run `make release-[patch|minor|major]` to tag with next release number and it will push up to GitHub. A GitHub action will then be triggered which will publish the official release using goreleaser.