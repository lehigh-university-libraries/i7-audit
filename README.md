# i7-audit
Various scripts used to ensure data in i7 made its way into i2

## Ensure all items have been migrated

1. [Extract the list of PIDs from your i7 solr instance](./00-extract-solr)
2. [Load pids into your i2 site](./02-load-pids) to ensure all nodes have been created

## Todo

- [ ] Ensure all files have been migrated
- [ ] Ensure metadata has been mapped properly
- [ ] Ensure all derivatives have been created
