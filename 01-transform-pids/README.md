# Extract PIDs from solr export

Get the PIDs for objects in the proper namespace
```
jq -r '.[] | .["dc.identifier"][] | select(test("^(digitalcollections|preserve):"))' all.json | sort >> pids_namespaced.csv
```

Get the PIDS for the islandora namespace (this will include duplicates also found above)
```
jq -r '.[] | .["dc.identifier"][] | select(test("^(islandora):"))' all.json | sort > pids_islandora.csv
```

Remove `chr(160)` i.e. non-breaking space character from the islandora namespace

```
cat pids_islandora.csv | sed 's/'$(echo -e "\u00A0")'//' > pids_islandora_clean.csv
```

Transform the namespaced PIDs into the islandora namespace

```
cat pids_namespaced.csv | sed 's/digitalcollections:/islandora:/' | sed 's/preserve:/islandora:/' > pids_global.csv
```


Make the final CSV a real CSV
```
cp pids_namespaced.csv pids_namespaced.tmp 
echo "pid" > pids_namespaced.csv
cat pids_namespaced.tmp >> pids_namespaced.csv
rm pids_namespaced.tmp
```

Get PIDs that are not namespaced

```
comm -23 pids_islandora_clean.csv pids_global.csv >> pids_namespaced.csv
```

Now `pids_namespaced.csv` contains all PIDs in i7 we could extract from solr.

Also check those object we could not find a PID for.

```
jq '.[] | select(.["dc.identifier"] == null)' all.json > missing.json
jq -r '.[] | select(all(."dc.identifier"[]; test("^(digitalcollections|preserve|islandora):") | not))' all.json >> missing.json
```
