#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
SCENARIO=cm-periodical-oom
NAMESPACE=$USER-$SCENARIO

echo "Creating namespace $NAMESPACE"
kubectl create namespace $NAMESPACE || true

echo "Labeling namespace $NAMESPACE for Istio injection"
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite || true

helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set resources.limits.memory=256Mi \
    --set replicaCount=3 \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    single $SCRIPT_DIR/../../helm/single 

helm delete --namespace $NAMESPACE client
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$SCENARIO-plan.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    client $SCRIPT_DIR/../../helm/client

