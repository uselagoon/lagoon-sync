#!/bin/bash

RSYNC_COMMAND=$(which rsync)

if ! command -v "${RSYNC_COMMAND}" > /dev/null
then
    echo "Installing and extracting rsync binary..."

    # Download
    cd /tmp/
    curl --progress-bar -L -o rsync "https://github.com/JBBgameich/rsync-static/releases/download/continuous/rsync-x86"

    # # Get from official source
    # curl --progress-bar -L -o rsync.tar.gz "https://download.samba.org/pub/rsync/binaries/debian-10-x86_64/latest.tar.gz"
    # tar -xzvf rsync.tar.gz
    # rm rsync.tar.gz

    # # Permissions
    chmod +x ./rsync
    cp rsync /usr/bin/
else
    echo "rsync installed at: $(which rsync)"
fi

# Test
rsync --version