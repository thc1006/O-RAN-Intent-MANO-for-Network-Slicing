#!/bin/bash

# Safe cleanup script for test processes
# This will NOT kill SSH connections or critical system processes

echo "🧹 Cleaning up test processes..."

# Kill any hanging WebSocket test servers
echo "Checking for WebSocket test servers..."
pgrep -f "websocket-server" | while read pid; do
    # Check if it's a test process (not production)
    if ps -p $pid -o args= | grep -q "test"; then
        echo "  Killing test WebSocket server (PID: $pid)"
        kill -TERM $pid 2>/dev/null
    fi
done

# Kill any hanging tmux sessions created by tests
echo "Checking for test tmux sessions..."
tmux list-sessions 2>/dev/null | grep -E "test-|claude-test|e2e-" | cut -d: -f1 | while read session; do
    echo "  Killing tmux session: $session"
    tmux kill-session -t "$session" 2>/dev/null
done

# Clean up temporary test files
echo "Cleaning up temporary test files..."
rm -rf /tmp/nephio-packages/* 2>/dev/null
rm -rf /tmp/argocd-apps/* 2>/dev/null
rm -rf /tmp/test-repo 2>/dev/null
rm -rf /tmp/bench-repo 2>/dev/null
rm -rf /tmp/claude/* 2>/dev/null

# Clean up Go test cache (optional)
echo "Cleaning Go test cache..."
go clean -testcache

# Show remaining processes (for information only)
echo ""
echo "📊 Active processes summary:"
echo "  tmux sessions: $(tmux list-sessions 2>/dev/null | wc -l)"
echo "  Go processes: $(pgrep -c go)"

# DO NOT kill these critical processes:
# - sshd (SSH daemon)
# - systemd processes
# - kernel processes
# - your current shell
# - docker/containerd
# - kubelet

echo ""
echo "✅ Cleanup complete!"
echo "⚠️  Note: SSH connections and critical system processes were NOT affected."