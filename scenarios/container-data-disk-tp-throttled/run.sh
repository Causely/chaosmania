#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source $SCRIPT_DIR/../../scripts/lib/chaosmania.sh

# Parse arguments
parse_args "$@"

# Setup scenario
SCENARIO=cm-container-data-disk-tp-throttled

# Setup namespace
setup_namespace $SCENARIO

# Deploy single instance
upgrade_single "single" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=1" "--set" "persistence.enabled=true" "--set" "persistence.size=100Gi"

# Deploy client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client" "single" "/scenarios/$SCENARIO-plan.yaml"
