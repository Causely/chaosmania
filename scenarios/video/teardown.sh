#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_TAG=latest
NAMESPACE=chaosmania

helm delete --namespace ${NAMESPACE} client-show-video
helm delete --namespace ${NAMESPACE} client-show-recommendations
helm delete --namespace ${NAMESPACE} client-upload
helm delete --namespace ${NAMESPACE} video

cd $SCRIPT_DIR
./delete_kafka_cluster.sh
