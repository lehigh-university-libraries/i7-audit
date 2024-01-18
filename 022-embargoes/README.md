
Get the embargo values for every PID in solr and save in `embargoes.csv`

```
for F in $(ls output); do
  jq -r '.response.docs[] | [.PID, .mods_originInfo_encoding_iso8601_qualifier_embargo_dateValid_dt, ."RELS_INT_embargo-expiry-notification-date_literal_s", ."RELS_INT_embargo-until_literal_s"] | @csv' output/$F | grep -v ",,,"
done > embargoes.csv
```

Get the nid<->pid mapping and save in `pids.csv`

```
SELECT entity_id, pid from _i7_pids i7
LEFT JOIN node__field_pid i2 ON i2.field_pid_value = i7.pid
WHERE i2.field_pid_value IS NOT NULL;
```

Get the vid<->pid mapping and save in `revisions.csv`
```
SELECT revision_id, pid from _i7_pids i7
LEFT JOIN node__field_pid i2 ON i2.field_pid_value = i7.pid
WHERE i2.field_pid_value IS NOT NULL;
```

Create a SQL command to update the node metadata with the i7 content

```
go run main.go
```

Run the SQL!

Then create the embargo entities

```
drush scr embargoes.php
```
