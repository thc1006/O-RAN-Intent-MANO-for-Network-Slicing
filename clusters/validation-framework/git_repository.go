// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// GitRepository provides Git operations for validation
type GitRepository struct {
	Config    GitConfig
	LocalPath string
	Auth      GitAuth
}

// GitAuth holds authentication information
type GitAuth struct {
	Type     string // token, ssh, none
	Token    string
	SSHKey   string
	Username string
	Password string
}

// GitCommit represents a Git commit
type GitCommit struct {
	Hash    string    `json:"hash"`
	Author  string    `json:"author"`
	Email   string    `json:"email"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
	Files   []string  `json:"files"`
}

// GitStatus represents Git repository status
type GitStatus struct {
	Branch     string            `json:"branch"`
	Ahead      int               `json:"ahead"`
	Behind     int               `json:"behind"`
	Modified   []string          `json:"modified"`
	Added      []string          `json:"added"`
	Deleted    []string          `json:"deleted"`
	Untracked  []string          `json:"untracked"`
	Staged     []string          `json:"staged"`
	Clean      bool              `json:"clean"`
	LastCommit GitCommit         `json:"lastCommit"`
	RemoteURL  string            `json:"remoteUrl"`
	Tags       []string          `json:"tags"`
	Remotes    map[string]string `json:"remotes"`
}

// GitSyncStatus represents synchronization status
type GitSyncStatus struct {
	Status        string    `json:"status"` // synced, ahead, behind, diverged
	LastSync      time.Time `json:"lastSync"`
	LocalCommit   string    `json:"localCommit"`
	RemoteCommit  string    `json:"remoteCommit"`
	ConflictFiles []string  `json:"conflictFiles,omitempty"`
}

// NewGitRepository creates a new Git repository instance
func NewGitRepository(config GitConfig) (*GitRepository, error) {
	repo := &GitRepository{
		Config: config,
	}

	// Determine local path
	if config.Path != "" {
		repo.LocalPath = config.Path
	} else {
		// Use current working directory
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		repo.LocalPath = wd
	}

	// Setup authentication
	if err := repo.setupAuth(); err != nil {
		return nil, fmt.Errorf("failed to setup authentication: %w", err)
	}

	// Validate repository
	if err := repo.validateRepository(); err != nil {
		return nil, fmt.Errorf("invalid Git repository: %w", err)
	}

	return repo, nil
}

// setupAuth configures Git authentication
func (gr *GitRepository) setupAuth() error {
	if gr.Config.AuthToken != "" {
		gr.Auth.Type = "token"
		gr.Auth.Token = gr.Config.AuthToken

		// Configure Git to use token
		if err := gr.configureTokenAuth(); err != nil {
			return err
		}
	} else if gr.Config.SSHKeyPath != "" {
		gr.Auth.Type = "ssh"
		gr.Auth.SSHKey = gr.Config.SSHKeyPath

		// Configure Git to use SSH key
		if err := gr.configureSSHAuth(); err != nil {
			return err
		}
	} else {
		gr.Auth.Type = "none"
	}

	return nil
}

// configureTokenAuth configures Git to use token authentication
func (gr *GitRepository) configureTokenAuth() error {
	// Set up credential helper for token authentication using secure execution
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"config", "--local", "credential.helper", "store"}
	_, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return fmt.Errorf("failed to configure git credential helper: %w", err)
	}

	return nil
}

// configureSSHAuth configures Git to use SSH key authentication
func (gr *GitRepository) configureSSHAuth() error {
	// Validate SSH key path for security
	if err := security.ValidateFilePath(gr.Auth.SSHKey); err != nil {
		return fmt.Errorf("invalid SSH key path: %w", err)
	}

	// Verify SSH key exists
	if err := security.ValidateFileExists(gr.Auth.SSHKey); err != nil {
		return fmt.Errorf("SSH key validation failed: %w", err)
	}

	// Set GIT_SSH_COMMAND environment variable with validated path
	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", gr.Auth.SSHKey)
	if err := security.ValidateEnvironmentValue(sshCmd); err != nil {
		return fmt.Errorf("invalid SSH command: %w", err)
	}
	os.Setenv("GIT_SSH_COMMAND", sshCmd)

	return nil
}

// validateRepository validates that the directory is a Git repository
func (gr *GitRepository) validateRepository() error {
	// Validate local path for security
	if err := security.ValidateDirectoryExists(gr.LocalPath); err != nil {
		return fmt.Errorf("invalid repository path: %w", err)
	}

	gitDir := filepath.Join(gr.LocalPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", gr.LocalPath)
	}

	// Check if we can run git commands using secure execution
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"rev-parse", "--git-dir"}
	_, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return fmt.Errorf("git command failed: %w", err)
	}

	return nil
}

// GetCurrentBranch returns the current Git branch
func (gr *GitRepository) GetCurrentBranch() (branch string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"rev-parse", "--abbrev-ref", "HEAD"}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetLastCommit returns the last commit information
func (gr *GitRepository) GetLastCommit() (commit string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"rev-parse", "HEAD"}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return "", fmt.Errorf("failed to get last commit: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetLastCommitInfo returns detailed information about the last commit
func (gr *GitRepository) GetLastCommitInfo() (*GitCommit, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"log", "-1", "--pretty=format:%H|%an|%ae|%ct|%s"}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit info: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), "|")
	if len(parts) != 5 {
		return nil, fmt.Errorf("unexpected git log output format")
	}

	timestamp, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse commit timestamp: %w", err)
	}

	// Get modified files for this commit using secure execution
	filesCtx, filesCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer filesCancel()

	filesArgs := []string{"diff-tree", "--no-commit-id", "--name-only", "-r", parts[0]}
	filesOutput, _ := security.SecureExecuteWithValidation(filesCtx, "git", security.ValidateGitArgs, filesArgs...)

	var files []string
	if len(filesOutput) > 0 {
		trimmed := strings.TrimSpace(string(filesOutput))
		if trimmed != "" {
			files = strings.Split(trimmed, "\n")
		}
	}

	return &GitCommit{
		Hash:    parts[0],
		Author:  parts[1],
		Email:   parts[2],
		Date:    time.Unix(timestamp, 0),
		Message: parts[4],
		Files:   files,
	}, nil
}

// IsClean returns true if the repository has no uncommitted changes
func (gr *GitRepository) IsClean() (isClean bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"status", "--porcelain"}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	trimmed := strings.TrimSpace(string(output))
	return trimmed == "", nil
}

// GetStatus returns detailed Git repository status
func (gr *GitRepository) GetStatus() (*GitStatus, error) {
	status := &GitStatus{
		Remotes: make(map[string]string),
	}

	// Get current branch
	branch, err := gr.GetCurrentBranch()
	if err != nil {
		return nil, err
	}
	status.Branch = branch

	// Get ahead/behind count
	ahead, behind, err := gr.getAheadBehind()
	if err == nil {
		status.Ahead = ahead
		status.Behind = behind
	}

	// Get porcelain status using secure execution
	statusCtx, statusCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer statusCancel()

	statusArgs := []string{"status", "--porcelain"}
	output, err := security.SecureExecuteWithValidation(statusCtx, "git", security.ValidateGitArgs, statusArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	// Parse porcelain output
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 3 {
			continue
		}

		statusCode := line[:2]
		fileName := line[3:]

		switch {
		case statusCode[0] == 'M' || statusCode[1] == 'M':
			status.Modified = append(status.Modified, fileName)
		case statusCode[0] == 'A':
			status.Added = append(status.Added, fileName)
		case statusCode[0] == 'D':
			status.Deleted = append(status.Deleted, fileName)
		case statusCode == "??":
			status.Untracked = append(status.Untracked, fileName)
		case statusCode[0] != ' ' && statusCode[0] != '?':
			status.Staged = append(status.Staged, fileName)
		}
	}

	status.Clean = len(output) == 0

	// Get last commit info
	lastCommit, err := gr.GetLastCommitInfo()
	if err == nil {
		status.LastCommit = *lastCommit
	}

	// Get remote URL
	remoteURL, err := gr.getRemoteURL()
	if err == nil {
		status.RemoteURL = remoteURL
	}

	// Get tags
	tags, err := gr.getTags()
	if err == nil {
		status.Tags = tags
	}

	// Get remotes
	remotes, err := gr.getRemotes()
	if err == nil {
		status.Remotes = remotes
	}

	return status, nil
}

// getAheadBehind gets the ahead/behind count compared to remote
func (gr *GitRepository) getAheadBehind() (ahead int, behind int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	args := []string{"rev-list", "--left-right", "--count", "HEAD...@{upstream}"}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(string(output))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output")
	}

	ahead, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	behind, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return ahead, behind, nil
}

// getRemoteURL gets the remote URL
func (gr *GitRepository) getRemoteURL() (url string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"remote", "get-url", "origin"}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// getTags gets repository tags
func (gr *GitRepository) getTags() (tags []string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"tag", "--sort=-version:refname"}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

// getRemotes gets all remotes
func (gr *GitRepository) getRemotes() (remotes map[string]string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"remote", "-v"}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return nil, err
	}

	remotes = make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.HasSuffix(line, "(fetch)") {
			remotes[parts[0]] = parts[1]
		}
	}

	return remotes, nil
}

// GetSyncStatus returns the synchronization status with remote
func (gr *GitRepository) GetSyncStatus() (status string, lastSync time.Time, err error) {
	// Fetch latest from remote
	if err := gr.fetchRemote(); err != nil {
		return "unknown", time.Time{}, err
	}

	// Get ahead/behind status
	ahead, behind, err := gr.getAheadBehind()
	if err != nil {
		return "unknown", time.Time{}, err
	}

	switch {
	case ahead == 0 && behind == 0:
		status = "synced"
	case ahead > 0 && behind == 0:
		status = "ahead"
	case ahead == 0 && behind > 0:
		status = "behind"
	case ahead > 0 && behind > 0:
		status = "diverged"
	}

	// Get last fetch time (approximate using .git/FETCH_HEAD)
	fetchHeadPath := filepath.Join(gr.LocalPath, ".git", "FETCH_HEAD")
	if info, err := os.Stat(fetchHeadPath); err == nil {
		lastSync = info.ModTime()
	}

	return status, lastSync, nil
}

// fetchRemote fetches latest changes from remote
func (gr *GitRepository) fetchRemote() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	args := []string{"fetch", "origin"}
	_, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	return nil
}

// Pull pulls latest changes from remote
func (gr *GitRepository) Pull(ctx context.Context) error {
	// Validate branch name for security
	if err := security.ValidateGitRef(gr.Config.Branch); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	args := []string{"pull", "origin", gr.Config.Branch}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return fmt.Errorf("git pull failed: %w, output: %s", err, string(output))
	}

	return nil
}

// Push pushes local changes to remote
func (gr *GitRepository) Push(ctx context.Context) error {
	// Validate branch name for security
	if err := security.ValidateGitRef(gr.Config.Branch); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	args := []string{"push", "origin", gr.Config.Branch}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return fmt.Errorf("git push failed: %w, output: %s", err, string(output))
	}

	return nil
}

// CreateBranch creates a new branch
func (gr *GitRepository) CreateBranch(branchName string) error {
	// Validate branch name for security
	if err := security.ValidateGitRef(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	args := []string{"checkout", "-b", branchName}
	_, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	return nil
}

// SwitchBranch switches to an existing branch
func (gr *GitRepository) SwitchBranch(branchName string) error {
	// Validate branch name for security
	if err := security.ValidateGitRef(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	args := []string{"checkout", branchName}
	_, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return fmt.Errorf("failed to switch to branch %s: %w", branchName, err)
	}

	return nil
}

// GetDiff returns the diff between two commits
func (gr *GitRepository) GetDiff(fromCommit, toCommit string) (diff string, err error) {
	// Validate commit references for security
	if err := security.ValidateGitRef(fromCommit); err != nil {
		return "", fmt.Errorf("invalid from commit: %w", err)
	}
	if err := security.ValidateGitRef(toCommit); err != nil {
		return "", fmt.Errorf("invalid to commit: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{"diff", fromCommit, toCommit}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// GetChangedFiles returns files changed between two commits
func (gr *GitRepository) GetChangedFiles(fromCommit, toCommit string) (files []string, err error) {
	// Validate commit references for security
	if err := security.ValidateGitRef(fromCommit); err != nil {
		return nil, fmt.Errorf("invalid from commit: %w", err)
	}
	if err := security.ValidateGitRef(toCommit); err != nil {
		return nil, fmt.Errorf("invalid to commit: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{"diff", "--name-only", fromCommit, toCommit}
	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

// Reset resets the repository to a specific commit
func (gr *GitRepository) Reset(commit string, hard bool) error {
	// Validate commit reference for security
	if err := security.ValidateGitRef(commit); err != nil {
		return fmt.Errorf("invalid commit: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{"reset"}
	if hard {
		args = append(args, "--hard")
	}
	args = append(args, commit)

	_, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return fmt.Errorf("failed to reset to commit %s: %w", commit, err)
	}

	return nil
}

// GetCommitHistory returns commit history
func (gr *GitRepository) GetCommitHistory(limit int) (commits []GitCommit, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{"log", "--pretty=format:%H|%an|%ae|%ct|%s"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}

	output, err := security.SecureExecuteWithValidation(ctx, "git", security.ValidateGitArgs, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "|")
		if len(parts) != 5 {
			continue
		}

		timestamp, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil {
			continue
		}

		commit := GitCommit{
			Hash:    parts[0],
			Author:  parts[1],
			Email:   parts[2],
			Date:    time.Unix(timestamp, 0),
			Message: parts[4],
		}

		commits = append(commits, commit)
	}

	return commits, nil
}
