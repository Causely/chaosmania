
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
IMAGE_REPO=quay.io/causely/chaosmania
IMAGE_TAG="SNAPSHOT"
NAMESPACE=blackhole-access
OTLP_COLLECTOR="http://opentelemetry-collector.default:4318"

HOSTS_FILE="/etc/hosts"

K8S_TEMPLATES_DIR="${SCRIPT_DIR}/conf/templates"

COREDNS_DIR="${SCRIPT_DIR}/conf/coredns"
INGRESS_DIR="${SCRIPT_DIR}/conf/ingress"
COREDNS_CM_COPY="${COREDNS_DIR}/coredns-configmap.yaml"
timestamp=$(date +"%Y-%m-%d_%H-%M-%S")
COREDNS_ORIG_COPY="${COREDNS_DIR}/origin-coredns-configmap_${timestamp}.yaml"

TMP_DIR="${SCRIPT_DIR}/conf/tmp"
BLACKHOLE_PID_FILE="${TMP_DIR}/npm_process.pid"
BLACKHOLE_LOG_FILE="${TMP_DIR}/npm_log.log"

RUN_NODEJS="false"
CUSTOM_COREDNS="false"
CUSTOM_INGRESS="false"
PLAN_YAML="/scenarios/cm-$NAMESPACE-plan.yaml"

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

print_file_in_color() {
    local file="$1"
    local color_code="${2:-$BLUE}"
    while IFS= read -r line; do
      print_in_color "$color_code" "$line"
    done < "$file"
}

cat_file_in_color() {
    local file="$1"
    local color_code="${2:-$BLUE}"
    printf "\e[${color_code}m%s"
    cat "$file"
    printf "\e[0m"
}

## ############ Filesys Functions ############ ##
make_dirs_if_not_exist() {
    local dirs=("$@")
    for dir in "${dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            print_in_color $INFO_TXT "Creating directory: $dir"
            mkdir -p "$dir"
        else
            print_in_color $INFO_TXT "Directory already exists: $dir"
        fi
    done
}