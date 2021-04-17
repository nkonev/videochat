#!/bin/bash
GIT_COMMIT=$(git rev-list -1 HEAD)
echo "{\"commit\": \"${GIT_COMMIT}\", \"microservice\": \"aaa\"}" > $1