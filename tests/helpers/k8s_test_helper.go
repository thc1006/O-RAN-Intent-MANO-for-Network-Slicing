// Package helpers provides test utility functions for Kubernetes testing environments.
package helpers

import (
	"os"
	"testing"
)

// Constants to avoid goconst linter issues
const (
	ciTrue        = "true"
	githubActions = "GITHUB_ACTIONS"
	ci            = "CI"
	travis        = "TRAVIS"
	circleci      = "CIRCLECI"
)

// SkipIfNoKubeconfig skips the test if kubeconfig is not available in CI environment
func SkipIfNoKubeconfig(t *testing.T) {
	t.Helper()

	// Check if running in CI
	if os.Getenv(ci) != ciTrue && os.Getenv(githubActions) != ciTrue {
		return // Not in CI, proceed with test
	}

	// Check for kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE") // Windows
		}
		if home != "" {
			kubeconfig = home + "/.kube/config"
		}
	}

	if kubeconfig == "" || !fileExists(kubeconfig) {
		t.Skip("Skipping test in CI: kubeconfig not available")
	}
}

// SkipInCI skips the test if running in CI environment
func SkipInCI(t *testing.T, reason string) {
	t.Helper()

	if os.Getenv(ci) == ciTrue || os.Getenv(githubActions) == ciTrue {
		if reason != "" {
			t.Skipf("Skipping test in CI: %s", reason)
		} else {
			t.Skip("Skipping test in CI")
		}
	}
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || !os.IsNotExist(err)
}

// IsRunningInCI returns true if the code is running in a CI environment
func IsRunningInCI() bool {
	return os.Getenv(ci) == ciTrue ||
		os.Getenv(githubActions) == ciTrue ||
		os.Getenv("GITLAB_CI") == ciTrue ||
		os.Getenv("JENKINS_HOME") != "" ||
		os.Getenv(travis) == ciTrue ||
		os.Getenv(circleci) == ciTrue
}
