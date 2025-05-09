######################################
## Script Utility Vars and Functions
######################################

## ############ Colors ############ ##
# Color codes
RED="31"
GREEN="32"
YELLOW="33"
BLUE="34"
MAGENTA="35"
CYAN="36"
WHITE="37"

# Text Colors
PROMPT_TXT=$RED
INFO_TXT=$YELLOW
DETAIL_TXT=$CYAN
EXAMPLE_TXT=$GREEN

## ############ Printing Functions ############ ##
print_in_color() {
    local color_code="$1"
    local text="$2"
    printf "\e[${color_code}m%s\e[0m\n" "$text"
}

# Function to display help message
show_help() {
    print_in_color "${DETAIL_TXT}" "Usage: $0 [options]"
    echo ""
    print_in_color "${DETAIL_TXT}" "Options:"
    print_in_color "${DETAIL_TXT}"  "  --namespace=NAMESPACE    Set the Kubernetes namespace"
    print_in_color "${INFO_TXT}"    "                       - (default) ${NAMESPACE}"
    exit 0
}


# Function to parse named arguments
parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            "--namespace"=*)
                NAMESPACE="${1#*=}"
                ;;
            --help)
                show_help
                ;;
            *)
                print_in_color "${PROMPT_TXT}" "Invalid argument: $1"
                show_help
                ;;
        esac
        shift
    done
}

# Function to check job status
check_job_status() {
    local namespace=$1
    local job_name=$2
    local max_attempts=${3:-60}  # Default to 60 attempts
    local interval=${4:-30}      # Default to 30 seconds between checks
    
    echo "Waiting for job $job_name to complete (checking every ${interval}s, max ${max_attempts} attempts)..."
    
    for ((i=1; i<=max_attempts; i++)); do
        # Check for completion
        if [ "$(kubectl get job $job_name -n $namespace -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}')" = "True" ]; then
            echo "✅ Job $job_name completed successfully!"
            return 0
        fi
        
        # Check for failure
        if [ "$(kubectl get job $job_name -n $namespace -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}')" = "True" ]; then
            echo "❌ Job $job_name failed!"
            return 1
        fi
        
        # If we haven't reached max attempts, wait and try again
        if [ $i -lt $max_attempts ]; then
            echo "Attempt $i/$max_attempts: Job still running..."
            sleep $interval
        fi
    done
    
    echo "⚠️ Job $job_name did not complete within the specified time limit"
    return 2
}

cleanup() {
    # Check if namespace provided
    if [ $# -ne 1 ] || [ -z "$1" ]; then
        echo "Error: Namespace not provided to cleanup."
    return 1
    fi
    local namespace=$1
    
    # Check if namespace exists
    if ! kubectl get namespace "$namespace" >/dev/null 2>&1; then
        echo "Error: Namespace $namespace does not exist."
        exit 1
    fi

    # Delete evicted pods
    kubectl delete pod -n "$namespace" --field-selector=status.phase=Failed || true

    # Delte other resources
    kubectl delete all --all -n "$namespace" || true
    kubectl delete configmap,secret,pvc,ingress --all -n "$namespace" || true

    # Delete Helm releases
    helm list -n "$namespace" --short | xargs -I {} helm uninstall {} -n "$namespace" || true

    # Delete the namespace
    kubectl delete namespace "$namespace" --grace-period=30

    echo "Namespace $namespace cleaned up, including evicted pods."
}

