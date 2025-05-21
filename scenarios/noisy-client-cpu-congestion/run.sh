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
SCENARIO=cm-noisy-client-cpu-congestion
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

helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set resources.limits.cpu="500m"\
    --set replicaCount=3 \
    --set business_application=$SCENARIO \
    --set otlp.enabled=true \
    single $SCRIPT_DIR/../../helm/single 

helm delete --namespace $NAMESPACE client1
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$SCENARIO-plan1.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=true \
    client1 $SCRIPT_DIR/../../helm/client

helm delete --namespace $NAMESPACE client2
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$SCENARIO-plan2.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=true \
    client2 $SCRIPT_DIR/../../helm/client

helm delete --namespace $NAMESPACE client3
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.plan=/scenarios/$SCENARIO-plan3.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=true \
    client3 $SCRIPT_DIR/../../helm/client

