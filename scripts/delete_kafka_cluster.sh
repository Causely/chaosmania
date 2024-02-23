#!/bin/bash

NS=chaosmania

kubectl delete -n ${NS} -f ../kubernetes/strimzi-kafka/kafka-topics.yaml
kubectl delete -n ${NS} -f ../kubernetes/strimzi-kafka/kafka-cluster-descriptor.yaml

helm delete -n ${NS} kafka-exporter
helm delete -n ${NS} strimzi-kafka-operator
