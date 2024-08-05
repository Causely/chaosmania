#!/bin/bash

source ./common_vars.sh

# Function to kill the background process
kill_npm_process() {
    if [ -f "$BLACKHOLE_PID_FILE" ]; then
        local pid=$(cat "${BLACKHOLE_PID_FILE}")
        if kill -0 "$pid" > /dev/null 2>&1; then
            kill "$pid"
            echo "Process with PID $pid has been terminated."
            rm "${BLACKHOLE_PID_FILE}"
            rm "${BLACKHOLE_LOG_FILE}"
        else
            echo "No process found with PID $pid. It might have already been terminated."
        fi
    else
        echo "PID file '$BLACKHOLE_PID_FILE' does not exist."
    fi
}

delete_chaos_ingress() {
    local dir="${INGRESS_DIR}"

    if [ -d "$dir" ]; then
        for file in "$dir"/*ingress.yaml; do
            if [ -f "$file" ]; then
                echo "Deleteing Ingress for file: $(basename $file)"
                kubectl -n "${NAMESPACE}" delete -f "$file"
            else
                echo "No '*ingress.yaml' files found in $dir ... continuing"
            fi
        done
    else
        echo "Directory $dir not found... continuing"
    fi
}

helm_delete() {
  helm delete --namespace $NAMESPACE boutique
  helm delete --namespace $NAMESPACE single
  helm delete --namespace $NAMESPACE client
}

# Main script execution
kill_npm_process
delete_chaos_ingress
helm_delete
