#!/usr/bin/env bash

set -Eeuo pipefail

###############################################################################

# Configuration

###############################################################################

VM_NAME="i-master"
CPU="2"
MEMORY="2GiB"
IMAGE="images:ubuntu/questing/cloud"

usage() {
cat <<EOF
Usage: $0 [options]

Options:
--cpu <count>        vCPU count (default: 2)
--memory <size>      Memory size (default: 2GiB)
--name <name>        VM name (default: i-master)

Examples:
$0
$0 --cpu 4 --memory 8GiB
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --cpu)
            CPU="$2"
            shift 2
        ;;
        --memory)
            MEMORY="$2"
            shift 2
        ;;
        --name)
            VM_NAME="$2"
            shift 2
        ;;
        -h|--help)
            usage
            exit 0
        ;;
        *)
            echo "Unknown argument: $1"
            exit 1
        ;;
    esac
done

###############################################################################

# Helpers

###############################################################################

log() {
    echo
    echo "===================================================="
    echo "$*"
    echo "===================================================="
}

wait_for_vm() {
    log "Waiting for VM to start"
    
    ```
    while true; do
        status=$(incus list "$VM_NAME" --format csv -c s 2>/dev/null || true)
        
        if [[ "$status" == "RUNNING" ]]; then
            break
        fi
        
        sleep 2
    done
    
    echo "VM is running"
    ```
    
}

wait_for_cloud_init() {
    log "Waiting for cloud-init"
    
    ```
    incus exec "$VM_NAME" -- cloud-init status --wait
    
    echo "Cloud-init completed"
    ```
    
}

exec_vm() {
    incus exec "$VM_NAME" -- bash -c "$1"
}

###############################################################################

# Create VM

###############################################################################

log "Creating VM $VM_NAME"

if incus info "$VM_NAME" >/dev/null 2>&1; then
    echo "VM already exists: $VM_NAME"
    exit 1
fi

incus launch "$IMAGE" "$VM_NAME"
--vm
-c limits.cpu="$CPU"
-c limits.memory="$MEMORY"

wait_for_vm
wait_for_cloud_init

###############################################################################

# Base packages

###############################################################################

log "Updating apt repositories"

exec_vm "
apt-get update
"

log "Installing prerequisites"

exec_vm "
DEBIAN_FRONTEND=noninteractive apt-get install -y
curl
wget
gpg
ca-certificates
apt-transport-https
software-properties-common
containerd
"

###############################################################################

# Containerd

###############################################################################

log "Configuring containerd"

exec_vm "
mkdir -p /etc/containerd

containerd config default > /etc/containerd/config.toml

sed -i
's/SystemdCgroup = false/SystemdCgroup = true/'
/etc/containerd/config.toml

systemctl enable containerd
systemctl restart containerd
"

###############################################################################

# Kernel + sysctl

###############################################################################

log "Configuring kernel modules"

exec_vm "
cat >/etc/modules-load.d/k8s.conf <<EOF
overlay
br_netfilter
EOF

modprobe overlay
modprobe br_netfilter
"

log "Configuring sysctl"

exec_vm "
touch /etc/sysctl.conf

grep -q '^net.ipv4.ip_forward=' /etc/sysctl.conf
&& sed -i 's/^net.ipv4.ip_forward=.*/net.ipv4.ip_forward=1/' /etc/sysctl.conf
|| echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf

grep -q '^net.bridge.bridge-nf-call-iptables=' /etc/sysctl.conf
|| echo 'net.bridge.bridge-nf-call-iptables=1' >> /etc/sysctl.conf

grep -q '^net.bridge.bridge-nf-call-ip6tables=' /etc/sysctl.conf
|| echo 'net.bridge.bridge-nf-call-ip6tables=1' >> /etc/sysctl.conf

sysctl -p
"

###############################################################################

# Kubernetes repo

###############################################################################

log "Installing Kubernetes repository"

exec_vm "
mkdir -p /etc/apt/keyrings

curl -fsSL
https://pkgs.k8s.io/core:/stable:/v1.34/deb/Release.key
| gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

chmod 644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo
'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.34/deb/ /' \

> /etc/apt/sources.list.d/kubernetes.list

apt-get update
"

###############################################################################

# kubeadm

###############################################################################

log "Installing kubeadm components"

exec_vm "
DEBIAN_FRONTEND=noninteractive apt-get install -y
kubelet
kubeadm
kubectl

apt-mark hold kubelet kubeadm kubectl

systemctl enable kubelet
"

###############################################################################

# Disable swap

###############################################################################

log "Disabling swap"

exec_vm "
swapoff -a || true

sed -ri '/\sswap\s/s/^#?/#/' /etc/fstab || true
"

###############################################################################

# Get IP

###############################################################################

log "Discovering VM IP"

MASTER_IP=$(incus list "$VM_NAME" --format csv -c 4 | awk '{print $1}')

echo "Master IP: $MASTER_IP"

###############################################################################

# Cluster init

###############################################################################

log "Initializing Kubernetes cluster"

exec_vm "
kubeadm init
--apiserver-advertise-address=${MASTER_IP}
--pod-network-cidr=10.244.0.0/16
"

###############################################################################

# kubeconfig

###############################################################################

log "Configuring kubectl"

exec_vm "
mkdir -p /root/.kube

cp /etc/kubernetes/admin.conf /root/.kube/config

chown root:root /root/.kube/config
"

###############################################################################

# Wait for API

###############################################################################

log "Waiting for API server"

until incus exec "$VM_NAME" -- bash -c
"export KUBECONFIG=/etc/kubernetes/admin.conf && kubectl get nodes >/dev/null 2>&1"
do
    sleep 5
done

###############################################################################

# Helm

###############################################################################

log "Installing Helm"

exec_vm "
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
"

###############################################################################

# Cilium

###############################################################################

log "Installing Cilium"

exec_vm "
export KUBECONFIG=/etc/kubernetes/admin.conf

helm repo add cilium https://helm.cilium.io
helm repo update

helm upgrade --install cilium cilium/cilium
--namespace kube-system
"

###############################################################################

# Wait for node

###############################################################################

log "Waiting for node readiness"

exec_vm "
export KUBECONFIG=/etc/kubernetes/admin.conf

kubectl wait
--for=condition=Ready
node/$(hostname)
--timeout=15m
"

###############################################################################

# Wait for Cilium

###############################################################################

log "Waiting for Cilium"

exec_vm "
export KUBECONFIG=/etc/kubernetes/admin.conf

kubectl rollout status
daemonset/cilium
-n kube-system
--timeout=15m
"

###############################################################################

# Cluster status

###############################################################################

log "Cluster status"

exec_vm "
export KUBECONFIG=/etc/kubernetes/admin.conf

kubectl get nodes -o wide
echo
kubectl get pods -A
"

###############################################################################

# Join command

###############################################################################

log "Worker join command"

JOIN_COMMAND=$(incus exec "$VM_NAME" -- bash -c
"export KUBECONFIG=/etc/kubernetes/admin.conf && kubeadm token create --print-join-command")

echo
echo "$JOIN_COMMAND"
echo

log "Cluster is ready"
