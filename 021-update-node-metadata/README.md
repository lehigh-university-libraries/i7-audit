
Get the owner, state, creation date, and last modified date for every PID in solr and save in `metadata.csv`

```
for F in $(ls output); do
  jq -r '.response.docs[] | [.PID, .fgs_ownerId_s, .fgs_state_s, .fgs_createdDate_dt, .fgs_lastModifiedDate_dt] | @csv' output/$F
done > metadata.csv
```


Populate a fedora owner <-> Drupal user mapping by creating a CSV with these contents (making sure to create all the owners in your i7 system in your new i2 system first) called `users.csv`

```
SELECT uid, name
FROM users_field_data
```

Get the nid<->pid mapping and save in `pids.csv`

```
SELECT entity_id, pid from _i7_pids i7
LEFT JOIN node__field_pid i2 ON i2.field_pid_value = i7.pid
WHERE i2.field_pid_value IS NOT NULL;
```