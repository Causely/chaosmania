#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
NAMESPACE=cm-cascading-failures

kubectl create namespace $NAMESPACE
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite

# Setup istio ingress gateway
helm install istio-ingressgateway istio/gateway -n $NAMESPACE \
    --set defaults.service.type=ClusterIP \
    --set labels.istio=cascading-failures-gateway

kubectl apply -f $SCRIPT_DIR/gateway.yaml -n $NAMESPACE

# Shared DB
echo "Deploying DB"
helm upgrade --install --namespace $NAMESPACE \
    --set global.postgresql.auth.postgresPassword=postgres \
    postgres oci://registry-1.docker.io/bitnamicharts/postgresql

# Inventory Service
echo "Deploying inventory service"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$NAMESPACE-inventory \
    inventory-service $SCRIPT_DIR/../../helm/single 

echo "Setup VS for inventory service"
kubectl apply -f $SCRIPT_DIR/inventory-vs.yaml -n $NAMESPACE

echo "Deploying client for inventory service"
helm delete --namespace $NAMESPACE client-inventory
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.host=istio-ingressgateway.$NAMESPACE.svc.cluster.local. \
    --set chaos.port="80" \
    --set chaos.header="Host:inventory.chaosmania.example.com" \
    --set chaos.plan=/scenarios/cascading-failures/inventory-plan.yaml \
    --set business_application=$NAMESPACE-inventory \
    client-inventory $SCRIPT_DIR/../../helm/client

# Order Service
echo "Deploying order service"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$NAMESPACE-order \
    order-service $SCRIPT_DIR/../../helm/single 

echo "Setup VS for order service"
kubectl apply -f $SCRIPT_DIR/order-vs.yaml -n $NAMESPACE

echo "Deploying client for order service"
helm delete --namespace $NAMESPACE client-order
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.host=istio-ingressgateway.$NAMESPACE.svc.cluster.local. \
    --set chaos.port="80" \
    --set chaos.header="Host:order.chaosmania.example.com" \
    --set chaos.plan=/scenarios/cascading-failures/order-plan.yaml \
    --set business_application=$NAMESPACE-order \
    client-order $SCRIPT_DIR/../../helm/client

# Payment Service
echo "Deploying payment service"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$NAMESPACE-payment \
    payment-service $SCRIPT_DIR/../../helm/single 

echo "Setup VS for payment service"
kubectl apply -f $SCRIPT_DIR/payment-vs.yaml -n $NAMESPACE

# Frontend Service
echo "Deploying frontend service"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$NAMESPACE-frontend \
    frontend-service $SCRIPT_DIR/../../helm/single 

echo "Setup VS for frontend service"
kubectl apply -f $SCRIPT_DIR/frontend-vs.yaml -n $NAMESPACE