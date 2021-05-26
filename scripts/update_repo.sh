#!/bin/bash

if [ "$GITHUB_ACTIONS" != "true" ]; then
    echo "This script is only meant to be run in GitHub Actions."
    exit 1
fi

mkdir upstream/distRepo/download
cp -f dist/*.rpm upstream/distRepo/download
cp -f dist/*.deb upstream/distRepo/download
cd upstream/distRepo || exit
chmod +x ./update.sh
./update.sh
