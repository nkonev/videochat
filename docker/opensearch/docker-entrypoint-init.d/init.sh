#!/bin/bash

set -e

HOST=$1
PORT=$2

echo "Got $HOST $PORT"

echo "Creating index template"
curl --fail-with-body -i -Ss -X PUT "http://$HOST:$PORT/_index_template/videochat_template" -H 'Content-Type: application/json' -d'
{
  "index_patterns": [
    "videochat-*"
  ],
  "template": {
    "aliases": {
      "videochat": {}
    },
    "settings": {
      "number_of_shards": 1,
      "number_of_replicas": 0
    }
  }
}
'
echo "Creating index template finished"

echo "Creating policy"
curl --fail-with-body -i -Ss -X PUT "http://$HOST:$PORT/_plugins/_ism/policies/delete_old_indexes_policy?pretty" -H 'Content-Type: application/json' -d'
{
  "policy": {
    "description": "delete old indexes",
    "default_state": "hot",
    "schema_version": 1,
    "states": [
      {
        "name": "hot",
        "transitions": [
          {
            "state_name": "delete",
            "conditions": {
              "min_index_age": "2d"
            }
          }
        ]
      },
      {
        "name": "delete",
        "actions": [
          {
            "delete": {}
          }
        ]
      }
    ],
    "ism_template": {
      "index_patterns": ["videochat-*"],
      "priority": 100
    }
  }
}
'
echo "Creating policy finished"

echo "Configuring task interval"
curl --fail-with-body -i -Ss -X PUT "http://$HOST:$PORT/_cluster/settings?pretty=true" -H 'Content-Type: application/json' -d'
{
  "persistent" : {
    "plugins.index_state_management.job_interval" : 1
  }
}
'
echo "Configuring task interval finished"
