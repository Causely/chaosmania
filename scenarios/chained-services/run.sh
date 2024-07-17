#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
NAMESPACE=cm-chained-services

kubectl create namespace $NAMESPACE
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite

echo "Deploying frontend"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$NAMESPACE \
    frontend $SCRIPT_DIR/../../helm/single 

echo "Deploying payment"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$NAMESPACE \
    payment $SCRIPT_DIR/../../helm/single 

echo "Deploying DB"
helm upgrade --install --namespace $NAMESPACE \
    --set global.postgresql.auth.postgresPassword=postgres \
    --set commonLabels.app\.kubernetes\.io/part-of=$NAMESPACE \
    postgres oci://registry-1.docker.io/bitnamicharts/postgresql

helm upgrade --install --namespace $NAMESPACE \
    --set serviceMonitor.enabled=true \
    --set config.datasource.host=postgres-postgresql \
    --set config.datasource.password=postgres \
    --set config.datasource.user=postgres \
    postgres-exporter prometheus-community/prometheus-postgres-exporter

echo "Deploying client"
helm delete --namespace $NAMESPACE client
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.host=frontend \
    --set chaos.plan=/scenarios/$NAMESPACE-plan.yaml \
    --set business_application=$NAMESPACE \
    client $SCRIPT_DIR/../../helm/client

