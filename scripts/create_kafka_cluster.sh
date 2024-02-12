#!/bin/bash

NS=chaosmania

# https://github.com/strimzi/strimzi-kafka-operator/tree/main/helm-charts/helm3/strimzi-kafka-operator
helm repo add strimzi https://strimzi.io/charts/
helm upgrade --install strimzi-kafka-operator strimzi/strimzi-kafka-operator -n ${NS} --set featureGates=+UseKRaft
helm upgrade --install kafka-exporter prometheus-community/prometheus-kafka-exporter --namespace=${NS} --values ../kubernetes/strimzi-kafka/exporters/prometheus_kafka_values.yaml

kubectl apply -n ${NS} -f ../kubernetes/strimzi-kafka/kafka-cluster-descriptor.yaml
kubectl apply -n ${NS} -f ../kubernetes/strimzi-kafka/kafka-topics.yaml