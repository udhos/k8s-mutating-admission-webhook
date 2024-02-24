#!/bin/bash

app=k8s-mutating-admission-webhook

version=$(go run ./cmd/webhook -version | awk '{ print $2 }' | awk -F= '{ print $2 }')

echo version=$version

docker build \
    --no-cache \
    -t udhos/$app:latest \
    -t udhos/$app:$version \
    -f docker/Dockerfile .

echo push:
echo "docker push udhos/$app:$version; docker push udhos/$app:latest" > docker-push.sh
chmod a+rx docker-push.sh
echo docker-push.sh:
cat docker-push.sh
