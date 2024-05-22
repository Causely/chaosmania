#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
NAMESPACE=cm-client-throttled

kubectl create namespace $NAMESPACE
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite

helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set resources.limits.cpu="500m"\
    --set replicaCount=3 \
    single $SCRIPT_DIR/../../helm/single 

helm delete --namespace $NAMESPACE client-throttled
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$NAMESPACE-throttled.yaml \
    client-throttled $SCRIPT_DIR/../../helm/client

helm delete --namespace $NAMESPACE client-not-throttled
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$NAMESPACE-not_throttled.yaml \
    client-not-throttled $SCRIPT_DIR/../../helm/client

