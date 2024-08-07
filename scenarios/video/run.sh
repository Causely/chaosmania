#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_TAG=latest
CHAOS_HOST=frontend
NAMESPACE=chaosmania

cd $SCRIPT_DIR

# conda activate causely
# pip install typer pydantic yaml
python ./make_video_plans.py upload create > ./video_upload.yaml
python ./make_video_plans.py show-recommendations create > ./video_show_recommendations.yaml
python ./make_video_plans.py show-video create > ./video_show_video.yaml

helm delete --namespace ${NAMESPACE} client-show-video
helm delete --namespace ${NAMESPACE} client-show-recommendations
helm delete --namespace ${NAMESPACE} client-upload

helm upgrade --install --create-namespace --namespace ${NAMESPACE} client-show-video ../../helm/client  --set chaos.plan="/scenarios/cm-video-video_show_video.yaml" --set chaos.host=${CHAOS_HOST} --set image.tag=${IMAGE_TAG}
helm upgrade --install --create-namespace --namespace ${NAMESPACE} client-show-recommendations ../../helm/client  --set chaos.plan="/scenarios/cm-video-video_show_recommendations.yaml" --set chaos.host=${CHAOS_HOST} --set image.tag=${IMAGE_TAG}
helm upgrade --install --create-namespace --namespace ${NAMESPACE} client-upload ../../helm/client  --set chaos.plan="/scenarios/cm-video-video_upload.yaml" --set chaos.host=${CHAOS_HOST} --set image.tag=${IMAGE_TAG}