#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source $SCRIPT_DIR/../../scripts/lib/chaosmania.sh

# Parse arguments
parse_args "$@"

# Setup scenario
SCENARIO=cm-chained-services-db-unavailable

# Setup namespace
setup_namespace $SCENARIO

# Deploy single instance
upgrade_single "frontend" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=2"
upgrade_single "payment" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=2"

# Do not deploy DB

# Deploy client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client" "frontend" "/scenarios/$SCENARIO-plan.yaml"
