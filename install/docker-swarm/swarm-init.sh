#!/usr/bin/env bash

docker swarm init
mkdir -p /mnt/chat-minio/data
chmod -R a+rw /mnt/chat-minio
mkdir -p /mnt/chat-storage-tmp
chmod -R a+rw /mnt/chat-storage-tmp
