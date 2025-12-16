// Copyright New Relic, Inc.
// SPDX-License-Identifier: Apache-2.0

// nrlicense manages license headers for forked codebases.
// It applies different headers based on whether files have been modified since the fork point.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const helpText = `Usage: nrlicense [flags] pattern [pattern ...]

This tool manages license headers for forked codebases, applying different
headers based on file modification history:

- Unmodified files: Retain original copyright header
- Modified files: Add dual copyright with modification notice
- New files: New Relic copyright only

Examples:
  nrlicense --check $(find . -name "*.go")
  nrlicense --fix --fork-commit v0.140.0 .
  nrlicense --check --verbose file1.go file2.go

Flags:
`

var (
	check      = flag.Bool("check", false, "check mode: verify headers without modifying files")
	fix        = flag.Bool("fix", false, "fix mode: add or update license headers")
	forkCommit = flag.String("fork-commit", "51061db5838300734ff23888e2396263f61146d9", "git commit/tag representing the fork point")
	verbose    = flag.Bool("verbose", false, "verbose output: show processed files")
	dryRun     = flag.Bool("dry-run", false, "dry run: show what would be changed without modifying files")
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, helpText)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	// Count how many modes are specified
	modesSet := 0
	if *check {
		modesSet++
	}
	if *fix {
		modesSet++
	}
	if *dryRun {
		modesSet++
	}

	if modesSet == 0 {
		fmt.Fprintln(os.Stderr, "Error: must specify one of --check, --fix, or --dry-run")
		flag.Usage()
		os.Exit(1)
	}

	if modesSet > 1 {
		fmt.Fprintln(os.Stderr, "Error: can only specify one of --check, --fix, or --dry-run")
		flag.Usage()
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: must specify at least one file or pattern")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize git detector
	detector, err := NewGitDetector(*forkCommit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing git detector: %v\n", err)
		os.Exit(1)
	}

	// Collect all files to process
	files, err := collectFiles(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "No files to process")
		os.Exit(0)
	}

	// Process files
	processor := &Processor{
		detector: detector,
		verbose:  *verbose,
		dryRun:   *dryRun,
		check:    *check,
	}

	exitCode := processor.ProcessFiles(files)
	os.Exit(exitCode)
}

// collectFiles expands file patterns and returns a list of files to process
func collectFiles(patterns []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		// Check if it's a direct file
		info, err := os.Stat(pattern)
		if err == nil && !info.IsDir() {
			if !seen[pattern] && shouldProcessFile(pattern) {
				files = append(files, pattern)
				seen[pattern] = true
			}
			continue
		}

		// Check if it's a directory
		if err == nil && info.IsDir() {
			err := filepath.Walk(pattern, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && !seen[path] && shouldProcessFile(path) {
					files = append(files, path)
					seen[path] = true
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walking directory %s: %w", pattern, err)
			}
			continue
		}

		// Try glob expansion
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("expanding pattern %s: %w", pattern, err)
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if !info.IsDir() && !seen[match] && shouldProcessFile(match) {
				files = append(files, match)
				seen[match] = true
			}
		}
	}

	return files, nil
}

// shouldProcessFile determines if a file should be processed based on extension
func shouldProcessFile(path string) bool {
	// Skip vendor, node_modules, .git, etc.
	if strings.Contains(path, "/vendor/") ||
		strings.Contains(path, "/node_modules/") ||
		strings.Contains(path, "/.git/") ||
		strings.Contains(path, "/third_party/") {
		return false
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".go", ".sh", ".py", ".java", ".js", ".ts", ".proto":
		return true
	default:
		return false
	}
}
