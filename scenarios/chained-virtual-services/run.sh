#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
SCENARIO=cm-chained-virtual-services
NAMESPACE=$USER-$SCENARIO

echo "Creating namespace $NAMESPACE"
kubectl create namespace $NAMESPACE || true

echo "Labeling namespace $NAMESPACE for Istio injection"
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite || true

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
echo "Deploying frontend"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$SCENARIO-app1 \
    --set otlp.enabled=$OTLP_ENABLED \
    frontend-app1 $SCRIPT_DIR/../../helm/single 

echo "Deploying payment"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$SCENARIO-app1 \
    --set otlp.enabled=$OTLP_ENABLED \
    payment-app1 $SCRIPT_DIR/../../helm/single 


echo "Setup VS for app1"
kubectl apply -f $SCRIPT_DIR/app1-vs.yaml -n $NAMESPACE

echo "Deploying client app1"
helm delete --namespace $NAMESPACE client-app1
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.host=istio-ingressgateway.$NAMESPACE.svc.cluster.local. \
    --set chaos.port="80" \
    --set chaos.header="Host:app1.chaosmania.example.com" \
    --set chaos.plan=/scenarios/cm-chained-virtual-services-app1-plan.yaml \
    --set business_application=$SCENARIO-app1 \
    --set otlp.enabled=$OTLP_ENABLED \
    client-app1 $SCRIPT_DIR/../../helm/client

# App 2
echo "Deploying frontend"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$SCENARIO-app2 \
    --set otlp.enabled=$OTLP_ENABLED \
    frontend-app2 $SCRIPT_DIR/../../helm/single 

echo "Deploying payment"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$SCENARIO-app2 \
    --set otlp.enabled=$OTLP_ENABLED \
    payment-app2 $SCRIPT_DIR/../../helm/single 

echo "Setup VS for app2"
kubectl apply -f $SCRIPT_DIR/app2-vs.yaml -n $NAMESPACE

echo "Deploying client app2"
helm delete --namespace $NAMESPACE client-app2
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.host=istio-ingressgateway.$NAMESPACE.svc.cluster.local. \
    --set chaos.port="80" \
    --set chaos.header="Host:app2.chaosmania.example.com" \
    --set chaos.plan=/scenarios/cm-chained-virtual-services-app2-plan.yaml \
    --set business_application=$SCENARIO-app2 \
    --set otlp.enabled=$OTLP_ENABLED \
    client-app2 $SCRIPT_DIR/../../helm/client
