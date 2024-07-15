#!/bin/bash

NAMESPACE=chaosmania

kubectl delete -n ${NAMESPACE} -f strimzi-kafka/kafka-topics.yaml
kubectl delete -n ${NAMESPACE} -f strimzi-kafka/kafka-cluster-descriptor.yaml

helm delete -n ${NAMESPACE} kafka-exporter
helm delete -n ${NAMESPACE} strimzi-kafka-operator
