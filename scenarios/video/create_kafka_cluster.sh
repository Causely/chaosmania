#!/bin/bash

NAMESPACE=chaosmania

# https://github.com/strimzi/strimzi-kafka-operator/tree/main/helm-charts/helm3/strimzi-kafka-operator
helm repo add strimzi https://strimzi.io/charts/
## UseKRaft featureGate introduced in Strimzi 0.36.1, enabled by default in 0.40.0 and removed in 0.42.0
helm upgrade --install strimzi-kafka-operator strimzi/strimzi-kafka-operator -n ${NAMESPACE} ## --set featureGates=+UseKRaft
helm upgrade --install kafka-exporter prometheus-community/prometheus-kafka-exporter --namespace=${NAMESPACE} --values strimzi-kafka/exporters/prometheus_kafka_values.yaml

kubectl apply -n ${NAMESPACE} -f strimzi-kafka/kafka-cluster-descriptor.yaml
kubectl apply -n ${NAMESPACE} -f strimzi-kafka/kafka-topics.yaml

kubectl wait --for=condition=Ready kafka -n ${NAMESPACE} chaosmania-kafka-cluster --timeout=60s
