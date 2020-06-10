#!/usr/bin/env bash
set -e

# It is best to start Minikube with these settings -> minikube start --memory 12Gi --cpus=4
# Requires Helm 3
# Need to manually add Elastic repo -> helm repo add elastic https://helm.elastic.co

chart=$(helm list -o json | jq -r '[.[].name] | index("elasticsearch")')
if [ "$chart" = "null" ]
then
    helm install elasticsearch elastic/elasticsearch \
        --set replicas=2 \
        --set antiAffinity="soft" \
        --set esJavaOpts="-Xmx128m -Xms128m" \
        --set resources.requests.cpu="100m" \
        --set resources.requests.memory="512M" \
        --set resources.limits.cpu="1000m" \
        --set resources.limits.memory="512M" \
        --set "volumeClaimTemplate.accessModes[0]"="ReadWriteOnce" \
        --set volumeClaimTemplate.storageClassName="standard" \
        --set volumeClaimTemplate.resources.requests.storage="200M"
else
    echo "Elasticsearch is already installed, do nothing"
fi
kubectl patch svc elasticsearch-master -p '{"spec": {"type": "NodePort"}}'

kubectl apply -f ./deploy/kafka-connect/zookeeper.yaml
kubectl apply -f ./deploy/kafka-connect/kafka.yaml
kubectl apply -f ./deploy/kafka-connect/kafka-connect.yaml

eval $(minikube docker-env)
temp=$(uuidgen | shasum | grep -oh '^\S*')

dirty=$(git status --porcelain)
if [ "$dirty" == "" ]
then
    if [[ $(git tag) ]]; then
        imagetag="$(git describe --tags master)"
    else
        imagetag="$(git rev-parse --short HEAD)"
    fi
else
    if [[ $(git tag) ]]; then
        imagetag="$(git describe --tags master)-dirty"
    else
        imagetag="$(git rev-parse --short HEAD)-dirty"
    fi
fi

operator-sdk build kafka-autoconnector:$temp
imageid=$(docker images kafka-autoconnector:$temp -q)
docker tag $imageid kafka-autoconnector:$imagetag
