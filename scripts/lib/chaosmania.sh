#!/bin/bash

# Common variables
IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest

# Parse common command line arguments
parse_args() {
    PREFIX_USER=false
    REPEATS_PER_PHASE=""
    PHASE_PATTERN=""
    RUNTIME_DURATION=""

    for arg in "$@"; do
        case $arg in
            --prefix-user)
                export PREFIX_USER=true
                shift
                ;;
            --repeats-per-phase)
                export REPEATS_PER_PHASE=$2
                shift 2
                ;;
            --phase-pattern)
                export PHASE_PATTERN=$2
                shift 2
                ;;
            --runtime-duration)
                export RUNTIME_DURATION=$2
                shift 2
                ;;
            --otlp-endpoint)
                export OTLP_ENDPOINT=$2
                shift 2
                ;;
        esac
    done
}

# Setup namespace
setup_namespace() {
    local SCENARIO=$1
    salt=$(uuidgen | cut -d '-' -f 1 | tr '[:upper:]' '[:lower:]')

    if [ "$PREFIX_USER" = true ]; then
        export NAMESPACE=$USER-$SCENARIO
    else
        export NAMESPACE=$SCENARIO
    fi

    export NAMESPACE=$NAMESPACE-$salt

    echo "Creating namespace $NAMESPACE"
    kubectl create namespace $NAMESPACE || true

    echo "Labeling namespace $NAMESPACE for Istio injection"
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite || true
}

# Build client arguments
build_client_args() {
    local CLIENT_ARGS=""
    if [ ! -z "$REPEATS_PER_PHASE" ]; then
        CLIENT_ARGS="$CLIENT_ARGS --set chaos.repeats_per_phase=$REPEATS_PER_PHASE"
    fi
    if [ ! -z "$PHASE_PATTERN" ]; then
        CLIENT_ARGS="$CLIENT_ARGS --set chaos.phase_pattern=$PHASE_PATTERN"
    fi
    if [ ! -z "$RUNTIME_DURATION" ]; then
        CLIENT_ARGS="$CLIENT_ARGS --set chaos.runtime_duration=$RUNTIME_DURATION"
    fi
    if [ ! -z "$OTLP_ENDPOINT" ]; then
        CLIENT_ARGS="$CLIENT_ARGS --set otlp.endpoint=$OTLP_ENDPOINT"
    fi
    echo "$CLIENT_ARGS"
}

# Common helm upgrade command for single deployment
upgrade_single() {
    local DEPLOYMENT_NAME=$1
    local NAMESPACE=$2
    local SCENARIO=$3
    local SCRIPT_DIR=$4
    shift 4  # Remove the first 4 arguments
    local EXTRA_ARGS=("$@")  # Capture all remaining arguments as an array

    echo "Deploying $DEPLOYMENT_NAME"
    helm upgrade --install --namespace $NAMESPACE \
        --set image.tag=$IMAGE_TAG \
        --set business_application=$SCENARIO \
        --set otlp.enabled=true \
        "${EXTRA_ARGS[@]}" \
        $DEPLOYMENT_NAME $SCRIPT_DIR/../../helm/single
}

# Common helm upgrade command for client deployment
upgrade_client() {
    local NAMESPACE=$1
    local SCENARIO=$2
    local SCRIPT_DIR=$3
    local CLIENT_NAME=$4
    local CHAOS_HOST=$5
    local PLAN_PATH=$6
    shift 6  # Remove the first 6 arguments
    local EXTRA_ARGS=${7:-}

    echo "Deploying client $CLIENT_NAME"
    helm delete --namespace $NAMESPACE $CLIENT_NAME || true
    helm upgrade --install --namespace $NAMESPACE \
        --set image.tag=$IMAGE_TAG \
        --set chaos.host=$CHAOS_HOST \
        --set chaos.plan=$PLAN_PATH \
        --set business_application=$SCENARIO \
        --set otlp.enabled=true \
        $(build_client_args) \
        $EXTRA_ARGS \
        $CLIENT_NAME $SCRIPT_DIR/../../helm/client
}

# Debug mode - set to true to enable command echoing
DEBUG=${DEBUG:-false}
if [ "$DEBUG" = true ]; then
    PS4='+(${BASH_SOURCE}:${LINENO}): '
    set -x
fi
