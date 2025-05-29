#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source $SCRIPT_DIR/../../scripts/lib/chaosmania.sh

# Parse arguments
parse_args "$@"

# Setup scenario
SCENARIO=cm-chained-services

# Setup namespace
setup_namespace $SCENARIO

# Deploy single instance
upgrade_single "frontend" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=2"
upgrade_single "payment" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=2"

echo "Deploying DB"
helm upgrade --install --namespace $NAMESPACE \
    --set global.postgresql.auth.postgresPassword=postgres \
    --set commonLabels."app\.kubernetes\.io/part-of"=$SCENARIO \
    postgres oci://registry-1.docker.io/bitnamicharts/postgresql
    # --set commonLabels.app\.kubernetes\.io/part-of=$NAMESPACE \

echo "Deploying postgres-exporter"
helm upgrade --install --namespace $NAMESPACE \
    --set serviceMonitor.enabled=true \
    --set config.datasource.host=postgres-postgresql \
    --set config.datasource.password=postgres \
    --set config.datasource.user=postgres \
    postgres-exporter prometheus-community/prometheus-postgres-exporter

# Deploy client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client" "frontend" "/scenarios/$SCENARIO-plan.yaml"
