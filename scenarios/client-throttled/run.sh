#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
NAMESPACE=cm-client-unauthorized

kubectl create namespace $NAMESPACE
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite

helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set resources.limits.cpu="500m"\
    --set replicaCount=3 \
    single $SCRIPT_DIR/../../helm/single 

helm delete --namespace $NAMESPACE client-unauthorized
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$NAMESPACE-unauthorized.yaml \
    client-unauthorized $SCRIPT_DIR/../../helm/client

helm delete --namespace $NAMESPACE client-not-unauthorized
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$NAMESPACE-not_unauthorized.yaml \
    client-not-unauthorized $SCRIPT_DIR/../../helm/client

