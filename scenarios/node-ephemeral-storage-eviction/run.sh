#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source $SCRIPT_DIR/../../scripts/lib/chaosmania.sh

# Parse arguments
parse_args "$@"

# Setup scenario
SCENARIO=cm-node-ephemeral-storage-eviction

# Setup namespace
setup_namespace $SCENARIO

# Deploy single instance
upgrade_single "single" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "resources.limits.ephemeral-storage=256Mi" "--set" "securityContext.readOnlyRootFilesystem=false" "--set" "replicaCount=3"

# Deploy client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client" "single" "/scenarios/$SCENARIO-plan.yaml"
