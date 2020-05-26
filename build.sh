#!/usr/bin/env bash
set -e

eval $(minikube docker-env)
temp=$(uuidgen | shasum | grep -oh '^\S*')

dirty=$(git status --porcelain)
if [ "$dirty" == "" ]
then
    commitHash="$(git rev-parse --short HEAD)"
else
    commitHash="$(git rev-parse --short HEAD)-dirty"
fi

operator-sdk build kafka-autoconnector:$temp
imageid=$(docker images kafka-autoconnector:$temp -q)
docker tag $imageid kafka-autoconnector:$commitHash