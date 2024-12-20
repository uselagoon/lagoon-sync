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
