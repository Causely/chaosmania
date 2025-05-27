#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source $SCRIPT_DIR/../../scripts/lib/chaosmania.sh

# Parse arguments
parse_args "$@"

# Setup scenario
SCENARIO=cm-client-throttled

# Setup namespace
setup_namespace $SCENARIO

# Deploy single instance
upgrade_single "single" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "resources.limits.cpu=500m" "--set" "replicaCount=3"

# Deploy throttled client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client-throttled" "single" "/scenarios/$SCENARIO-throttled.yaml"

# Deploy non-throttled client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client-not-throttled" "single" "/scenarios/$SCENARIO-not_throttled.yaml"

