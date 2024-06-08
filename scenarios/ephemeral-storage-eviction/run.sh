#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
NAMESPACE=cm-ephemeral-storage-eviction

kubectl create namespace $NAMESPACE
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite

helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set resources.limits.ephemeral-storage=256Mi \
    --set securityContext.readOnlyRootFilesystem=false \
    --set replicaCount=3 \
    --set business_application=$NAMESPACE \
    single $SCRIPT_DIR/../../helm/single 

helm delete --namespace $NAMESPACE client
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$NAMESPACE-plan.yaml \
    --set business_application=$NAMESPACE \
    client $SCRIPT_DIR/../../helm/client

