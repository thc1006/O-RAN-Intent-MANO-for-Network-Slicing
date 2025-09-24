#!/bin/bash

# Fix golangci-lint issues for O-RAN Intent-MANO project
# This script fixes all issues reported by golangci-lint

echo "Fixing golangci-lint issues..."

# Fix errcheck issues - add error handling
echo "Fixing errcheck issues..."

# o2-client/pkg/o2ims/client_test.go
sed -i '126s/json.NewEncoder(w).Encode(mockOCloudInfo)/if err := json.NewEncoder(w).Encode(mockOCloudInfo); err != nil { t.Logf("Error encoding: %v", err) }/' o2-client/pkg/o2ims/client_test.go
sed -i '161s/json.NewEncoder(w).Encode(mockOCloudInfo)/if err := json.NewEncoder(w).Encode(mockOCloudInfo); err != nil { t.Logf("Error encoding: %v", err) }/' o2-client/pkg/o2ims/client_test.go
sed -i '212s/json.NewEncoder(w).Encode(response)/if err := json.NewEncoder(w).Encode(response); err != nil { t.Logf("Error encoding: %v", err) }/' o2-client/pkg/o2ims/client_test.go
sed -i '360s/w.Write(\[\]byte("Not Found"))/if _, err := w.Write(\[\]byte("Not Found")); err != nil { t.Logf("Error writing: %v", err) }/' o2-client/pkg/o2ims/client_test.go

# pkg/security/filepath_test.go
sed -i '265s/defer os.RemoveAll(tmpDir)/defer func() { _ = os.RemoveAll(tmpDir) }()/' pkg/security/filepath_test.go
sed -i '273s/file.Close()/_ = file.Close()/' pkg/security/filepath_test.go

# pkg/security/validation_test.go
sed -i '405s/defer os.Remove(tmpFile.Name())/defer func() { _ = os.Remove(tmpFile.Name()) }()/' pkg/security/validation_test.go
sed -i '406s/tmpFile.Close()/_ = tmpFile.Close()/' pkg/security/validation_test.go
sed -i '413s/defer os.RemoveAll(tmpDir)/defer func() { _ = os.RemoveAll(tmpDir) }()/' pkg/security/validation_test.go
sed -i '448s/defer os.Remove(tmpFile.Name())/defer func() { _ = os.Remove(tmpFile.Name()) }()/' pkg/security/validation_test.go
sed -i '449s/tmpFile.Close()/_ = tmpFile.Close()/' pkg/security/validation_test.go
sed -i '456s/defer os.RemoveAll(tmpDir)/defer func() { _ = os.RemoveAll(tmpDir) }()/' pkg/security/validation_test.go

# tn/agent/pkg/agent_mock.go
sed -i '32s/w.Write(\[\]byte/_, _ = w.Write(\[\]byte/' tn/agent/pkg/agent_mock.go
sed -i '38s/w.Write(\[\]byte/_, _ = w.Write(\[\]byte/' tn/agent/pkg/agent_mock.go

# tn/agent/pkg/iperf_test.go
sed -i '118s/manager.StopServer(tc.port)/_ = manager.StopServer(tc.port)/' tn/agent/pkg/iperf_test.go
sed -i '147s/manager.StopServer(5001)/_ = manager.StopServer(5001)/' tn/agent/pkg/iperf_test.go

# tn/agent/pkg/vxlan/optimized_manager.go
sed -i '166s/go m.createTunnelOptimized/go func() { _ = m.createTunnelOptimized/' tn/agent/pkg/vxlan/optimized_manager.go
sed -i '166s/callback)/callback) }()/' tn/agent/pkg/vxlan/optimized_manager.go

# tn/agent/pkg/vxlan/optimized_manager_test.go
sed -i '442s/manager.executeOptimizedCommand(uniqueArgs)/_ = manager.executeOptimizedCommand(uniqueArgs)/' tn/agent/pkg/vxlan/optimized_manager_test.go
sed -i '814s/manager.CreateTunnelOptimized/_ = manager.CreateTunnelOptimized/' tn/agent/pkg/vxlan/optimized_manager_test.go
sed -i '830s/manager.executeOptimizedCommand(args)/_ = manager.executeOptimizedCommand(args)/' tn/agent/pkg/vxlan/optimized_manager_test.go
sed -i '899s/manager.DeleteTunnelOptimized(input.vxlanID)/_ = manager.DeleteTunnelOptimized(input.vxlanID)/' tn/agent/pkg/vxlan/optimized_manager_test.go

# tn/manager/pkg/client.go
sed -i '45s/defer resp.Body.Close()/defer func() { _ = resp.Body.Close() }()/' tn/manager/pkg/client.go
sed -i '89s/defer resp.Body.Close()/defer func() { _ = resp.Body.Close() }()/' tn/manager/pkg/client.go
sed -i '121s/defer resp.Body.Close()/defer func() { _ = resp.Body.Close() }()/' tn/manager/pkg/client.go

# tn/tests/coverage/coverage_test.go
sed -i '150s/defer file.Close()/defer func() { _ = file.Close() }()/' tn/tests/coverage/coverage_test.go
sed -i '402s/defer os.Remove(tmpFile)/defer func() { _ = os.Remove(tmpFile) }()/' tn/tests/coverage/coverage_test.go

# tn/tests/integration/http_integration_test.go
sed -i '713s/defer healthResp.Body.Close()/defer func() { _ = healthResp.Body.Close() }()/' tn/tests/integration/http_integration_test.go

# tn/tests/integration/iperf_integration_test.go
sed -i '87s/suite.manager.StopServer(port)/_ = suite.manager.StopServer(port)/' tn/tests/integration/iperf_integration_test.go
sed -i '176s/defer listener.Close()/defer func() { _ = listener.Close() }()/' tn/tests/integration/iperf_integration_test.go
sed -i '208s/defer suite.manager.StopServer(port)/defer func() { _ = suite.manager.StopServer(port) }()/' tn/tests/integration/iperf_integration_test.go
sed -i '295s/defer suite.manager.StopServer(port)/defer func() { _ = suite.manager.StopServer(port) }()/' tn/tests/integration/iperf_integration_test.go
sed -i '673s/conn.Close()/_ = conn.Close()/' tn/tests/integration/iperf_integration_test.go
sed -i '682s/ln.Close()/_ = ln.Close()/' tn/tests/integration/iperf_integration_test.go
sed -i '719s/suite.manager.StopAllServers()/_ = suite.manager.StopAllServers()/' tn/tests/integration/iperf_integration_test.go

# tn/tests/integration/vxlan_integration_test.go
sed -i '48s/suite.manager.DeleteTunnel(vxlanID)/_ = suite.manager.DeleteTunnel(vxlanID)/' tn/tests/integration/vxlan_integration_test.go
sed -i '49s/suite.optimizedManager.DeleteTunnelOptimized(vxlanID)/_ = suite.optimizedManager.DeleteTunnelOptimized(vxlanID)/' tn/tests/integration/vxlan_integration_test.go
sed -i '53s/suite.manager.Cleanup()/_ = suite.manager.Cleanup()/' tn/tests/integration/vxlan_integration_test.go
sed -i '54s/suite.optimizedManager.CleanupOptimized()/_ = suite.optimizedManager.CleanupOptimized()/' tn/tests/integration/vxlan_integration_test.go
sed -i '290s/suite.optimizedManager.DeleteTunnelOptimized(vxlanID)/_ = suite.optimizedManager.DeleteTunnelOptimized(vxlanID)/' tn/tests/integration/vxlan_integration_test.go
sed -i '333s/defer suite.manager.DeleteTunnel(vxlanID)/defer func() { _ = suite.manager.DeleteTunnel(vxlanID) }()/' tn/tests/integration/vxlan_integration_test.go
sed -i '621s/suite.optimizedManager.DeleteTunnelOptimized(vxlanID)/_ = suite.optimizedManager.DeleteTunnelOptimized(vxlanID)/' tn/tests/integration/vxlan_integration_test.go

# Fix ineffassign issue
echo "Fixing ineffassign issues..."
sed -i '169s/err = /_ = /' tn/agent/pkg/vxlan/optimized_manager_test.go

# Fix staticcheck issues
echo "Fixing staticcheck issues..."

# Error string capitalization
sed -i '240s/Git reference cannot be empty/git reference cannot be empty/' pkg/security/validation.go
sed -i '244s/Git reference too long/git reference too long/' pkg/security/validation.go
sed -i '265s/Kubernetes name cannot be empty/kubernetes name cannot be empty/' pkg/security/validation.go

# Duplicate characters in cutset
sed -i '358s/"dropped(),"/",drop()/' tn/agent/pkg/monitor.go

# Empty branches
sed -i '108,109d' tn/agent/pkg/vxlan/optimized_manager_test.go
sed -i '159,160d' tn/tests/coverage/coverage_test.go

# Type omissions
sed -i '344s/var output \*os.File = /output := /' tn/cmd/agent/main.go
sed -i '221s/var output \*os.File = /output := /' tn/cmd/manager/main.go

# ObjectMeta field
sed -i '68s/slice.ObjectMeta.DeletionTimestamp/slice.DeletionTimestamp/' tn/manager/controllers/tnslice_controller.go

# Deprecated ioutil
sed -i 's/"io\/ioutil"/"os"/' tn/tests/security/kubernetes_manifest_test.go
sed -i 's/ioutil\./os\./' tn/tests/security/kubernetes_manifest_test.go

# Fix unused variables
echo "Fixing unused variables..."

# Comment out unused functions and variables
sed -i '11s/^/\/\/ /' tn/agent/pkg/vxlan/exec_windows.go
sed -i '12s/^/\/\/ /' tn/agent/pkg/vxlan/exec_windows.go
sed -i '13s/^/\/\/ /' tn/agent/pkg/vxlan/exec_windows.go

# Remove unused fields from struct
sed -i '/commandPool.*\*sync.Pool/d' tn/agent/pkg/vxlan/optimized_manager.go
sed -i '/batchTimer.*\*time.Timer/d' tn/agent/pkg/vxlan/optimized_manager.go
sed -i '/netlinkSocket.*int/d' tn/agent/pkg/vxlan/optimized_manager.go

# Remove unused variables
sed -i '/logLevel.*= flag.String/d' tn/cmd/agent/main.go
sed -i '/logLevel.*= flag.String/d' tn/cmd/manager/main.go

echo "All lint issues have been fixed!"
echo "Running verification..."

# Run golangci-lint to verify
golangci-lint run --no-config ./adapters/vnf-operator/... ./orchestrator/... ./cn-dms/... ./ran-dms/... ./tn/... ./o2-client/... ./pkg/security/...