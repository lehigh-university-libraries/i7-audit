## Extract just pids from solr

If all you need is a lisit of pids, you can just view the contents of the fedora object store directory

```
find /opt/islandora/fedora-objectStore -type f > pids.csv
awk -F 'info%3Afedora%2F' '{print $2}' pids.csv| perl -pe 's/%([0-9a-f]{2})/chr hex $1/ieg' > pids_decoded.csv
```

## Extract documents from solr

If you want to get the metadata for all the PIDs you can do this

### Extract

First, port forward port 8080 to your local machine

```
ssh i7.domain -L 8080:localhost:8080
```

Then in another terminal window run a script to crawl the solr index. Get the value for RECORDS based on how many document are in your solr index.

```
#!/usr/bin/env bash

set -eou pipefail

OFFSET=0
LIMIT=100
RECORDS=380432

mkdir -p output

while [ "$OFFSET" -lt "$RECORDS" ]; do
    if [ ! -f "output/solr.$OFFSET.json" ]; then
        url="http://localhost:8080/solr/collection1/select?q=*%3A*&start=$OFFSET&rows=$LIMIT&wt=json&indent=true"
        echo "Fetching $url"
        curl -o output/solr.$OFFSET.json -s "$url"
        sleep 1
    fi

    OFFSET=$((OFFSET + LIMIT))
done

```

### Transform

Then trim down the solr documents

```
#!/usr/bin/env bash

set -eou pipefail

OFFSET=0
LIMIT=100
RECORDS=380432
TOTAL=0

mkdir -p transform

while [ "$OFFSET" -lt "$RECORDS" ]; do
    if [ -f "output/solr.$OFFSET.json" ]; then
        if [ ! -f "transform/solr.$OFFSET.json" ]; then
            # TODO: should also include
            # 'RELS_EXT_isPageOf_uri_ms', 'RELS_EXT_isConstituentOf_uri_ms', 'RELS_EXT_isMemberOf_uri_ms',
            # 'RELS_EXT_isMemberOfCollection_uri_ms', 'RELS_EXT_hasModel_uri_s', 'PID'
            jq '.response.docs[] |= with_entries(select(.key | startswith("dc") or (startswith("mods") and endswith("_mt"))))' output/solr.$OFFSET.json | jq .response.docs > transform/solr.$OFFSET.json
        fi
    fi
    OFFSET=$((OFFSET + LIMIT))
done

```

Put the output into a single doc

```
jq . transform/*.json > all.json
```
