#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
NAMESPACE=cm-simple-kafka

kubectl create namespace $NAMESPACE
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite

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
    --set business_application=$NAMESPACE \
    --set services[0].name=kafka-producer \
    --set services[0].type=kafka-producer \
    --set services[0].config.peer_service=kafka \
    --set services[0].config.peer_namespace=$NAMESPACE \
    --set services[0].config.brokers[0]=my-kafka-controller-0.my-kafka-controller-headless.$NAMESPACE.svc.cluster.local:9092 \
    --set services[0].config.brokers[1]=my-kafka-controller-1.my-kafka-controller-headless.$NAMESPACE.svc.cluster.local:9092 \
    --set services[0].config.brokers[2]=my-kafka-controller-2.my-kafka-controller-headless.$NAMESPACE.svc.cluster.local:9092 \
    --set services[0].config.username=user1 \
    --set services[0].config.password="$PASSWORD" \
    --set services[0].config.tls_enable=false \
    --set services[0].config.sasl_enable=true \
    producer $SCRIPT_DIR/../../helm/single 

echo
echo "Deploying consumer"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set business_application=$NAMESPACE \
    --set background_services[0].name=kafka-consumer \
    --set background_services[0].type=kafka-consumer \
    --set background_services[0].config.peer_service=kafka \
    --set background_services[0].config.peer_namespace=$NAMESPACE \
    --set background_services[0].config.brokers[0]=my-kafka.$NAMESPACE.svc.cluster.local:9092 \
    --set background_services[0].config.username=user1 \
    --set background_services[0].config.password="$PASSWORD" \
    --set background_services[0].config.tls_enable=false \
    --set background_services[0].config.sasl_enable=true \
    --set background_services[0].config.topic=test1 \
    --set background_services[0].config.group=my-consumer-group \
    --set background_services[0].config.script="function run() { var msg = ctx.get_message(); ctx.print('Received message: ' + msg); }" \
    --set enabled_background_services[0]="kafka-consumer" \
    consumer $SCRIPT_DIR/../../helm/single 

echo
echo "Deploying client"
helm delete --namespace $NAMESPACE client
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.host=producer \
    --set chaos.plan=/scenarios/$NAMESPACE-plan.yaml \
    --set business_application=$NAMESPACE \
    client $SCRIPT_DIR/../../helm/client

