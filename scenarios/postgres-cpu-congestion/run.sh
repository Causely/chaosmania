#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source $SCRIPT_DIR/../../scripts/lib/chaosmania.sh

# Parse arguments
parse_args "$@"

# Setup scenario
SCENARIO=cm-postgres-cpu-congestion

# Setup namespace
setup_namespace $SCENARIO

echo "Deploying DB"
helm upgrade --install --namespace $NAMESPACE \
    --set global.postgresql.auth.postgresPassword=postgres \
    --set commonLabels."app\.kubernetes\.io/part-of"=$SCENARIO \
    --set primary.resources.limits.cpu=400m \
    -f $SCRIPT_DIR/postgres_values.yaml \
    postgres oci://registry-1.docker.io/bitnamicharts/postgresql
    # --set commonLabels.app\.kubernetes\.io/part-of=$NAMESPACE \

echo "Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql -n $NAMESPACE --timeout=300s

# Deploy single instance
upgrade_single "frontend" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=1"

# Deploy single instance
upgrade_single "payment" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=2"


echo "Deploying postgres-exporter..."
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm upgrade --install --namespace $NAMESPACE \
    --set serviceMonitor.enabled=false \
    --set config.datasource.host=postgres-postgresql \
    --set config.datasource.password=postgres \
    --set config.datasource.user=postgres \
    postgres-exporter prometheus-community/prometheus-postgres-exporter

echo "Waiting for Frontend to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=frontend -n $NAMESPACE --timeout=300s

echo "Deploying client"
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client" "frontend" "/scenarios/$SCENARIO-plan.yaml"

echo "Configure scraper"
kubectl create secret generic \
    --save-config \
    --dry-run \
    -o yaml \
    --namespace $NAMESPACE \
    --from-literal=username="postgres" \
    --from-literal=password="postgres" \
    --from-literal=host="postgres-postgresql" \
    --from-literal=port=5432 \
    --from-literal=database="postgres" \
    $SCENARIO-postgres-credentials | kubectl apply -f -

kubectl label secret $SCENARIO-postgres-credentials --namespace $NAMESPACE "causely.ai/scraper=Postgresql" --overwrite
