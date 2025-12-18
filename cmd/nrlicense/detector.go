// Copyright New Relic, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"io/fs"
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
	// StatusNewApache means the file was created after the fork, and is licensed under Apache 2.0.
	StatusNewApache
	// StatusNewProprietary means the file was created after the fork, and is licensed under the NR software license.
	StatusNewProprietary
	// StatusUnknown means we couldn't determine the status
	StatusUnknown
)

func (s FileStatus) String() string {
	switch s {
	case StatusUnmodified:
		return "unmodified"
	case StatusModified:
		return "modified"
	case StatusNewApache:
		return "newApache"
	case StatusNewProprietary:
		return "newProprietary"
	default:
		return "unknown"
	}
}

// GitDetector detects file modification status relative to a fork point
type GitDetector struct {
	forkCommit string
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
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("fork commit %s not found: %w", forkCommit, err)
	}

	return &GitDetector{
		forkCommit: forkCommit,
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
		return d.GetNewFileStatusFromLicense(filePath)
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

// getNewFileStatusFromLicense searches for a LICENSE file in all parent directories
func (d *GitDetector) GetNewFileStatusFromLicense(filePath string) (FileStatus, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return StatusUnknown, fmt.Errorf("resolving absolute path: %w", err)
	}
	dir := filepath.Dir(absPath)

	// Search through file's parent directories for LICENSE files
	for dir != d.repoRoot {
		res, err := filepath.Glob(fmt.Sprintf("%s/LICENSE_*", dir))
		if err != nil {
			return StatusUnknown, fmt.Errorf("searching for license: %w", err)
		}
		if len(res) > 1 {
			return StatusUnknown, fmt.Errorf("more than one LICENSE file found")
		} else if len(res) == 1 {
			license := filepath.Base(res[0])
			if strings.Contains(license, "_NEWRELIC_") {
				return StatusNewProprietary, nil
			} else if strings.Contains(license, "_APACHE_") {
				return StatusNewApache, nil
			} else {
				return StatusUnknown, fmt.Errorf("improper LICENSE filename: %s (expected LICENSE_NEWRELIC_[component] or LICENSE_APACHE_[component])", license)
			}
		}
		dir = filepath.Dir(dir)
	}

	// If no LICENSE is found, file is assumed Apache
	return StatusNewApache, nil
}

// fileExistsAtCommit checks if a file exists at a given commit
func (d *GitDetector) fileExistsAtCommit(filePath, commit string) (bool, error) {
	cmd := exec.Command("git", "cat-file", "-e", fmt.Sprintf("%s:%s", commit, filePath))
	err := cmd.Run()
	if err != nil {
		// File doesn't exist at this commit
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 128 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// fileModifiedSince checks if a file has been modified since a given commit
func (d *GitDetector) fileModifiedSince(filePath, commit string) (bool, error) {
	// Use git log to see if there are any commits affecting this file since the fork point
	cmd := exec.Command("git", "log", "--oneline", fmt.Sprintf("%s..HEAD", commit), "--", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("running git log: %w", err)
	}

	// If there's any output, the file has been modified
	return out.Len() > 0, nil
}

// fileDiffSince checks if a file has a diff with that file at a given commit
func (d *GitDetector) fileDiffSince(filePath string, commit string) (bool, error) {
	cmd := exec.Command("git", "diff", commit, "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("getting file diff since: %w", err)
	}
	return len(out) > 0, nil
}

// GetFileContentAtFork retrieves the file content at the fork point (for comparison)
func (d *GitDetector) GetFileContentAtFork(filePath string) ([]byte, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", d.forkCommit, filePath))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting file content at fork: %w", err)
	}
	return output, nil
}

// GetModificationDescription returns a description of what was modified in the file
func (d *GitDetector) GetModificationDescription(filePath string) string {
	commitHistoryURLSinceFork := fmt.Sprintf(
		"https://github.com/newrelic/nrdot-collector-components/commits/main/%s?since=2025-11-26",
		filepath.Clean(filePath),
	)
	return commitHistoryURLSinceFork
}

// GetProprietaryLicenseDirectories a description of directories covered under the NR proprietary license
func (d *GitDetector) GetTopLevelLicenseDescription() (string, error) {
	licensedDirs := []string{}
	err := filepath.WalkDir(d.repoRoot, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dirEntry.IsDir() {
			matches, err := filepath.Glob(fmt.Sprintf("%s/LICENSE_NEWRELIC_*", path))
			if err != nil {
				return err
			}
			if len(matches) > 0 {
				dir, err := filepath.Rel(d.repoRoot, filepath.Dir(matches[0]))
				if err != nil {
					return err
				}
				dir = fmt.Sprintf("New Relic Software License - %s", dir)
				licensedDirs = append(licensedDirs, dir)
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("getting proprietary license directories: %w", err)
	}
	return strings.Join(licensedDirs, "\n"), nil
}
