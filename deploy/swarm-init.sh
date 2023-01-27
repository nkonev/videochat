#!/usr/bin/env bash

docker swarm init
docker network create --driver=overlay proxy_backend
docker node update --label-add 'blog.server.role=db' $(docker node ls -q)