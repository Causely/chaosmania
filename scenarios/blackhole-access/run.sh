#!/bin/bash

source ./common_vars.sh

# Function to display help message
show_help() {
    print_in_color "${DETAIL_TXT}" "Usage: $0 [options]"
    echo ""
    print_in_color "${DETAIL_TXT}" "Options:"

    print_in_color "${DETAIL_TXT}"  "  --run-external-svc=RUN_NODEJS          Set the environment. Possible values:"
    print_in_color "${INFO_TXT}"    "                       - (default) ${RUN_NODEJS}"
    print_in_color "${INFO_TXT}"    "                       - \"false\": do not run nodejs external service; provide own implementation"
    print_in_color "${INFO_TXT}"    "                       - \"true\": runs external nodejs proxy application locally"

    print_in_color "${DETAIL_TXT}"  "  --custom-coredns=true          Boolean flag to apply custom CoreDNS configuration"
    print_in_color "${INFO_TXT}"    "                       - (default) ${CUSTOM_COREDNS}"
    print_in_color "${INFO_TXT}"    "                       - \"false\": ignore custom CoreDNS configuration"
    print_in_color "${INFO_TXT}"    "                       - \"true\": apply custom CoreDNS configuration from: ${COREDNS_CM_COPY}"

    print_in_color "${DETAIL_TXT}"  "  --custom-ingress=true          Boolean flag to apply custom K8s Ingress rules"
    print_in_color "${INFO_TXT}"    "                       - (default) ${CUSTOM_INGRESS}"
    print_in_color "${INFO_TXT}"    "                       - \"false\": ignore custom Ingress rules"
    print_in_color "${INFO_TXT}"    "                       - \"true\": apply custom Ingress rule(s) from: ${INGRESS_DIR}/*_ingress.tpl.yaml"


    print_in_color "${DETAIL_TXT}"  "  --plan=PLAN_YAML          Set the path to the plan file"
    print_in_color "${INFO_TXT}"    "                       - (default) ${PLAN_YAML}"


    print_in_color "${DETAIL_TXT}"  "  --namespace=NAMESPACE    Set the Kubernetes namespace"
    print_in_color "${INFO_TXT}"    "                       - (default) ${NAMESPACE}"

    print_in_color "${DETAIL_TXT}"  "  --image-tag=IMAGE_TAG     Set the Docker image tag"
    print_in_color "${INFO_TXT}"    "                       - (default) ${IMAGE_TAG}"

    print_in_color "${DETAIL_TXT}"  "  --otlp-collector=OTLP_COLLECTOR     Set the url of the OTLP collector endpoint"
    print_in_color "${INFO_TXT}"    "                       - (default) ${OTLP_COLLECTOR}"

    print_in_color "${DETAIL_TXT}"  "  --help              Display this help message and exit"
    echo ""

    print_in_color "${EXAMPLE_TXT}" "Examples:"
    print_in_color "${INFO_TXT}"    "Local NodeJS external service with custom coredns & Ingress rules:"
    print_in_color "${EXAMPLE_TXT}" "  $0 --run-external-svc=\"true\" --custom-coredns=\"true\" --custom-ingress=\"true\" "
    echo ""
    print_in_color "${INFO_TXT}"    "Using own external Load Balancer (no NodeJS, No Ingress rules, No CoreDNS overrides):"
    print_in_color "${EXAMPLE_TXT}"    "./run.sh  (no args, but plan.yaml must point to external ALB)"
    echo ""
    print_in_color "${INFO_TXT}"    "Other Options"
    print_in_color "${EXAMPLE_TXT}" "  $0 --namespace=my-namespace --image-tag=v1.0.0"
    print_in_color "${EXAMPLE_TXT}" "  $0 --otlp-collector=http://opentelemetry-collector.default:4318"
    print_in_color "${EXAMPLE_TXT}" "  $0 --plan=/plans/some-other-plan.yaml"

    exit 0
}

# Function to parse named arguments
parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            "--run-external-svc"=*)
                RUN_NODEJS="${1#*=}"
                ;;
            "--namespace"=*)
                NAMESPACE="${1#*=}"
                ;;
            "--image-tag"=*)
                IMAGE_TAG="${1#*=}"
                ;;
            "--otlp-collector"=*)
                OTLP_COLLECTOR="${1#*=}"
                ;;
            "--plan"=*)
                PLAN_YAML="${1#*=}"
                ;;
            "--custom-coredns"=*)
                CUSTOM_COREDNS="${1#*=}"
                ;;
            "--custom-ingress"=*)
                CUSTOM_INGRESS="${1#*=}"
                ;;
            --help)
                show_help
                ;;
            *)
                print_in_color "${PROMPT_TXT}" "Invalid argument: $1"
                show_help
                ;;
        esac
        shift
    done
}

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
        for file in "$dir"/*ingress.tpl.yaml; do
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
  local helm_delete_command="helm delete --namespace ${NAMESPACE} boutique"
  local helm_upgrade_command="helm upgrade --install --create-namespace --namespace ${NAMESPACE} \
      --set image.tag=${IMAGE_TAG} \
      --set business_application=${NAMESPACE} \
      --set otlp.enabled=true \
      --set otlp.endpoint=${OTLP_COLLECTOR} \
      boutique ${SCRIPT_DIR}/../../helm/boutique"

  print_in_color "${PROMPT_TXT}" "${helm_delete_command}"
  eval "${helm_delete_command}"

  # Execute the command
  print_in_color "${INFO_TXT}" "${helm_upgrade_command}"
  eval "${helm_upgrade_command}"
}

deploy_single() {
    local helm_delete_command="helm delete --namespace ${NAMESPACE} single"
    local helm_upgrade_command="helm upgrade --install --namespace ${NAMESPACE} \
        --set image.tag=${IMAGE_TAG} \
        --set replicaCount=1 \
        --set business_application=${NAMESPACE }\
        --set otlp.enabled=true \
        --set otlp.endpoint=${OTLP_COLLECTOR} \
        single ${SCRIPT_DIR}/../../helm/single"

    print_in_color "${PROMPT_TXT}" "${helm_delete_command}"
    eval "${helm_delete_command}"

    # Execute the command
    print_in_color "${INFO_TXT}" "${helm_upgrade_command}"
    eval "${helm_upgrade_command}"

}

deploy_plan_client() {
    local helm_delete_command="helm delete --namespace ${NAMESPACE} client"
    local helm_upgrade_command="helm upgrade --install --create-namespace --namespace ${NAMESPACE} \
      --set image.tag=${IMAGE_TAG} \
      --set chaos.host=frontend \
      --set chaos.plan=${PLAN_YAML} \
      --set business_application=${NAMESPACE} \
      client ${SCRIPT_DIR}/../../helm/client"

    print_in_color "${PROMPT_TXT}" "${helm_delete_command}"
    eval "${helm_delete_command}"

    # Execute the command
    print_in_color "${INFO_TXT}" "${helm_upgrade_command}"
    eval "${helm_upgrade_command}"
}

run_npm_commands() {
    local target_dir="${SCRIPT_DIR}/external-services/blackhole-app/"
    mkdir -p "${SCRIPT_DIR}/conf/tmp"

    if [ -d "$target_dir" ]; then
        (cd "$target_dir" && npm install > /dev/null 2>&1)
        cd "$target_dir" || exit
        nohup sh -c "NAMESPACE=${NAMESPACE} npm start" > "${BLACKHOLE_LOG_FILE}" 2>&1 & echo $! > "${BLACKHOLE_PID_FILE}"
        echo "Started npm start in directory '$target_dir' with output redirected to '${BLACKHOLE_LOG_FILE}' and PID stored in '${BLACKHOLE_PID_FILE}'."
        cd "${SCRIPT_DIR}" || exit
    else
        echo "Directory $target_dir does not exist."
    fi
}

# Call the argument parsing function with all script arguments
parse_args "$@"

## Main script execution
kubectl create namespace "${NAMESPACE}"
kubectl label namespace "${NAMESPACE}" istio-injection=enabled --overwrite

if [ "$CUSTOM_COREDNS" == "true" ]; then
  print_in_color $INFO_TXT "Applying custom CoreDNS configmap"
  update_coredns_configmap
fi

if [ "$CUSTOM_INGRESS" == "true" ]; then
    print_in_color $INFO_TXT "Applying Ingress rules"
    process_ingress_files
fi

if [ "$RUN_NODEJS" == "true" ]; then
    print_in_color $INFO_TXT "Running nodejs service"
    run_npm_commands
fi

deploy_boutique
deploy_single
deploy_plan_client