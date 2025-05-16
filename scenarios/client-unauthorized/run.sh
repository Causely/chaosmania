#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
SCENARIO=cm-client-unauthorized
NAMESPACE=$USER-$SCENARIO

echo "Creating namespace $NAMESPACE"
kubectl create namespace $NAMESPACE || true

echo "Labeling namespace $NAMESPACE for Istio injection"
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite || true

helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set resources.limits.cpu="500m"\
    --set replicaCount=3 \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    single $SCRIPT_DIR/../../helm/single 

helm delete --namespace $NAMESPACE client-unauthorized
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$SCENARIO-unauthorized.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    client-unauthorized $SCRIPT_DIR/../../helm/client

helm delete --namespace $NAMESPACE client-not-unauthorized
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$SCENARIO-not_unauthorized.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    client-not-unauthorized $SCRIPT_DIR/../../helm/client

