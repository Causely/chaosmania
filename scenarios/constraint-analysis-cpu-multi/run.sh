#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
NAMESPACE=cm-constraint-analysis-cpu-multi

kubectl create namespace $NAMESPACE
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite

echo "Deploying frontend"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set business_application=$NAMESPACE \
    frontend $SCRIPT_DIR/../../helm/single 

echo "Deploying payment"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set business_application=$NAMESPACE \
    payment $SCRIPT_DIR/../../helm/single

echo "Deploying cart"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set business_application=$NAMESPACE \
    cart $SCRIPT_DIR/../../helm/single

echo "Deploying shipping"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set business_application=$NAMESPACE \
    shipping $SCRIPT_DIR/../../helm/single

echo "Deploying product"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set business_application=$NAMESPACE \
    product $SCRIPT_DIR/../../helm/single

echo "Deploying client"
helm delete --namespace $NAMESPACE client
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.host=frontend \
    --set chaos.plan=/scenarios/$NAMESPACE-plan.yaml \
    --set business_application=$NAMESPACE \
    client $SCRIPT_DIR/../../helm/client

