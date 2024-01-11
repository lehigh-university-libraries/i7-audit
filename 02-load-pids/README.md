# Load PIDs

Create a MySQL table to load the PIDs into

```
CREATE TABLE `_i7_pids` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `pid` varchar(1024) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `pid` (`pid`)
)
```

Load `pids_namespaced.csv` into the table

## Find missing PIDs

Finally, we can see what PIDs are in i7 but missing from i2

```
SELECT pid from _i7_pids i7
LEFT JOIN node__field_pid i2 ON i2.field_pid_value = i7.pid
WHERE i2.field_pid_value IS NULL;
```
