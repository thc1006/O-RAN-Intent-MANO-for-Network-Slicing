#!/bin/bash
set -e

echo "Setting up Kind cluster for TN testing..."

# Create Kind cluster
kind create cluster --config kind-config.yaml

# Wait for cluster to be ready
kubectl wait --for=condition=Ready nodes --all --timeout=120s

# Install TN CRDs
kubectl apply -f ../../manager/config/crd/bases/

# Deploy TN Manager
kubectl create namespace tn-system || true
kubectl apply -f ../../manager/config/rbac/
kubectl apply -f ../../manager/config/manager/

# Deploy TN Agent as DaemonSet
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: tn-agent
  namespace: tn-system
spec:
  selector:
    matchLabels:
      app: tn-agent
  template:
    metadata:
      labels:
        app: tn-agent
    spec:
      hostNetwork: true
      hostPID: true
      containers:
      - name: tn-agent
        image: tn-agent:latest
        imagePullPolicy: IfNotPresent
        securityContext:
          privileged: true
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: NAMESPACE
          value: tn-system
        volumeMounts:
        - name: host-proc
          mountPath: /host/proc
        - name: host-sys
          mountPath: /host/sys
      volumes:
      - name: host-proc
        hostPath:
          path: /proc
      - name: host-sys
        hostPath:
          path: /sys
      tolerations:
      - effect: NoSchedule
        operator: Exists
EOF

echo "Waiting for TN components to be ready..."
kubectl -n tn-system wait --for=condition=Ready pods --all --timeout=120s

echo "TN test environment setup complete!"
echo ""
echo "To run tests:"
echo "  go test ./tn/tests/iperf -v"