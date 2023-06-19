# Installation

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


## Quick setup (Linux example)

This script will install `lagoon-sync` and create a configuration file that will connect to a `mariadb` instance. 

```
#!/usr/bin/env bash

DOWNLOAD_PATH=$(curl -sL "https://api.github.com/repos/uselagoon/lagoon-sync/releases/latest" | grep "browser_download_url" | cut -d \" -f 4 | grep linux_amd64) && wget -O /usr/local/bin/lagoon-sync $DOWNLOAD_PATH && chmod a+x /usr/local/bin/lagoon-sync && chmod +x /usr/local/bin/lagoon-sync

cat > .lagoon-sync <<EOF
lagoon-sync:
    mariadb:
      config:
        hostname: "\$MARIADB_DATABASE"
        username: "\$MARIADB_USERNAME"
        password: "\$MARIADB_PASSWORD"
        port: "\$MARIADB_PORT"
        database: "\$MARIADB_DATABASE"
      local:
        config:
          hostname: "\$MARIADB_HOST"
          username: "\$MARIADB_USERNAME"
          password: "\$MARIADB_PASSWORD"
          port: "\$MARIADB_PORT"
          database: "\$MARIADB_DATABASE"
EOF
```