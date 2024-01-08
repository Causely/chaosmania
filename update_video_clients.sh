#!/bin/bash

python plans/make_video_plans.py upload create > ./plans/video_upload.yaml 
python plans/make_video_plans.py show-recommendations create > ./plans/video_show_recommendations.yaml 
python plans/make_video_plans.py show-video create > ./plans/video_show_video.yaml 

helm delete --namespace platform client-show-video
helm delete --namespace platform client-show-recommendations
helm delete --namespace platform client-upload

helm upgrade --install --create-namespace --namespace platform client-show-video ./helm/client  --set chaos.plan="/plans/video_show_video.yaml" --set chaos.host=frontend --set image.tag=steffen
helm upgrade --install --create-namespace --namespace platform client-show-recommendations ./helm/client  --set chaos.plan="/plans/video_show_recommendations.yaml" --set chaos.host=frontend --set image.tag=steffen
helm upgrade --install --create-namespace --namespace platform client-upload ./helm/client  --set chaos.plan="/plans/video_upload.yaml" --set chaos.host=frontend --set image.tag=steffen