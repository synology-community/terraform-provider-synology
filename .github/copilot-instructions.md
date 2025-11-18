---
applyTo: '**'
---
# Synology DSM 7.2 Kubernetes Deployment Troubleshooting Guide

## Executive Summary

We've attempted to run various Kubernetes distributions on Synology DSM 7.2 and encountered consistent system-level limitations that prevent modern Kubernetes from running properly in containerized environments on this platform.

## Failed Attempts Summary

### 1. Talos Linux
**Issues Encountered:**
- `opentree` syscall not implemented (DSM kernel too old)
- `nftables` functionality restricted
- Missing `/proc/sys/net/ipv4/tcp_keepalive_*` parameters
- Shadow bind mount failures

**Root Cause:** Talos expects bare-metal/VM environment with newer kernel features

### 2. K3s (Multiple Versions)
**Issues Encountered:**
- **Overlayfs:** "no such device" - overlayfs kernel module unavailable
- **Fuse-overlayfs:** Missing `mount.fuse3` executable in container
- **Pids cgroup controller:** Fatal error - controller not available and cgroup fs is read-only
- **Bridge netfilter:** `/proc/sys/net/bridge/bridge-nf-call-iptables` missing

**Versions Attempted:**
- `rancher/k3s:latest` 
- `rancher/k3s:v1.26.15-k3s1`
- `rancher/k3s:v1.21.14-k3s1` 
- `rancher/k3s:v1.20.15-k3s1`

### 3. K0s (Multiple Versions)
**Issues Encountered:**
- **Runtime detection:** "unknown runtime type unix, must be either of remote or docker"
- **Overlayfs failures:** Same storage snapshotter issues as K3s
- **Pids cgroup controller:** Same fatal limitation

**Versions Attempted:**
- `k0sproject/k0s:latest`
- `k0sproject/k0s:v1.28.4-k0s.0`
- `k0sproject/k0s:v1.27.7-k0s.0`
- `k0sproject/k0s:v1.23.17-k0s.0`

### 4. Kind (Kubernetes in Docker)
**Issues Encountered:**
- Container exits after "fixing cgroup mounts for all subsystems"
- `/proc/self/uid_map` missing
- cgroupns limitations

### 5. Other Attempts
**MicroK8s:** Repository access issues, incorrect image paths
**Minimal API Server:** Deprecated flags, port restrictions

## Core System Limitations on Synology DSM 7.2

### 1. Cgroup Controller Limitations
```bash
# Available controllers (missing 'pids'):
mount | grep cgroup
# Shows: cpu, freezer, blkio, devices, memory, cpuacct, cpuset
# Missing: pids (CRITICAL for modern Kubernetes)
```

### 2. Read-only Cgroup Filesystem
```bash
mkdir /sys/fs/cgroup/pids
# ERROR: Read-only file system
```

### 3. Kernel Module Limitations
```bash
modprobe overlay
# May not be available or functional in containers
```

### 4. Syscall Limitations
- `opentree` not implemented (newer Linux feature)
- Limited container runtime detection capabilities

## Comprehensive Testing Strategy

### SSH Connection
```bash
ssh root@nas.local
# Or use your specific Synology IP
ssh root@192.168.1.xxx
```

## Testing Matrix - All Combinations

### Phase 1: Storage Snapshotter Testing
Test all K8s distributions with different storage backends:

#### K3s Comprehensive Testing
```bash
# Test 1: K3s + Native Snapshotter + Old Version
docker run --privileged -d \
  --name k3s-test1 \
  -p 6443:6443 \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  rancher/k3s:v1.19.16-k3s1 \
  server --snapshotter=native --disable=traefik,servicelb,metrics-server

# Wait 5 minutes, check logs
docker logs k3s-test1 --tail 50

# Test 2: K3s + Native + Even Older
docker rm -f k3s-test1
docker run --privileged -d \
  --name k3s-test2 \
  -p 6443:6443 \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  rancher/k3s:v1.18.20-k3s1 \
  server --snapshotter=native --disable=traefik,servicelb,metrics-server

# Test 3: K3s + Host Networking
docker rm -f k3s-test2
docker run --privileged -d \
  --name k3s-test3 \
  --network=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  rancher/k3s:v1.20.15-k3s1 \
  server --snapshotter=native --disable=traefik,servicelb

# Test 4: K3s + Maximum Privilege Escalation
docker rm -f k3s-test3
docker run --privileged -d \
  --name k3s-test4 \
  -p 6443:6443 \
  --cap-add=ALL \
  --security-opt apparmor=unconfined \
  --security-opt seccomp=unconfined \
  --cgroupns=host \
  --pid=host \
  -v /:/host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  -v /proc:/proc \
  -v /dev:/dev \
  rancher/k3s:v1.21.14-k3s1 \
  server --snapshotter=native --disable=traefik,servicelb
```

#### K0s Comprehensive Testing
```bash
# Test 5: K0s + Old Version + Custom Config
docker rm -f k3s-test4
cat > k0s-minimal.yaml << 'EOF'
apiVersion: k0s.k0sproject.io/v1beta1
kind: ClusterConfig
spec:
  images:
    default_pull_policy: IfNotPresent
EOF

docker run --privileged -d \
  --name k0s-test5 \
  -p 6443:6443 \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  -v $(pwd)/k0s-minimal.yaml:/etc/k0s/k0s.yaml \
  k0sproject/k0s:v1.21.14-k0s.0 \
  k0s controller --single --config=/etc/k0s/k0s.yaml

# Test 6: K0s + Host Everything
docker rm -f k0s-test5
docker run --privileged -d \
  --name k0s-test6 \
  --network=host \
  --cgroupns=host \
  --pid=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  k0sproject/k0s:v1.22.17-k0s.0 \
  k0s controller --single

# Test 7: K0s + Pre-1.20 Version
docker rm -f k0s-test6
docker run --privileged -d \
  --name k0s-test7 \
  -p 6443:6443 \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  k0sproject/k0s:v1.19.15-k0s.0 \
  k0s controller --single
```

### Phase 2: Alternative Distributions
```bash
# Test 8: MicroK8s (Correct Image)
docker rm -f k0s-test7
# Note: MicroK8s doesn't have a direct Docker image, skip this

# Test 9: Kind + Host Namespace
docker run --privileged -d \
  --name kind-test9 \
  --network=host \
  --cgroupns=host \
  --pid=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  kindest/node:v1.19.16

# Test 10: Kind + Very Old Version
docker rm -f kind-test9
docker run --privileged -d \
  --name kind-test10 \
  -p 6443:6443 \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  kindest/node:v1.18.20

# Test 11: Minikube Alternative
docker rm -f kind-test10
docker run --privileged -d \
  --name minikube-test11 \
  -p 8443:8443 \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  kicbase/stable:v0.0.30

# Test 12: Custom Minimal Kubernetes
docker rm -f minikube-test11
docker run --privileged -d \
  --name minimal-test12 \
  -p 6443:6443 \
  -p 2379:2379 \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  k8s.gcr.io/etcd:3.4.13-0 \
  etcd --data-dir=/etcd-data --listen-client-urls=http://0.0.0.0:2379 --advertise-client-urls=http://127.0.0.1:2379
```

### Phase 3: Extreme Compatibility Testing
```bash
# Test 13: Kubernetes 1.16 (Very Old)
docker rm -f minimal-test12
docker run --privileged -d \
  --name k3s-ancient13 \
  -p 6443:6443 \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  rancher/k3s:v1.16.15-k3s1 \
  server --disable=traefik

# Test 14: K3s + CgroupFS Driver Override
docker rm -f k3s-ancient13
docker run --privileged -d \
  --name k3s-cgroupfs14 \
  -p 6443:6443 \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  rancher/k3s:v1.20.15-k3s1 \
  server \
    --snapshotter=native \
    --disable=traefik,servicelb,metrics-server \
    --kubelet-arg="--cgroup-driver=cgroupfs" \
    --kubelet-arg="--runtime-cgroups=/systemd/system.slice" \
    --kubelet-arg="--kubelet-cgroups=/systemd/system.slice"

# Test 15: K3s + All Controllers Disabled
docker rm -f k3s-cgroupfs14
docker run --privileged -d \
  --name k3s-nocontrollers15 \
  -p 6443:6443 \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  rancher/k3s:v1.21.14-k3s1 \
  server \
    --snapshotter=native \
    --disable=traefik,servicelb,metrics-server,local-storage \
    --kubelet-arg="--cgroup-driver=cgroupfs" \
    --kubelet-arg="--cgroups-per-qos=false" \
    --kubelet-arg="--enforce-node-allocatable=" \
    --kubelet-arg="--kube-reserved=" \
    --kubelet-arg="--system-reserved=" \
    --kubelet-arg="--eviction-hard=" \
    --kube-controller-manager-arg="--controllers=*,-nodelifecycle,-route,-service"
```

## Systematic Testing Approach

### For Each Test:
1. **Wait Time:** Give each test 5-10 minutes to initialize
2. **Log Analysis:** 
   ```bash
   docker logs CONTAINER_NAME --tail 50
   docker logs CONTAINER_NAME --follow  # Watch real-time
   ```
3. **Success Indicators:**
   - "Node ready" or "Server ready"
   - API server listening on port
   - No fatal cgroup errors
4. **Failure Indicators:**
   - "pids cgroup controller not found"
   - Container exits immediately
   - "function not implemented" errors

### Testing Commands for Success:
```bash
# If any test succeeds, verify with:
docker exec CONTAINER_NAME kubectl get nodes
# Or external access:
curl -k https://localhost:6443/version
```

## Alternative Solutions if All Tests Fail

### 1. Virtual Machine Approach (Recommended)
```bash
# Use Synology Virtual Machine Manager:
# 1. Install VM Manager from Package Center
# 2. Create Ubuntu 22.04 VM (4GB RAM, 20GB disk)
# 3. Install K3s natively:
curl -sfL https://get.k3s.io | sh -
```

### 2. Docker Swarm Alternative
```bash
# If Kubernetes fails entirely, use Docker Swarm:
docker swarm init
docker service create --name web --replicas 3 -p 80:80 nginx
```

### 3. Podman Alternative
```bash
# Install Podman as Docker alternative (if available)
# Podman has different cgroup handling
```

## Expected Success Rate
Based on the system limitations discovered:
- **Modern K8s (1.24+):** 0% success rate (pids cgroup requirement)
- **Intermediate K8s (1.20-1.23):** 10-20% success rate
- **Old K8s (1.16-1.19):** 30-50% success rate
- **VM Approach:** 95% success rate

## Recommendation
If all containerized approaches fail, proceed with the **Virtual Machine approach** using Synology's built-in VM Manager, as this bypasses all the DSM container limitations and provides full kernel feature access.
