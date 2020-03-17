#!/bin/bash

set -e
MARKER_FILENAME=/opt/jboss/one-shot-import-applied

if [[ ! -f "$MARKER_FILENAME" ]]; then
  echo "============================"
  echo "One-shot import started !!!"
  echo "============================"
  /opt/jboss/keycloak/bin/jboss-cli.sh --file=/opt/jboss/my-cli-scripts/0-launcher-ha.cli
  touch $MARKER_FILENAME
  echo "============================"
  echo "One-shot import finished !!!"
  echo "============================"
else
  echo "============================"
  echo "One-shot import skipped  !!!"
  echo "============================"
fi