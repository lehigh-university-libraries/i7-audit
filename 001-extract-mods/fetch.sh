#!/usr/bin/env bash

set -eou pipefail

mkdir -p xml/preserve xml/digitalcollections

while IFS=',' read -r NID PID; do
    if [ "$PID" = "pid" ]; then
      continue
    fi

    IFS=':' read -r DOMAIN ID <<< "$PID"
    FILE="xml/$DOMAIN/$PID.xml"
    if [ ! -f "$FILE" ]; then
      curl -so $FILE "https://$DOMAIN.lib.lehigh.edu/islandora/object/$PID/datastream/MODS/download"
    fi
done < pids.csv
