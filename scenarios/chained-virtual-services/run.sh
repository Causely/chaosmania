#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source $SCRIPT_DIR/../../scripts/lib/chaosmania.sh

# Parse arguments
parse_args "$@"

# Setup scenario
SCENARIO=cm-chained-virtual-services

# Setup namespace
setup_namespace $SCENARIO


# Setup istio ingress gateway
echo "Deploying istio ingress gateway helm chart"
helm install istio-ingressgateway istio/gateway -n $NAMESPACE \
    --set service.type=ClusterIP \
    --set labels.istio=chained-virtual-services-gateway

echo "Deploying istio ingress gateway"
kubectl apply -f $SCRIPT_DIR/gateway.yaml -n $NAMESPACE

# Shared DB
echo "Deploying DB"
helm upgrade --install --namespace $NAMESPACE \
    --set global.postgresql.auth.postgresPassword=postgres \
    postgres oci://registry-1.docker.io/bitnamicharts/postgresql
    # --set commonLabels.app\.kubernetes\.io/part-of=$NAMESPACE \

# App 1
# Deploy single instances
upgrade_single "frontend-app1" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=2"
upgrade_single "payment-app1" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=2"

echo "Setup VS for app1"
kubectl apply -f $SCRIPT_DIR/app1-vs.yaml -n $NAMESPACE

# Deploy client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client-app1" "istio-ingressgateway" "/scenarios/$SCENARIO-app1-plan.yaml" "--set chaos.header=\"Host:app1.chaosmania.example.com\""

# App 2
# Deploy single instances
upgrade_single "frontend-app2" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=2"
upgrade_single "payment-app2" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=2"

echo "Setup VS for app2"
kubectl apply -f $SCRIPT_DIR/app2-vs.yaml -n $NAMESPACE

# Deploy client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client-app2" "istio-ingressgateway" "/scenarios/$SCENARIO-app2-plan.yaml" "--set chaos.header=\"Host:app2.chaosmania.example.com\""
