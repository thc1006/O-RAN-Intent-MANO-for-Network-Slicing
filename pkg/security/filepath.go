package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AllowedDirectory represents a directory that is allowed for file operations
type AllowedDirectory struct {
	Path        string   // Base path that is allowed
	Extensions  []string // Allowed file extensions (e.g., [".yaml", ".yml", ".json"])
	Recursive   bool     // Whether subdirectories are allowed
	Description string   // Description of what this directory is for
}

// FilePathValidator provides secure file path validation
type FilePathValidator struct {
	allowedDirs     []AllowedDirectory
	maxPathLength   int
	maxFilenameSize int64
}

// NewFilePathValidator creates a new file path validator with default settings
func NewFilePathValidator() *FilePathValidator {
	return &FilePathValidator{
		allowedDirs:     make([]AllowedDirectory, 0),
		maxPathLength:   4096,           // Maximum path length
		maxFilenameSize: 100 * 1024 * 1024, // 100MB max file size
	}
}

// AddAllowedDirectory adds a directory to the allowlist
func (v *FilePathValidator) AddAllowedDirectory(dir AllowedDirectory) {
	// Clean the directory path
	dir.Path = filepath.Clean(dir.Path)
	v.allowedDirs = append(v.allowedDirs, dir)
}

// SetMaxPathLength sets the maximum allowed path length
func (v *FilePathValidator) SetMaxPathLength(length int) {
	v.maxPathLength = length
}

// SetMaxFileSize sets the maximum allowed file size
func (v *FilePathValidator) SetMaxFileSize(size int64) {
	v.maxFilenameSize = size
}

// ValidateFilePath validates a file path for security issues
func (v *FilePathValidator) ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Check path length
	if len(path) > v.maxPathLength {
		return fmt.Errorf("file path too long: %d characters (max: %d)", len(path), v.maxPathLength)
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal: %s", path)
	}

	// Check for absolute path traversal on Unix systems
	if strings.HasPrefix(cleanPath, "/") && !isAbsolutePathAllowed(cleanPath) {
		return fmt.Errorf("absolute path not in allowed directories: %s", cleanPath)
	}

	// Check for null bytes (path injection)
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null byte")
	}

	// Validate against allowed directories if any are configured
	if len(v.allowedDirs) > 0 {
		if err := v.validateAgainstAllowedDirs(cleanPath); err != nil {
			return err
		}
	}

	return nil
}

// ValidateFilePathAndExtension validates both path and file extension
func (v *FilePathValidator) ValidateFilePathAndExtension(path string, allowedExts []string) error {
	if err := v.ValidateFilePath(path); err != nil {
		return err
	}

	if len(allowedExts) > 0 {
		ext := strings.ToLower(filepath.Ext(path))
		allowed := false
		for _, allowedExt := range allowedExts {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file extension %s not allowed, allowed extensions: %v", ext, allowedExts)
		}
	}

	return nil
}

// SafeReadFile safely reads a file after validation
func (v *FilePathValidator) SafeReadFile(path string) ([]byte, error) {
	if err := v.ValidateFilePath(path); err != nil {
		return nil, fmt.Errorf("file path validation failed: %w", err)
	}

	// Get file info to check size
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	if info.Size() > v.maxFilenameSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", info.Size(), v.maxFilenameSize)
	}

	// Check if it's actually a file
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", path)
	}

	return os.ReadFile(path)
}

// SafeOpenFile safely opens a file after validation
func (v *FilePathValidator) SafeOpenFile(path string) (*os.File, error) {
	if err := v.ValidateFilePath(path); err != nil {
		return nil, fmt.Errorf("file path validation failed: %w", err)
	}

	return os.Open(path)
}

// validateAgainstAllowedDirs checks if the path is within allowed directories
func (v *FilePathValidator) validateAgainstAllowedDirs(cleanPath string) error {
	for _, allowedDir := range v.allowedDirs {
		if v.isPathAllowed(cleanPath, allowedDir) {
			return nil
		}
	}
	return fmt.Errorf("path not in any allowed directory: %s", cleanPath)
}

// isPathAllowed checks if a path is allowed based on an AllowedDirectory configuration
func (v *FilePathValidator) isPathAllowed(cleanPath string, allowedDir AllowedDirectory) bool {
	absAllowedPath, err := filepath.Abs(allowedDir.Path)
	if err != nil {
		return false
	}

	absCleanPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return false
	}

	// Check if the path is within the allowed directory
	rel, err := filepath.Rel(absAllowedPath, absCleanPath)
	if err != nil {
		return false
	}

	// Path must not go up from the allowed directory
	if strings.HasPrefix(rel, "..") {
		return false
	}

	// If recursive is false, path must be directly in the allowed directory
	if !allowedDir.Recursive && strings.Contains(rel, string(filepath.Separator)) {
		return false
	}

	// Check file extension if specified
	if len(allowedDir.Extensions) > 0 {
		ext := strings.ToLower(filepath.Ext(cleanPath))
		allowed := false
		for _, allowedExt := range allowedDir.Extensions {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return true
}

// isAbsolutePathAllowed checks if an absolute path is in a generally safe location
func isAbsolutePathAllowed(path string) bool {
	// This is a basic check - you might want to customize this based on your needs
	dangerousPaths := []string{
		"/etc/",
		"/proc/",
		"/sys/",
		"/dev/",
		"/root/",
		"/boot/",
	}

	for _, dangerous := range dangerousPaths {
		if strings.HasPrefix(path, dangerous) {
			return false
		}
	}
	return true
}

// CreateValidatorForKubernetes creates a validator configured for Kubernetes files
func CreateValidatorForKubernetes(workingDir string) *FilePathValidator {
	validator := NewFilePathValidator()

	// Add common Kubernetes directories
	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "clusters"),
		Extensions:  []string{".yaml", ".yml", ".json"},
		Recursive:   true,
		Description: "Kubernetes cluster configurations",
	})

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "config"),
		Extensions:  []string{".yaml", ".yml", ".json", ".toml"},
		Recursive:   true,
		Description: "Configuration files",
	})

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "manifests"),
		Extensions:  []string{".yaml", ".yml"},
		Recursive:   true,
		Description: "Kubernetes manifests",
	})

	return validator
}

// CreateValidatorForLogs creates a validator configured for log files
func CreateValidatorForLogs(workingDir string) *FilePathValidator {
	validator := NewFilePathValidator()

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "logs"),
		Extensions:  []string{".log", ".txt"},
		Recursive:   true,
		Description: "Log files",
	})

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        "/var/log",
		Extensions:  []string{".log", ".txt"},
		Recursive:   true,
		Description: "System log files",
	})

	return validator
}

// CreateValidatorForConfig creates a validator configured for configuration files
func CreateValidatorForConfig(workingDir string) *FilePathValidator {
	validator := NewFilePathValidator()

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        workingDir,
		Extensions:  []string{".yaml", ".yml", ".json", ".toml", ".conf", ".cfg"},
		Recursive:   false,
		Description: "Configuration files in working directory",
	})

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "config"),
		Extensions:  []string{".yaml", ".yml", ".json", ".toml", ".conf", ".cfg"},
		Recursive:   true,
		Description: "Configuration files in config directory",
	})

	return validator
}

// ValidateFilePath is a global function for backward compatibility
func ValidateFilePath(path string) error {
	validator := NewFilePathValidator()
	return validator.ValidateFilePath(path)
}

// CreateValidatorForLogs creates a validator configured for log files with enhanced security
func CreateValidatorForLogsEnhanced(workingDir string) *FilePathValidator {
	validator := NewFilePathValidator()

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "logs"),
		Extensions:  []string{".log", ".txt", ".out"},
		Recursive:   true,
		Description: "Application log files",
	})

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "reports"),
		Extensions:  []string{".log", ".txt", ".json", ".xml", ".html"},
		Recursive:   true,
		Description: "Test and validation reports",
	})

	// Be more restrictive with system logs
	if workingDir != "/" {
		validator.AddAllowedDirectory(AllowedDirectory{
			Path:        "/var/log/containers",
			Extensions:  []string{".log"},
			Recursive:   false,
			Description: "Container log files (read-only)",
		})
	}

	return validator
}

// CreateValidatorForTestData creates a validator for test data files
func CreateValidatorForTestData(workingDir string) *FilePathValidator {
	validator := NewFilePathValidator()

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "testdata"),
		Extensions:  []string{".yaml", ".yml", ".json", ".txt", ".csv", ".xml"},
		Recursive:   true,
		Description: "Test data files",
	})

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "tests"),
		Extensions:  []string{".yaml", ".yml", ".json", ".txt", ".go"},
		Recursive:   true,
		Description: "Test files and fixtures",
	})

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "golden"),
		Extensions:  []string{".yaml", ".yml", ".json", ".txt", ".golden"},
		Recursive:   true,
		Description: "Golden test files",
	})

	return validator
}

// CreateValidatorForSnapshots creates a validator for test snapshots
func CreateValidatorForSnapshots(workingDir string) *FilePathValidator {
	validator := NewFilePathValidator()

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "testdata", "snapshots"),
		Extensions:  []string{".json", ".yaml", ".yml"},
		Recursive:   false,
		Description: "Test snapshot files",
	})

	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        filepath.Join(workingDir, "snapshots"),
		Extensions:  []string{".json", ".yaml", ".yml"},
		Recursive:   false,
		Description: "Test snapshot files",
	})

	return validator
}

// SecureJoinPath safely joins path components and validates the result
func SecureJoinPath(base string, components ...string) (string, error) {
	// Validate base path
	if err := ValidateFilePath(base); err != nil {
		return "", fmt.Errorf("invalid base path: %w", err)
	}

	// Clean and validate each component
	cleanComponents := make([]string, len(components))
	for i, component := range components {
		// Remove any path separators and dangerous characters
		clean := filepath.Clean(component)
		if strings.Contains(clean, "..") || strings.Contains(clean, "/") || strings.Contains(clean, "\\") {
			return "", fmt.Errorf("invalid path component: %s", component)
		}
		cleanComponents[i] = clean
	}

	// Join with base
	result := base
	for _, component := range cleanComponents {
		result = filepath.Join(result, component)
	}

	// Clean and validate final result
	result = filepath.Clean(result)

	// Ensure result is still under base directory
	relPath, err := filepath.Rel(base, result)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path escapes base directory: %s", result)
	}

	return result, nil
}

// ValidateAndCleanPath validates and cleans a path for safe usage
func ValidateAndCleanPath(path string, allowedExtensions []string) (string, error) {
	// Basic validation
	if err := ValidateFilePath(path); err != nil {
		return "", err
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Additional validation after cleaning
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path contains directory traversal after cleaning: %s", path)
	}

	// Check file extension if specified
	if len(allowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(cleanPath))
		allowed := false
		for _, allowedExt := range allowedExtensions {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("file extension %s not allowed, allowed extensions: %v", ext, allowedExtensions)
		}
	}

	return cleanPath, nil
}

// SecureDirMode provides secure directory permissions (750)
const SecureDirMode = 0750

// PrivateDirMode provides private directory permissions (700)
const PrivateDirMode = 0700

// SecureFileMode provides secure file permissions (640)
const SecureFileMode = 0640

// SecureCreateFile creates a file with secure permissions
func SecureCreateFile(filename string) (*os.File, error) {
	// Validate file path for security
	if err := ValidateFilePath(filename); err != nil {
		return nil, fmt.Errorf("file path validation failed: %w", err)
	}

	// Create file with secure permissions
	return os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, SecureFileMode)
}

// SecureCreateDir creates a directory with secure permissions
func SecureCreateDir(dirname string) error {
	// Validate directory path for security
	if err := ValidateFilePath(dirname); err != nil {
		return fmt.Errorf("directory path validation failed: %w", err)
	}

	return os.MkdirAll(dirname, SecureDirMode)
}

// ValidateGitRef validates a Git reference (commit hash, branch name, tag)
func ValidateGitRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("git reference cannot be empty")
	}

	// Check for null bytes
	if strings.Contains(ref, "\x00") {
		return fmt.Errorf("git reference contains null byte")
	}

	// Check for command injection characters
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\"", "'", "\\"}
	for _, char := range dangerousChars {
		if strings.Contains(ref, char) {
			return fmt.Errorf("git reference contains dangerous character: %s", char)
		}
	}

	// Check length (Git refs should be reasonable length)
	if len(ref) > 256 {
		return fmt.Errorf("git reference too long: %d characters (max: 256)", len(ref))
	}

	// Basic validation for common Git ref patterns
	// Commit hashes are 7-40 hex characters
	// Branch/tag names should not start with certain characters
	if strings.HasPrefix(ref, "-") || strings.HasPrefix(ref, ".") {
		return fmt.Errorf("invalid git reference format: %s", ref)
	}

	return nil
}