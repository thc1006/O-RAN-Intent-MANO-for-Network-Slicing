package pkg

import "os"

// IsRunningInCI checks if the code is running in a CI environment
func IsRunningInCI() bool {
	// Check common CI environment variables
	return os.Getenv("CI") == "true" ||
		os.Getenv("GITHUB_ACTIONS") == "true" ||
		os.Getenv("GITLAB_CI") == "true" ||
		os.Getenv("JENKINS_HOME") != "" ||
		os.Getenv("TRAVIS") == "true" ||
		os.Getenv("CIRCLECI") == "true"
}

// ShouldMockVXLAN determines if VXLAN operations should be mocked
func ShouldMockVXLAN() bool {
	// Mock VXLAN in CI or when explicitly requested
	return IsRunningInCI() || os.Getenv("MOCK_VXLAN") == "true"
}

// ShouldSkipKubernetesTests determines if Kubernetes-dependent tests should be skipped
func ShouldSkipKubernetesTests() bool {
	// Skip if in CI and no kubeconfig is present
	if !IsRunningInCI() {
		return false
	}

	// Check if kubeconfig exists
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}

	_, err := os.Stat(kubeconfig)
	return err != nil
}