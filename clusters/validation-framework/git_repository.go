// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Date      time.Time `json:"date"`
	Message   string    `json:"message"`
	Files     []string  `json:"files"`
}

// GitStatus represents Git repository status
type GitStatus struct {
	Branch         string            `json:"branch"`
	Ahead          int               `json:"ahead"`
	Behind         int               `json:"behind"`
	Modified       []string          `json:"modified"`
	Added          []string          `json:"added"`
	Deleted        []string          `json:"deleted"`
	Untracked      []string          `json:"untracked"`
	Staged         []string          `json:"staged"`
	Clean          bool              `json:"clean"`
	LastCommit     GitCommit         `json:"lastCommit"`
	RemoteURL      string            `json:"remoteUrl"`
	Tags           []string          `json:"tags"`
	Remotes        map[string]string `json:"remotes"`
}

// GitSyncStatus represents synchronization status
type GitSyncStatus struct {
	Status        string    `json:"status"`        // synced, ahead, behind, diverged
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
	// Set up credential helper for token authentication
	cmd := exec.Command("git", "config", "--local", "credential.helper", "store")
	cmd.Dir = gr.LocalPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure git credential helper: %w", err)
	}

	return nil
}

// configureSSHAuth configures Git to use SSH key authentication
func (gr *GitRepository) configureSSHAuth() error {
	// Verify SSH key exists
	if _, err := os.Stat(gr.Auth.SSHKey); os.IsNotExist(err) {
		return fmt.Errorf("SSH key not found: %s", gr.Auth.SSHKey)
	}

	// Set GIT_SSH_COMMAND environment variable
	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", gr.Auth.SSHKey)
	os.Setenv("GIT_SSH_COMMAND", sshCmd)

	return nil
}

// validateRepository validates that the directory is a Git repository
func (gr *GitRepository) validateRepository() error {
	gitDir := filepath.Join(gr.LocalPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", gr.LocalPath)
	}

	// Check if we can run git commands
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = gr.LocalPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git command failed: %w", err)
	}

	return nil
}

// GetCurrentBranch returns the current Git branch
func (gr *GitRepository) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetLastCommit returns the last commit information
func (gr *GitRepository) GetLastCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get last commit: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetLastCommitInfo returns detailed information about the last commit
func (gr *GitRepository) GetLastCommitInfo() (*GitCommit, error) {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%H|%an|%ae|%ct|%s")
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
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

	// Get modified files for this commit
	filesCmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", parts[0])
	filesCmd.Dir = gr.LocalPath
	filesOutput, _ := filesCmd.Output()

	var files []string
	if len(filesOutput) > 0 {
		files = strings.Split(strings.TrimSpace(string(filesOutput)), "\n")
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
func (gr *GitRepository) IsClean() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return len(strings.TrimSpace(string(output))) == 0, nil
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

	// Get porcelain status
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = gr.LocalPath
	output, err := cmd.Output()
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
func (gr *GitRepository) getAheadBehind() (int, int, error) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(string(output))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output")
	}

	ahead, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	behind, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return ahead, behind, nil
}

// getRemoteURL gets the remote URL
func (gr *GitRepository) getRemoteURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// getTags gets repository tags
func (gr *GitRepository) getTags() ([]string, error) {
	cmd := exec.Command("git", "tag", "--sort=-version:refname")
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

// getRemotes gets all remotes
func (gr *GitRepository) getRemotes() (map[string]string, error) {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	remotes := make(map[string]string)
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
func (gr *GitRepository) GetSyncStatus() (string, time.Time, error) {
	// Fetch latest from remote
	if err := gr.fetchRemote(); err != nil {
		return "unknown", time.Time{}, err
	}

	// Get ahead/behind status
	ahead, behind, err := gr.getAheadBehind()
	if err != nil {
		return "unknown", time.Time{}, err
	}

	var status string
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
	var lastSync time.Time
	if info, err := os.Stat(fetchHeadPath); err == nil {
		lastSync = info.ModTime()
	}

	return status, lastSync, nil
}

// fetchRemote fetches latest changes from remote
func (gr *GitRepository) fetchRemote() error {
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = gr.LocalPath

	// Suppress output unless there's an error
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	return nil
}

// Pull pulls latest changes from remote
func (gr *GitRepository) Pull(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "pull", "origin", gr.Config.Branch)
	cmd.Dir = gr.LocalPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %w, output: %s", err, string(output))
	}

	return nil
}

// Push pushes local changes to remote
func (gr *GitRepository) Push(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "push", "origin", gr.Config.Branch)
	cmd.Dir = gr.LocalPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %w, output: %s", err, string(output))
	}

	return nil
}

// CreateBranch creates a new branch
func (gr *GitRepository) CreateBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = gr.LocalPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	return nil
}

// SwitchBranch switches to an existing branch
func (gr *GitRepository) SwitchBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = gr.LocalPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch to branch %s: %w", branchName, err)
	}

	return nil
}

// GetDiff returns the diff between two commits
func (gr *GitRepository) GetDiff(fromCommit, toCommit string) (string, error) {
	cmd := exec.Command("git", "diff", fromCommit, toCommit)
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// GetChangedFiles returns files changed between two commits
func (gr *GitRepository) GetChangedFiles(fromCommit, toCommit string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", fromCommit, toCommit)
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
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
	args := []string{"reset"}
	if hard {
		args = append(args, "--hard")
	}
	args = append(args, commit)

	cmd := exec.Command("git", args...)
	cmd.Dir = gr.LocalPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reset to commit %s: %w", commit, err)
	}

	return nil
}

// GetCommitHistory returns commit history
func (gr *GitRepository) GetCommitHistory(limit int) ([]GitCommit, error) {
	args := []string{"log", "--pretty=format:%H|%an|%ae|%ct|%s"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = gr.LocalPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	var commits []GitCommit
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