#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

$SCRIPT_DIR/chained-services/run.sh
# $SCRIPT_DIR/chained-virtual-services/run.sh
$SCRIPT_DIR/client-throttled/run.sh
$SCRIPT_DIR/client-unauthorized/run.sh
$SCRIPT_DIR/container-cpu-congestion/run.sh
$SCRIPT_DIR/container-data-disk-tp-throttled/run.sh
$SCRIPT_DIR/ephemeral-storage-eviction/run.sh
$SCRIPT_DIR/ndots-nxdomain/run.sh
# $SCRIPT_DIR/node-cpu-congestion/run.sh
# $SCRIPT_DIR/node-ephemeral-storage-eviction/run.sh
# $SCRIPT_DIR/node-oom-eviction/run.sh
$SCRIPT_DIR/noisy-client-cpu-congestion/run.sh
$SCRIPT_DIR/periodical-crash/run.sh
$SCRIPT_DIR/periodical-oom/run.sh
