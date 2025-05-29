#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source $SCRIPT_DIR/../../scripts/lib/chaosmania.sh

# Parse arguments
parse_args "$@"

# Setup scenario
SCENARIO=cm-noisy-client-cpu-congestion

# Setup namespace
setup_namespace $SCENARIO

# Deploy single instance
upgrade_single "single" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "resources.limits.cpu=500m" "--set" "replicaCount=3"

# Deploy client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client1" "single" "/scenarios/$SCENARIO-plan1.yaml"
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client2" "single" "/scenarios/$SCENARIO-plan2.yaml"
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client3" "single" "/scenarios/$SCENARIO-plan3.yaml"
