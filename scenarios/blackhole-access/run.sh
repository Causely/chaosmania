#!/bin/bash

source ./common_vars.sh

update_coredns_configmap() {
    local dir="${SCRIPT_DIR}/../../kubernetes/blackhole"

    if [ -f "${COREDNS_CM_COPY}" ]; then
        echo "Updating CoreDNS ConfigMap from file: ${COREDNS_CM_COPY}"
        kubectl -n kube-system apply -f "${COREDNS_CM_COPY}"
    else
        echo "CoreDNS ConfigMap file not found in the directory...skipping"
    fi
}

# Function to iterate over each YAML file in a directory and present a prompt
process_ingress_files() {
    local dir="${INGRESS_DIR}"

    if [ -d "$dir" ]; then
        for file in "$dir"/*ingress.yaml; do
            if [ -f "$file" ]; then
                echo "Applying Ingress for file: $(basename $file)"
                kubectl -n "${NAMESPACE}" apply -f "$file"
            else
                echo "No '*ingress.yaml' files found in $dir ... aborting"
                exit 1;
            fi
        done
    else
        echo "Directory $dir not found... aborting"
        exit 1;
    fi
}

deploy_boutique() {
  helm delete --namespace $NAMESPACE boutique
  helm upgrade --install --create-namespace --namespace $NAMESPACE \
      --set image.tag=$IMAGE_TAG \
      --set business_application=$NAMESPACE \
      --set otlp.enabled=true \
      --set otlp.endpoint="${OTLP_COLLECTOR}" \
      boutique $SCRIPT_DIR/../../helm/boutique
}

deploy_single() {
  helm delete --namespace $NAMESPACE single
  helm upgrade --install --namespace $NAMESPACE \
      --set image.tag=$IMAGE_TAG \
      --set replicaCount=1 \
      --set business_application=$NAMESPACE \
      --set otlp.enabled=true \
      --set otlp.endpoint="${OTLP_COLLECTOR}" \
      single $SCRIPT_DIR/../../helm/single
}

deploy_plan_client() {
  helm delete --namespace $NAMESPACE client
  helm upgrade --install --create-namespace --namespace $NAMESPACE \
      --set image.tag=$IMAGE_TAG \
      --set chaos.host=frontend \
      --set chaos.plan=/scenarios/$NAMESPACE/plan.yaml \
      --set business_application=$NAMESPACE \
      client $SCRIPT_DIR/../../helm/client
}

run_npm_commands() {
    local target_dir="${SCRIPT_DIR}/../../external-services/blackhole-app/"
    mkdir -p "${SCRIPT_DIR}/conf/tmp"

    if [ -d "$target_dir" ]; then
        (cd "$target_dir" && npm install > /dev/null 2>&1)
        cd "$target_dir" || exit
        nohup sh -c "NAMESPACE=$NAMESPACE npm start" > "${BLACKHOLE_LOG_FILE}" 2>&1 & echo $! > "${BLACKHOLE_PID_FILE}"
        echo "Started npm start in directory '$target_dir' with output redirected to '${BLACKHOLE_LOG_FILE}' and PID stored in '${BLACKHOLE_PID_FILE}'."
        cd "$SCRIPT_DIR" || exit
    else
        echo "Directory $target_dir does not exist."
    fi
}

## Main script execution
kubectl create namespace $NAMESPACE
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
update_coredns_configmap
process_ingress_files
deploy_boutique
deploy_single
deploy_plan_client
## alternative to deploy_plan_client - for custom or local-only plan not in public repo yet
## comment-out deploy_plan_client and uncomment the following line
#helm upgrade --install --create-namespace --namespace $NAMESPACE  --set image.tag=$IMAGE_TAG  --set chaos.host=frontend  --set chaos.plan="plans/blackhole_access.yaml" --set business_application=$NAMESPACE client $SCRIPT_DIR/../../helm/client
run_npm_commands