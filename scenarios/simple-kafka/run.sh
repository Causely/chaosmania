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
SCENARIO=cm-simple-kafka
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

echo
echo "Deploying Kafka"
helm upgrade --install --namespace $NAMESPACE \
    my-kafka oci://registry-1.docker.io/bitnamicharts/kafka

PASSWORD=$(kubectl get secret my-kafka-user-passwords --namespace $NAMESPACE -o jsonpath='{.data.client-passwords}' | base64 -d | cut -d , -f 1)

echo
echo "Deploying producer"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set business_application=$SCENARIO \
    --set services[0].name=kafka-producer \
    --set services[0].type=kafka-producer \
    --set services[0].config.peer_service=kafka \
    --set services[0].config.peer_namespace=$SCENARIO \
    --set services[0].config.brokers[0]=my-kafka-controller-0.my-kafka-controller-headless.$SCENARIO.svc.cluster.local:9092 \
    --set services[0].config.brokers[1]=my-kafka-controller-1.my-kafka-controller-headless.$SCENARIO.svc.cluster.local:9092 \
    --set services[0].config.brokers[2]=my-kafka-controller-2.my-kafka-controller-headless.$SCENARIO.svc.cluster.local:9092 \
    --set services[0].config.username=user1 \
    --set services[0].config.password="$PASSWORD" \
    --set services[0].config.tls_enable=false \
    --set services[0].config.sasl_enable=true \
    --set otlp.enabled=true \
    producer $SCRIPT_DIR/../../helm/single 

echo
echo "Deploying consumer"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set business_application=$SCENARIO \
    --set background_services[0].name=kafka-consumer \
    --set background_services[0].type=kafka-consumer \
    --set background_services[0].config.peer_service=kafka \
    --set background_services[0].config.peer_namespace=$SCENARIO \
    --set background_services[0].config.brokers[0]=my-kafka.$SCENARIO.svc.cluster.local:9092 \
    --set background_services[0].config.username=user1 \
    --set background_services[0].config.password="$PASSWORD" \
    --set background_services[0].config.tls_enable=false \
    --set background_services[0].config.sasl_enable=true \
    --set background_services[0].config.topic=test1 \
    --set background_services[0].config.group=my-consumer-group \
    --set background_services[0].config.script="function run() { var msg = ctx.get_message(); ctx.print('Received message: ' + msg); }" \
    --set enabled_background_services[0]="kafka-consumer" \
    --set otlp.enabled=true \
    consumer $SCRIPT_DIR/../../helm/single 

echo
echo "Deploying exporter"
helm upgrade --install --namespace $NAMESPACE \
    --set prometheus.serviceMonitor.enabled=true \
    --set kafkaServer[0]=my-kafka-controller-0.my-kafka-controller-headless.$SCENARIO.svc.cluster.local:9092 \
    --set sasl.enabled=true \
    --set sasl.scram.enabled=true \
    --set sasl.scram.mechanism=scram-sha256 \
    --set sasl.scram.username=user1 \
    --set sasl.scram.password="$PASSWORD" \
    --set prometheus.serviceMonitor.enabled=true \
    --set prometheus.serviceMonitor.namespace=$SCENARIO \
    --set prometheus.serviceMonitor.relabelings[0].action=replace \
    --set prometheus.serviceMonitor.relabelings[0].replacement=my-kafka.$SCENARIO.svc.cluster.local:9092 \
    --set prometheus.serviceMonitor.relabelings[0].targetLabel=target \
    kafka-exporter prometheus-community/prometheus-kafka-exporter

echo
echo "Deploying client"
helm delete --namespace $NAMESPACE client
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.host=producer \
    --set chaos.plan=/scenarios/$SCENARIO-plan.yaml \
    --set business_application=$SCENARIO \
    --set otlp.enabled=true \
    client $SCRIPT_DIR/../../helm/client

