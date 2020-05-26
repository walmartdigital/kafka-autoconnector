#!/usr/bin/env bash
set -e

eval $(minikube docker-env)
temp=$(uuidgen | shasum | grep -oh '^\S*')
commitHash="$(git rev-parse --short HEAD)-dirty"

operator-sdk build kafka-autoconnector:$temp
imageid=$(docker images kafka-autoconnector:$temp -q)
docker tag $imageid kafka-autoconnector:$commitHash