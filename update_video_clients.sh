#!/bin/bash

NS=chaosmania
IMAGE_TAG=latest
CHAOS_HOST=frontend

python plans/make_video_plans.py upload create > ./plans/video_upload.yaml 
python plans/make_video_plans.py show-recommendations create > ./plans/video_show_recommendations.yaml 
python plans/make_video_plans.py show-video create > ./plans/video_show_video.yaml 

helm delete --namespace ${NS} client-show-video
helm delete --namespace ${NS} client-show-recommendations
helm delete --namespace ${NS} client-upload

helm upgrade --install --create-namespace --namespace ${NS} client-show-video ./helm/client  --set chaos.plan="/plans/video_show_video.yaml" --set chaos.host=${CHAOS_HOST} --set image.tag=${IMAGE_TAG}
helm upgrade --install --create-namespace --namespace ${NS} client-show-recommendations ./helm/client  --set chaos.plan="/plans/video_show_recommendations.yaml" --set chaos.host=${CHAOS_HOST} --set image.tag=${IMAGE_TAG}
helm upgrade --install --create-namespace --namespace ${NS} client-upload ./helm/client  --set chaos.plan="/plans/video_upload.yaml" --set chaos.host=${CHAOS_HOST} --set image.tag=${IMAGE_TAG}