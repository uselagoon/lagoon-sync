#!/bin/bash

if [ ! -f binaries/rsync ];
then
    echo "Installing and extracting rsync binary..."

    # Download
    cd binaries/
    curl --progress-bar -L -o rsync "https://github.com/JBBgameich/rsync-static/releases/download/continuous/rsync-x86"
    # curl --progress-bar -L -o rsync "https://github.com/JBBgameich/rsync-static/releases/download/continuous/rsync-arm"

    # Permissions
    chmod +x rsync
fi

# Test
file ./binaries/rsync

if [[ -x "./binaries/rsync" ]]
then
    echo "rsync binary is executable"
else
    echo "rsync binary can't execute - check OS/Arch version, running x86 and not ARM CPU"
fi