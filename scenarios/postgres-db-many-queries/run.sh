#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source $SCRIPT_DIR/../../scripts/lib/chaosmania.sh

# Parse arguments
parse_args "$@"

# Setup scenario
SCENARIO=cm-postgres-db-many-queries

# Setup namespace
setup_namespace $SCENARIO

# Deploy single instance
upgrade_single "single" $NAMESPACE $SCENARIO $SCRIPT_DIR "--set" "replicaCount=1"

# Deploy client
upgrade_client $NAMESPACE $SCENARIO $SCRIPT_DIR "client" "single" "/scenarios/$SCENARIO-plan.yaml"
