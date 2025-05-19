#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
SCENARIO=cm-noisy-client-cpu-congestion
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

helm delete --namespace $NAMESPACE client1
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$SCENARIO-plan1.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    client1 $SCRIPT_DIR/../../helm/client

helm delete --namespace $NAMESPACE client2
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$SCENARIO-plan2.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    client2 $SCRIPT_DIR/../../helm/client

helm delete --namespace $NAMESPACE client3
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$SCENARIO-plan3.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=$OTLP_ENABLED \
    client3 $SCRIPT_DIR/../../helm/client

