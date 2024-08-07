#!/bin/bash

source ./common_vars.sh

# Function to display help message
show_help() {
    print_in_color "${DETAIL_TXT}" "Usage: $0 [options]"
    echo ""
    print_in_color "${DETAIL_TXT}" "Options:"

    print_in_color "${DETAIL_TXT}"  "  --namespace=NAMESPACE    Set the Kubernetes namespace"
    print_in_color "${INFO_TXT}"    "                       - (default) ${NAMESPACE}"

    print_in_color "${DETAIL_TXT}"  "  --help              Display this help message and exit"
    echo ""

    print_in_color "${EXAMPLE_TXT}" "Examples:"
    print_in_color "${EXAMPLE_TXT}" "  $0 --namespace=my-namespace"

    exit 0
}

# Function to parse named arguments
parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --namespace=*)
                NAMESPACE="${1#*=}"
                ;;
            --help)
                show_help
                ;;
            *)
                echo "Invalid argument: $1"
                show_help
                ;;
        esac
        shift
    done
}

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
        for file in "$dir"/*ingress.tpl.yaml; do
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
  local helm_delete_boutique="helm delete --namespace ${NAMESPACE} boutique"
  print_in_color "${PROMPT_TXT}" "${helm_delete_boutique}"
  eval "${helm_delete_boutique}"

  local helm_delete_single="helm delete --namespace ${NAMESPACE} single"
  print_in_color "${PROMPT_TXT}" "${helm_delete_single}"
  eval "${helm_delete_single}"

  local helm_delete_client="helm delete --namespace ${NAMESPACE} client"
  print_in_color "${PROMPT_TXT}" "${helm_delete_client}"
  eval "${helm_delete_client}"
}

# Main script execution
kill_npm_process
delete_chaos_ingress
helm_delete
