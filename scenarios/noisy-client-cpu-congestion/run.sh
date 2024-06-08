#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
NAMESPACE=cm-noisy-client-cpu-congestion

kubectl create namespace $NAMESPACE
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite

helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set resources.limits.cpu="500m"\
    --set replicaCount=3 \
    --set business_application=$NAMESPACE \
    single $SCRIPT_DIR/../../helm/single 

helm delete --namespace $NAMESPACE client1
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$NAMESPACE-plan1.yaml \
    --set business_application=$NAMESPACE \
    client1 $SCRIPT_DIR/../../helm/client

helm delete --namespace $NAMESPACE client2
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$NAMESPACE-plan2.yaml \
    --set business_application=$NAMESPACE \
    client2 $SCRIPT_DIR/../../helm/client

helm delete --namespace $NAMESPACE client3
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$NAMESPACE-plan3.yaml \
    --set business_application=$NAMESPACE \
    client3 $SCRIPT_DIR/../../helm/client

