#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
NAMESPACE=cm-postgres-cpu-congestion

kubectl create namespace $NAMESPACE
kubectl label namespace $NAMESPACE istio-injection=disabled --overwrite


echo "Deploying DB"
helm upgrade --install --namespace $NAMESPACE \
    --set global.postgresql.auth.postgresPassword=postgres \
    --set commonLabels."app\.kubernetes\.io/part-of"=$NAMESPACE \
    --set primary.resources.limits.cpu=400m \
    -f $SCRIPT_DIR/postgres_values.yaml \
    postgres oci://registry-1.docker.io/bitnamicharts/postgresql
    # --set commonLabels.app\.kubernetes\.io/part-of=$NAMESPACE \

helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set business_application=$NAMESPACE \
    frontend $SCRIPT_DIR/../../helm/single 

echo "Deploying payment"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$NAMESPACE \
    payment $SCRIPT_DIR/../../helm/single 

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

echo "Configure scraper"
kubectl create secret generic \
    --save-config \
    --dry-run \
    -o yaml \
    --namespace $NAMESPACE \
    --from-literal=username="postgres" \
    --from-literal=password="postgres" \
    --from-literal=host="postgres-postgresql.$NAMESPACE.svc.cluster.local" \
    --from-literal=port=5432 \
    --from-literal=database="postgres" \
    $NAMESPACE-postgres-credentials | kubectl apply -f -

kubectl label secret $NAMESPACE-postgres-credentials --namespace $NAMESPACE "causely.ai/scraper=Postgresql" --overwrite