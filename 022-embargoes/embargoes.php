<?php

use Drupal\embargo\Entity\Embargo;
use Drupal\embargo\EmbargoInterface;

$results = \Drupal::database()->query("SELECT entity_id, field_embargo_expiry_value
  FROM node__field_embargo_expiry");
foreach ($results as $row) {
  $embargo = Embargo::create([
    'embargo_type' => EmbargoInterface::EMBARGO_TYPE_FILE,
    'expiration_type' => EmbargoInterface::EXPIRATION_TYPE_SCHEDULED,
    'expiration_date' => explode('T', $row->field_embargo_expiry_value)[0],
    'embargoed_node' => $row->entity_id,
  ]);

  $embargo->save();
}
