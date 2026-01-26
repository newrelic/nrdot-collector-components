// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileStatus represents the modification status of a file
type FileStatus int

const (
	// StatusUnmodified means the file exists in the fork but hasn't been modified
	StatusUnmodified FileStatus = iota
	// StatusModified means the file existed in the fork and has been modified
	StatusModified
	// StatusNew means the file was created after the fork, and is licensed under Apache 2.0.
	StatusNew
	// StatusUnknown means we couldn't determine the status
	StatusUnknown
)

func (s FileStatus) String() string {
	switch s {
	case StatusUnmodified:
		return "unmodified"
	case StatusModified:
		return "modified"
	case StatusNew:
		return "newApache"
	default:
		return "unknown"
	}
}

// GitDetector detects file modification status relative to a fork point
type GitDetector struct {
	forkCommit string
	forkDate   string
	repoRoot   string
}

// validatePath ensures the file path is within the repository
func (d *GitDetector) validatePath(filePath string) error {
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolving absolute path: %w", err)
	}

	// Ensure it's within the repository
	if !strings.HasPrefix(absPath, d.repoRoot) {
		return fmt.Errorf("path outside repository: %s (repo root: %s)", absPath, d.repoRoot)
	}

	return nil
}

// getDateAfterForkCommit returns the date of the commit immediately after the fork commit in YYYY-MM-DD format.
// Input commit is the last commit befor the fork, so we need the date of the next commit to accurately represent our changes.
func getDateAfterForkCommit(commit string) (string, error) {
	cmd := exec.Command("git", "log", "--reverse", "--ancestry-path", fmt.Sprintf("%s..main", commit), "--format=%cs")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting next commit date: %w", err)
	}

	dates := strings.Split(string(output), "\n")
	date := strings.TrimSpace(dates[0])
	if date == "" {
		return "", fmt.Errorf("no commits found after fork point")
	}
	return date, nil
}

// NewGitDetector creates a new GitDetector
func NewGitDetector(forkCommit string) (*GitDetector, error) {
	// Verify we're in a git repository
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	repoRoot := strings.TrimSpace(string(output))

	// Verify the fork commit exists
	cmd = exec.Command("git", "rev-parse", "--verify", forkCommit)
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("fork commit %s not found: %w", forkCommit, err)
	}

	// Validate that commit hash is reachable if the repository is shallow
	cmd = exec.Command("git", "rev-parse", "--is-shallow-repository")
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("checking if shallow repository: %w", err)
	}
	if strings.TrimSpace(string(output)) == "true" {
		// We cannot fetch here because shallow repositories are locked during concurrently-running (-j2) makefile jobs. Throw error instead.
		cmd = exec.Command("git", "cat-file", "-e", forkCommit)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("fork commit %s is not reachable in shallow repository (shallow clone may need deeper history)", forkCommit)
		}
	}

	forkDate, err := getDateAfterForkCommit(forkCommit)
	if err != nil {
		return nil, err
	}

	return &GitDetector{
		forkCommit: forkCommit,
		forkDate:   forkDate,
		repoRoot:   repoRoot,
	}, nil
}

// GetFileStatus determines if a file has been modified since the fork point
func (d *GitDetector) GetFileStatus(filePath string) (FileStatus, error) {
	// Validate path is within repository
	if err := d.validatePath(filePath); err != nil {
		return StatusUnknown, err
	}
	// Check if file exists at fork point
	existsAtFork, err := d.fileExistsAtCommit(filePath, d.forkCommit)
	if err != nil {
		return StatusUnknown, fmt.Errorf("checking if file exists at fork: %w", err)
	}

	if !existsAtFork {
		return StatusNew, nil
	}

	// File exists at fork, check if it's there's a difference
	modified, err := d.fileDiffSince(filePath, d.forkCommit)
	if err != nil {
		return StatusUnknown, fmt.Errorf("checking if file modified: %w", err)
	}

	if modified {
		return StatusModified, nil
	}

	return StatusUnmodified, nil
}

// fileExistsAtCommit checks if a file exists at a given commit
func (*GitDetector) fileExistsAtCommit(filePath, commit string) (bool, error) {
	commitPath := fmt.Sprintf("%s:%s", commit, filePath)
	cmd := exec.Command("git", "cat-file", "-e", commitPath)
	err := cmd.Run()
	if err != nil {
		// File doesn't exist at this commit
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 128 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// fileDiffSince checks if a file has a diff with that file at a given commit
func (*GitDetector) fileDiffSince(filePath, commit string) (bool, error) {
	cmd := exec.Command("git", "diff", commit, "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("getting file diff since: %w", err)
	}
	return len(out) > 0, nil
}

// GetFileContentAtFork retrieves the file content at the fork point (for comparison)
func (d *GitDetector) GetFileContentAtFork(filePath string) ([]byte, error) {
	commitPath := fmt.Sprintf("%s:%s", d.forkCommit, filePath)
	cmd := exec.Command("git", "show", commitPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting file content at fork: %w", err)
	}
	return output, nil
}

// GetModificationDescription returns a description of what was modified in the file
func (d *GitDetector) GetModificationDescription(filePath string) string {
	commitHistoryURLSinceFork := fmt.Sprintf(
		"https://github.com/newrelic/nrdot-collector-components/commits/main/%s?since=%s",
		filepath.Clean(filePath),
		d.forkDate,
	)
	return commitHistoryURLSinceFork
}
