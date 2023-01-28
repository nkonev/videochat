#!/usr/bin/env bash

docker swarm init
mkdir -p /mnt/chat-minio/data
chmod -R a+rw /mnt/chat-minio
