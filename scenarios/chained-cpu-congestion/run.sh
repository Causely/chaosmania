#!/bin/sh

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG=latest
SCENARIO=cm-chained-cpu-congestion
NAMESPACE=$USER-$SCENARIO

echo "Creating namespace $NAMESPACE"
kubectl create namespace $NAMESPACE || true

echo "Labeling namespace $NAMESPACE for Istio injection"
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite || true


echo "Deploying frontend"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$SCENARIO \
    frontend $SCRIPT_DIR/../../helm/single 

echo "Deploying payment"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=1 \
    --set resources.limits.cpu="1000m"\
    --set business_application=$SCENARIO \
    payment-service $SCRIPT_DIR/../../helm/single 

echo "Deploying orders"
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set replicaCount=2 \
    --set business_application=$SCENARIO \
    order-service $SCRIPT_DIR/../../helm/single 

echo "Deploying client"
helm delete --namespace $NAMESPACE client || true
helm upgrade --install --namespace $NAMESPACE \
    --set image.tag=$IMAGE_TAG \
    --set chaos.host=frontend \
    --set chaos.plan=/scenarios/$SCENARIO-plan.yaml \
    --set business_application=$SCENARIO \
    client $SCRIPT_DIR/../../helm/client

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

# Check job status with default values (30 minutes total wait time)
check_job_status $NAMESPACE client

# Ask user if they want to run the scenario again
read -p "Would you like to run this scenario again? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Running scenario again..."
    exec $0  # Re-execute the script
fi

