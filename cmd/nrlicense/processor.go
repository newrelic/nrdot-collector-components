// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

// Processor handles the processing of files
type Processor struct {
	detector   *GitDetector
	verbose    bool
	dryRun     bool
	check      bool
	topLicense bool

	// Counters
	processed   int32
	modified    int32
	failed      int32
	needsUpdate int32
}

// ProcessFiles processes a list of files
func (p *Processor) ProcessFiles(files []string) int {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrency

	for _, file := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := p.processFile(filePath); err != nil {
				atomic.AddInt32(&p.failed, 1)
				fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", filePath, err)
			}
		}(file)
	}

	wg.Wait()

	// Print summary
	p.printSummary()

	// Return exit code
	if p.failed > 0 {
		return 1
	}
	if p.check && p.needsUpdate > 0 {
		return 1
	}

	return 0
}

// processFile processes a single file
func (p *Processor) processFile(filePath string) error {
	atomic.AddInt32(&p.processed, 1)

	// Check if file is generated
	headerInfo, err := ParseFileHeader(filePath)
	if err != nil {
		return fmt.Errorf("parsing header: %w", err)
	}

	if headerInfo.IsGenerated {
		if p.verbose {
			fmt.Printf("Skipping generated file: %s\n", filePath)
		}
		return nil
	}

	// Detect file status
	status, err := p.detector.GetFileStatus(filePath)
	if err != nil {
		return fmt.Errorf("detecting status: %w", err)
	}

	if p.verbose {
		fmt.Printf("Processing %s [%s]\n", filePath, status)
	}

	// For unmodified files, verify they have the original header
	if status == StatusUnmodified {
		if p.check {
			// In check mode, verify the header is correct
			var correct bool
			correct, err = CheckHeader(filePath, status)
			if err != nil {
				return fmt.Errorf("checking header: %w", err)
			}
			if !correct {
				atomic.AddInt32(&p.needsUpdate, 1)
				fmt.Printf("Missing or incorrect header (expected %s header): %s\n", status, filePath)
			}
		}
		// In fix mode, don't modify unmodified files
		return nil
	}

	// Generate the appropriate header
	modDescription := ""
	if status == StatusModified {
		modDescription = p.detector.GetModificationDescription(filePath)
	}

	newHeader, err := GenerateHeader(status, modDescription, filePath)
	if err != nil {
		return fmt.Errorf("generating header: %w", err)
	}

	// If no header generated (e.g., for unmodified files), skip
	if newHeader == "" {
		return nil
	}

	// Check mode: verify header is correct
	if p.check {
		correct, err := CheckHeader(filePath, status)
		if err != nil {
			return fmt.Errorf("checking header: %w", err)
		}
		if !correct {
			atomic.AddInt32(&p.needsUpdate, 1)
			fmt.Printf("Missing or incorrect header (expected %s header): %s\n", status, filePath)
		}
		return nil
	}

	// Dry run mode: show what would be changed
	if p.dryRun {
		fmt.Printf("Would update %s:\n", filePath)
		fmt.Println(newHeader)
		atomic.AddInt32(&p.modified, 1)
		return nil
	}

	// Fix mode: apply the header
	if err := ApplyHeader(filePath, newHeader); err != nil {
		return fmt.Errorf("applying header: %w", err)
	}

	atomic.AddInt32(&p.modified, 1)
	if p.verbose {
		fmt.Printf("Updated header: %s\n", filePath)
	}

	return nil
}

// printSummary prints a summary of the processing
func (p *Processor) printSummary() {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("Processed: %d files\n", p.processed)

	switch {
	case p.check:
		if p.needsUpdate > 0 {
			fmt.Printf("Files needing updates: %d\n", p.needsUpdate)
		} else {
			fmt.Println("All files have correct headers")
		}
	case p.dryRun:
		fmt.Printf("Files that would be modified: %d\n", p.modified)
	default:
		fmt.Printf("Modified: %d files\n", p.modified)
	}

	if p.failed > 0 {
		fmt.Printf("Failed: %d files\n", p.failed)
	}
	fmt.Println(strings.Repeat("=", 50))
}

func (p *Processor) ProcessTopLevelLicense() int {
	// Generate top-level licensing file
	if p.topLicense {
		description, err := p.detector.GetTopLevelLicenseDescription()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating top-level license: %v\n", err)
			return 1
		}

		switch {
		case p.check:
			passed, err := CheckTopLevelLicense(p.detector.repoRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error validating top level license %v\n", err)
				return 1
			}
			if !passed {
				fmt.Println("Missing or incorrect top-level LICENSING file.")
				return 1
			}
		case p.dryRun:
			fmt.Printf("Directories with proprietary LICENSE files:\n%s", description)
		default:
			err := GenerateTopLevelLicense(p.detector.repoRoot, description)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating top level license %v\n", err)
				return 1
			}
		}
	}
	return 0
}
