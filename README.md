# Lagoon-sync

Lagoon-sync is part of the Lagoon cli toolset and, indeed, works closely with its parent project.

## Usage

### Syncing a mariadb database

`lagoon-sync sync --remote-project-name=amazeelabsv4-com --remote-environment-name=dev`git


## Releases

We are using goreleaser for the build, release and publish steps that will be ran from a github action on a pushed tag.

You can also use this tool locally to see what would we released with `make local-snapshot` or `goreleaser release --snapshot --skip-publish --rm-dist`

Note: If verifying signing locally, you might need to (if on mac) - `export GPG_TTY=$(tty)` - https://unix.stackexchange.com/a/257065
