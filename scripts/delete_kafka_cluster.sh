#!/bin/bash

NS=chaosmania

helm delete -n ${NS} strimzi-kafka-operator
helm delete -n ${NS} kafka-exporter

kubectl delete -n ${NS} -f ../kubernetes/strimzi-kafka/kafka-cluster-descriptor.yaml
kubectl delete -n ${NS} -f ../kubernetes/strimzi-kafka/kafka-topics.yaml