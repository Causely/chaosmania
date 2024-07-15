#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_TAG=latest
NAMESPACE=chaosmania

kubectl create namespace ${NAMESPACE}
cd $SCRIPT_DIR
./create_kafka_cluster.sh

helm upgrade --install --namespace ${NAMESPACE} \
    --set image.tag=$IMAGE_TAG \
    --values $SCRIPT_DIR/values.yaml \
    video ../../helm/video
