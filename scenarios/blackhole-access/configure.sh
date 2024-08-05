#!/bin/bash

source ./common_vars.sh

# Function to display the contents of /etc/hosts
show_hosts() {
    print_in_color $INFO_TXT "Current contents of /etc/hosts:"
    cat_file_in_color "$HOSTS_FILE" "$DETAIL_TXT"
}

# Function to prompt user if they need to edit /etc/hosts
prompt_edit_hosts() {
    while true; do
        print_in_color $INFO_TXT "127.0.0.1 should contain hostname(s) of external services that you want to blackhole and K8s <service>.<namespace> return entries."
        print_in_color EXAMPLE_TXT "EXAMPLE:     127.0.0.1 shipping.blackhole-external shipping.blackhole-access"
        print_in_color $PROMT_TXT "Do you want to open $HOSTS_FILE for editing? (yes/no): "
        read -r choice
        case "$choice" in
            yes|y|Y )
                sudo ${EDITOR:-nano} "$HOSTS_FILE"
                break
                ;;
            no|n|N )
                print_in_color $INFO_TXT "Continuing without editing..."
                break
                ;;
            * )
                print_in_color $PROMT_TXT "Invalid input. Please enter yes or no."
                ;;
        esac
    done
}

# Function to display the CoreDNS ConfigMap
show_coredns_configmap() {
    print_in_color $INFO_TXT "Current CoreDNS ConfigMap:"
    print_in_color $DETAIL_TXT "$(kubectl -n kube-system get configmap coredns -o yaml)"
}

# Function to show the current local IP address
show_local_ip() {
    print_in_color $INFO_TXT "Current local IP address(es):"
    if command -v ip &> /dev/null; then
        ip addr show | grep 'inet ' | awk '{print $2}' | cut -d/ -f1
    else
        ifconfig | grep 'inet ' | awk '{print $2}'
    fi
}

# Function to prompt user for action on CoreDNS ConfigMap
prompt_edit_coredns() {
    while true; do
        print_in_color $INFO_TXT "CoreDNS ConfigMap should contain following block within '.:53 {' block:"
        print_in_color $EXAMPLE_TXT "hosts custom.hosts shipping.chaos {
             <local-network-ip> shipping.chaos
             fallthrough
           }"
        show_local_ip
        print_in_color $PROMT_TXT "Do you want to create a copy of CoreDNS ConfigMap to editing? (yes/no): "
        read -r choice
        case "$choice" in
            yes|y|Y )
                print_in_color $INFO_TXT "Copy current kube-system/coredns ConfigMap to $COREDNS_CM_COPY:"
                kubectl -n kube-system get configmap coredns -o yaml > "${COREDNS_CM_COPY}"
                kubectl -n kube-system get configmap coredns -o yaml > "${COREDNS_ORIG_COPY}"
                ${EDITOR:-nano} "$COREDNS_CM_COPY"
                break
                ;;
            no|n|N )
                echo "Continuing without editing..."
                break
                ;;
            * )
                echo "Invalid input. Please enter yes or no."
                ;;
        esac
    done
}

# Function to iterate over each YAML file in a directory and present a prompt
process_ingress_files() {
    local dir="${SCRIPT_DIR}/../../kubernetes/blackhole"

    if [ -d "$dir" ]; then
        for file in "$dir"/*ingress.yaml; do
            if [ -f "$file" ]; then
                filename=$(basename "$file")
                while true; do
                    print_in_color $PROMT_TXT "Do you want to include Ingress rule for $filename? (yes/no): "
                    read -r apply_choice
                    case "$apply_choice" in
                        yes|y|Y )
                            print_in_color $INFO_TXT "copying file: $filename"
                            ingress_copy="${INGRESS_DIR}/$filename"
                            cp "$file" "${ingress_copy}"
                            while true; do
                                cat_file_in_color "$file" "$DETAIL_TXT"
                                echo -n "Do you want to edit this $filename now? (yes/no): "
                                read -r edit_choice
                                case "$edit_choice" in
                                    yes|y|Y )
                                        ${EDITOR:-vi} "$ingress_copy"
                                        break
                                        ;;
                                    no|n|N )
                                        break
                                        ;;
                                    * )
                                        echo "Invalid input. Please enter yes or no."
                                        ;;
                                esac
                            done
                            break
                            ;;
                        no|n|N )
                            print_in_color $INFO_TXT "Skipping file: $filename"
                            break
                            ;;
                        * )
                            echo "Invalid input. Please enter yes or no."
                            ;;
                    esac
                done
            else
                echo "No YAML files found in the directory."
                break
            fi
        done
    else
        echo "Directory $dir does not exist."
    fi
}


# Main script execution
static_dirs=(
    "${COREDNS_DIR}"
    "${INGRESS_DIR}"
)
make_dirs_if_not_exist "${static_dirs[@]}"

show_hosts
prompt_edit_hosts
show_coredns_configmap
prompt_edit_coredns
process_ingress_files