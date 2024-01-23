Ensure every file in i2 migrated from i7 came across OK.

Get the sha1 of every file in i7

```
nohup php sha1.php > sha1s.tsv &
```

Get the sha1 of every file in i2, by PID

```
SELECT field_pid_value AS pid, f.sha1 FROM media__field_media_use mu
  INNER JOIN media__field_media_of mo ON mo.entity_id = mu.entity_id
  INNER JOIN node__field_pid p ON p.entity_id = field_media_of_target_id
  LEFT JOIN media__field_media_image mi ON mi.entity_id = mu.entity_id
  LEFT JOIN media__field_media_file mf ON mf.entity_id = mu.entity_id
  LEFT JOIN media__field_media_document md ON md.entity_id = mu.entity_id
  LEFT JOIN media__field_media_audio_file ma ON ma.entity_id = mu.entity_id
  LEFT JOIN media__field_media_video_file mv ON mv.entity_id = mu.entity_id
  INNER JOIN file_managed f ON f.fid = field_media_image_target_id
    OR f.fid = field_media_file_target_id
    OR f.fid = field_media_document_target_id
    OR f.fid = field_media_audio_file_target_id
    OR f.fid = field_media_video_file_target_id
  WHERE field_media_use_target_id = 16
  ORDER BY field_pid_value
```
