#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Parse command line arguments
PREFIX_USER=false
for arg in "$@"; do
    case $arg in
        --prefix-user)
            PREFIX_USER=true
            shift
            ;;
    esac
done

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
SCENARIO=cm-chained-services
# Set namespace based on --prefix-user flag
if [ "$PREFIX_USER" = true ]; then
    NAMESPACE=$USER-$SCENARIO
else
    NAMESPACE=$SCENARIO
fi

echo "Creating namespace $NAMESPACE"
kubectl create namespace $NAMESPACE || true

echo "Labeling namespace $NAMESPACE for Istio injection"
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite || true

echo "Deploying frontend"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$SCENARIO \
    --set otlp.enabled=true \
    frontend $SCRIPT_DIR/../../helm/single 

echo "Deploying payment"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$SCENARIO \
    --set otlp.enabled=true \
    payment $SCRIPT_DIR/../../helm/single 

echo "Deploying DB"
helm upgrade --install --namespace $NAMESPACE \
    --set global.postgresql.auth.postgresPassword=postgres \
    --set commonLabels."app\.kubernetes\.io/part-of"=$SCENARIO \
    postgres oci://registry-1.docker.io/bitnamicharts/postgresql
    # --set commonLabels.app\.kubernetes\.io/part-of=$NAMESPACE \

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
    --set chaos.plan=/scenarios/$SCENARIO-plan.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=true \
    client $SCRIPT_DIR/../../helm/client

