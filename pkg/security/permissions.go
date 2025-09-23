// Package security provides secure file system permission constants and utilities
// following the principle of least privilege.
package security

import "os"

// File Permission Constants
// These constants follow the principle of least privilege for file system operations.
const (
	// SecureFileMode - Read/write for owner only (0600)
	// Used for: configuration files, logs, reports, sensitive data
	SecureFileMode os.FileMode = 0600

	// ReadOnlyFileMode - Read-only for owner only (0400)
	// Used for: read-only configuration files, certificates
	ReadOnlyFileMode os.FileMode = 0400

	// ExecutableFileMode - Read/execute for owner only (0500)
	// Used for: scripts that need to be executed
	ExecutableFileMode os.FileMode = 0500

	// SecureDirMode - Read/write/execute for owner, read/execute for group (0750)
	// Used for: directories that need group access
	SecureDirMode os.FileMode = 0750

	// PrivateDirMode - Read/write/execute for owner only (0700)
	// Used for: private directories, temporary directories
	PrivateDirMode os.FileMode = 0700

	// TempDirMode - Read/write/execute for owner only (0700)
	// Used for: temporary directories
	TempDirMode os.FileMode = 0700
)

// SecureCreateFile creates a file with secure permissions (0600)
func SecureCreateFile(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, SecureFileMode)
}

// SecureCreateDir creates a directory with secure permissions (0750)
func SecureCreateDir(path string) error {
	return os.MkdirAll(path, SecureDirMode)
}

// SecureCreatePrivateDir creates a directory with private permissions (0700)
func SecureCreatePrivateDir(path string) error {
	return os.MkdirAll(path, PrivateDirMode)
}

// SecureWriteFile writes data to a file with secure permissions (0600)
func SecureWriteFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, SecureFileMode)
}