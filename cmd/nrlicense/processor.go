// Copyright New Relic, Inc.
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

	// Generate top-level licensing file
	if p.topLicense {
		proprietaryLicenseDescription, err := p.detector.GetProprietaryLicenseDescription()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating top-level license: %v\n", err)
			return 1
		}
		if p.dryRun {
			fmt.Printf("Directories with proprietary LICENSE files:\n%s", proprietaryLicenseDescription)
		} else {
			GenerateTopLevelLicense(p.detector.repoRoot, proprietaryLicenseDescription)
		}

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
			correct, err := CheckHeader(filePath, status, headerInfo.ExistingCopyright)
			if err != nil {
				return fmt.Errorf("checking header: %w", err)
			}
			if !correct {
				atomic.AddInt32(&p.needsUpdate, 1)
				fmt.Printf("Missing or incorrect header: %s\n", filePath)
			}
		}
		// In fix mode, don't modify unmodified files
		return nil
	}

	// Generate the appropriate header
	modDescription := ""
	if status == StatusModified {
		modDescription, err = p.detector.GetModificationDescription(filePath)
		if err != nil {
			// Non-fatal, use default
			modDescription = "Modified for New Relic distribution"
		}
	}

	newHeader, err := GenerateHeader(status, headerInfo.ExistingCopyright, modDescription, filePath)
	if err != nil {
		return fmt.Errorf("generating header: %w", err)
	}

	// If no header generated (e.g., for unmodified files), skip
	if newHeader == "" {
		return nil
	}

	// Check mode: verify header is correct
	if p.check {
		correct, err := CheckHeader(filePath, status, headerInfo.ExistingCopyright)
		if err != nil {
			return fmt.Errorf("checking header: %w", err)
		}
		if !correct {
			atomic.AddInt32(&p.needsUpdate, 1)
			fmt.Printf("Missing or incorrect header: %s\n", filePath)
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

	if p.check {
		if p.needsUpdate > 0 {
			fmt.Printf("Files needing updates: %d\n", p.needsUpdate)
		} else {
			fmt.Println("All files have correct headers")
		}
	} else if p.dryRun {
		fmt.Printf("Files that would be modified: %d\n", p.modified)
	} else {
		fmt.Printf("Modified: %d files\n", p.modified)
	}

	if p.failed > 0 {
		fmt.Printf("Failed: %d files\n", p.failed)
	}
	fmt.Println(strings.Repeat("=", 50))
}
