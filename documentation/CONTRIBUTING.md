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
