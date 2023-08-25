#!/bin/bash

repo=k8s-mutating-admission-webhook

version=$(go run ./cmd/webhook -version | awk '{ print $2 }' | awk -F= '{ print $2 }')

echo version=$version

docker build \
    --no-cache \
    -t udhos/$repo:latest \
    -t udhos/$repo:$version \
    -f docker/Dockerfile .

echo "push: docker push udhos/$repo:$version; docker push udhos/$repo:latest"
