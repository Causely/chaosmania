#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source $SCRIPT_DIR/../../scripts/lib/chaosmania.sh

# Parse arguments
parse_args "$@"

# Setup scenario
SCENARIO=cm-client-unauthorized

# Setup namespace
setup_namespace $SCENARIO

# Deploy single instance
upgrade_single "single" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "resources.limits.cpu=500m" "--set" "replicaCount=3"

# Deploy client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client-unauthorized" "single" "/scenarios/$SCENARIO-unauthorized.yaml"

# Deploy non-authorized client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client-not-unauthorized" "single" "/scenarios/$SCENARIO-not_unauthorized.yaml"
