#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
SCENARIO=cm-chained-cpu-congestion
NAMESPACE=$USER-$SCENARIO

echo "Creating namespace $NAMESPACE"
kubectl create namespace $NAMESPACE || true

echo "Labeling namespace $NAMESPACE for Istio injection"
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite || true


echo "Deploying frontend"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    frontend $SCRIPT_DIR/../../helm/single 

echo "Deploying payment"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set resources.limits.cpu="1000m"\
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    payment-service $SCRIPT_DIR/../../helm/single 

echo "Deploying orders"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    order-service $SCRIPT_DIR/../../helm/single 

echo "Deploying client"
helm delete --namespace $NAMESPACE client || true
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.host=frontend \
    --set chaos.plan=/scenarios/$SCENARIO-plan.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    client $SCRIPT_DIR/../../helm/client
