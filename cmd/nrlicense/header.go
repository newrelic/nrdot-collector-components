// Copyright New Relic, Inc.
// SPDX-License-Identifier: Apache-2.0

// Portions of this file are adapted from github.com/google/addlicense
// Copyright 2018 Google LLC, licensed under Apache 2.0.
// Functions adapted: hashBang, isGenerated, hasLicense, and comment style detection.

package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed templates/newrelic-header-apache.txt
var newrelicApacheHeaderTemplate string

//go:embed templates/newrelic-header-proprietary.txt
var newrelicProprietaryHeaderTemplate string

//go:embed templates/modified-header.txt
var modifiedHeaderTemplate string

//go:embed templates/top-level-license.txt
var topLevelLicenseTemplate string

// HeaderInfo contains information about a file's license header
type HeaderInfo struct {
	HasHeader              bool
	IsGenerated            bool
	ExistingCopyright      string
	ExistingSPDXIdentifier string
	HeaderLines            []string
	ContentStartLine       int
}

// ParseFileHeader analyzes a file's existing header
func ParseFileHeader(filePath string) (*HeaderInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	info := &HeaderInfo{
		HeaderLines: []string{},
	}

	// Check for generated files using improved detection
	if isGeneratedFile(content) {
		info.IsGenerated = true
		return info, nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	inHeader := false
	ext := filepath.Ext(filePath)

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Detect start of license header
		if !inHeader && isCommentLine(line, ext) {
			if containsCopyright(line) || containsSPDX(line) {
				inHeader = true
				info.HasHeader = true
				info.HeaderLines = append(info.HeaderLines, line)

				if containsCopyright(line) {
					info.ExistingCopyright = extractCopyright(line)
				}
				continue
			}
		}

		// Continue collecting header line
		if inHeader {
			if isCommentLine(line, ext) || strings.TrimSpace(line) == "" {
				// SPDX identifier is consider the last part of the header (to prevent deletion of non-header comments)
				if info.ExistingSPDXIdentifier != "" && !isEmptyOrEmptyComment(line, ext) {
					info.ContentStartLine = lineNum
					break
				}

				info.HeaderLines = append(info.HeaderLines, line)

				if containsCopyright(line) && info.ExistingCopyright == "" {
					info.ExistingCopyright = extractCopyright(line)
				}

				if containsSPDX(line) && info.ExistingSPDXIdentifier == "" {
					info.ExistingSPDXIdentifier = extractSPDXIdentifier(line)
				}

				continue
			}
			info.ContentStartLine = lineNum
			break
		}

		// If we hit non-comment content without finding a header, stop
		if !inHeader && strings.TrimSpace(line) != "" && !isCommentLine(line, ext) {
			info.ContentStartLine = lineNum
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// If we're still in header at EOF, set content start to end
	if inHeader && info.ContentStartLine == 0 {
		info.ContentStartLine = lineNum + 1
	}

	return info, nil
}

// GenerateHeader creates the appropriate header based on file status
func GenerateHeader(status FileStatus, modDescription, filePath string) (string, error) {
	ext := filepath.Ext(filePath)

	switch status {
	case StatusUnmodified:
		// Keep original header - return empty to signal no change needed
		return "", nil

	case StatusModified:
		return generateModifiedHeader(modDescription, ext), nil

	case StatusNewApache:
		return generateNewApacheHeader(ext), nil

	case StatusNewProprietary:
		return generateNewProprietaryHeader(ext), nil

	default:
		return "", fmt.Errorf("unknown file status: %v", status)
	}
}

// generateModifiedHeader creates a dual copyright header for modified files
func generateModifiedHeader(modDescription, ext string) string {
	comment := getCommentStyle(ext)

	// Use template and replace placeholder
	template := modifiedHeaderTemplate
	if modDescription != "" && modDescription != "Modified for New Relic distribution" {
		template = strings.Replace(template, "{{DESCRIPTION}}", modDescription, 1)
	} else {
		// Remove the placeholder line if no custom description
		template = strings.Replace(template, "{{DESCRIPTION}}\n", "", 1)
	}

	// Add comment prefix to each line
	lines := strings.Split(strings.TrimSpace(template), "\n")
	var buf bytes.Buffer
	for _, line := range lines {
		if line == "" {
			buf.WriteString(comment + "\n")
		} else {
			buf.WriteString(comment + " " + line + "\n")
		}
	}

	return buf.String()
}

// generateNewHeader creates a header for new files
func generateNewApacheHeader(ext string) string {
	comment := getCommentStyle(ext)

	// Use template
	lines := strings.Split(strings.TrimSpace(newrelicApacheHeaderTemplate), "\n")
	var buf bytes.Buffer
	for _, line := range lines {
		if line == "" {
			buf.WriteString(comment + "\n")
		} else {
			buf.WriteString(comment + " " + line + "\n")
		}
	}

	return buf.String()
}

// generateNewHeader creates a header for new files
func generateNewProprietaryHeader(ext string) string {
	comment := getCommentStyle(ext)

	// Use template
	lines := strings.Split(strings.TrimSpace(newrelicProprietaryHeaderTemplate), "\n")
	var buf bytes.Buffer
	for _, line := range lines {
		if line == "" {
			buf.WriteString(comment + "\n")
		} else {
			buf.WriteString(comment + " " + line + "\n")
		}
	}

	return buf.String()
}

// isCommentLine checks if a line is a comment based on file extension
func isCommentLine(line, ext string) bool {
	trimmed := strings.TrimSpace(line)

	switch ext {
	case ".go", ".proto":
		return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*")
	case ".sh", ".yaml", ".yml":
		return strings.HasPrefix(trimmed, "#")
	default:
		// Default to checking both styles
		return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#")
	}
}

// isEmptyOrEmptyComment checks if a line is blank, or a blank comment e.g. "//"
func isEmptyOrEmptyComment(line, ext string) bool {
	_, prefix, _ := getCommentPrefix(ext)
	prefix = strings.TrimSpace(prefix)
	comment := strings.TrimPrefix(strings.TrimSpace(line), prefix)
	return comment == ""
}

// containsCopyright checks if a line contains copyright information
func containsCopyright(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "copyright")
}

// containsSPDX checks if a line contains SPDX identifier
func containsSPDX(line string) bool {
	return strings.Contains(line, "SPDX-License-Identifier")
}

// extractCopyright extracts the copyright holder from a copyright line
func extractCopyright(line string) string {
	line = removeCommentMarkers(line)

	// Remove "Copyright" prefixs
	if idx := strings.Index(strings.ToLower(line), "copyright"); idx >= 0 {
		line = line[idx+9:] // len("copyright") = 9
		line = strings.TrimSpace(line)
	}

	return line
}

// extractSPDX extracts the SPDX itentifier from the appropriate line
func extractSPDXIdentifier(line string) string {
	line = removeCommentMarkers(line)
	line = strings.TrimPrefix(line, "SPDX-License-Identifier:")
	line = strings.TrimSpace(line)
	return line
}

// removeCommentMarkers removes any comment markers from a line.
func removeCommentMarkers(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "//")
	line = strings.TrimPrefix(line, "/*")
	line = strings.TrimPrefix(line, "*")
	line = strings.TrimPrefix(line, "#")
	line = strings.TrimSpace(line)
	return line
}

// getCommentStyle returns the comment prefix for a file extension.
// This is a simplified wrapper around getCommentPrefix for single-line headers.
func getCommentStyle(ext string) string {
	_, mid, _ := getCommentPrefix("file" + ext)
	if mid == "" {
		return "//" // Default fallback
	}
	return strings.TrimSpace(mid)
}

// ApplyHeader replaces the header in a file
func ApplyHeader(filePath, newHeader string) error {
	// Get original file info to preserve permissions
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	originalMode := fileInfo.Mode()

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Check if file is generated
	if isGeneratedFile(content) {
		return nil
	}

	// Check for existing license
	if !hasLicenseHeader(content) {
		// No existing header, just add new one after shebang
		var newContent bytes.Buffer

		// Preserve shebang/special directives
		shebang := hashBang(content)
		if len(shebang) > 0 {
			newContent.Write(shebang)
			content = content[len(shebang):]
			// Ensure newline after shebang
			if shebang[len(shebang)-1] != '\n' {
				newContent.WriteString("\n")
			}
		}

		// Write header
		newContent.WriteString(newHeader)
		if !strings.HasSuffix(newHeader, "\n\n") {
			newContent.WriteString("\n")
		}

		// Write rest of content
		newContent.Write(content)

		return os.WriteFile(filePath, newContent.Bytes(), originalMode)
	}

	// Parse existing header
	headerInfo, err := ParseFileHeader(filePath)
	if err != nil {
		return err
	}

	var newContent bytes.Buffer

	// Preserve shebang/special directives at the very beginning
	shebang := hashBang(content)
	if len(shebang) > 0 {
		newContent.Write(shebang)
	}

	// Write new header
	newContent.WriteString(newHeader)

	// Add blank line after header if needed
	if !strings.HasSuffix(newHeader, "\n\n") {
		newContent.WriteString("\n")
	}

	// Write the rest of the file (skip old header and shebang if present)
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	hasShebang := len(shebang) > 0

	for scanner.Scan() {
		lineNum++

		// Skip the shebang line if present (we already wrote it)
		if lineNum == 1 && hasShebang {
			continue
		}

		// Skip header lines
		if lineNum < headerInfo.ContentStartLine {
			continue
		}

		// Write content lines
		newContent.WriteString(scanner.Text())
		newContent.WriteString("\n")
	}

	// Write back to file
	return os.WriteFile(filePath, newContent.Bytes(), originalMode)
}

// ReadFileContent reads the content of a file after the header
func ReadFileContent(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	headerInfo, err := ParseFileHeader(filePath)
	if err != nil {
		return nil, err
	}

	// Return content starting from after the header
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var result bytes.Buffer
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum >= headerInfo.ContentStartLine {
			result.WriteString(scanner.Text())
			result.WriteString("\n")
		}
	}

	return result.Bytes(), scanner.Err()
}

// CheckHeader verifies if a file has the correct header for its status
func CheckHeader(filePath string, status FileStatus) (bool, error) {
	headerInfo, err := ParseFileHeader(filePath)
	if err != nil {
		return false, err
	}

	// Generated files are always OK
	if headerInfo.IsGenerated {
		return true, nil
	}

	switch status {
	case StatusUnmodified:
		// Should have original copyright only
		return headerInfo.HasHeader && !strings.Contains(headerInfo.ExistingCopyright, "New Relic"), nil

	case StatusModified:
		// Should have dual copyright
		hasOriginal := headerInfo.HasHeader && headerInfo.ExistingCopyright != ""
		hasNewRelic := false
		for _, line := range headerInfo.HeaderLines {
			if strings.Contains(line, "New Relic") {
				hasNewRelic = true
				break
			}
		}
		// original license must not be modified
		hasApacheLicense := strings.Contains(headerInfo.ExistingSPDXIdentifier, "Apache-2.0")
		return hasOriginal && hasNewRelic && hasApacheLicense, nil

	case StatusNewApache:
		// Should have New Relic copyright only, with apache 2.0 license
		correctCopyright := strings.Contains(headerInfo.ExistingCopyright, "New Relic")
		correctSPDXIdentifier := strings.Contains(headerInfo.ExistingSPDXIdentifier, "Apache-2.0")
		return headerInfo.HasHeader && correctCopyright && correctSPDXIdentifier, nil

	case StatusNewProprietary:
		// Should have New Relic hopyright only, with NR proprietary license
		correctCopyright := headerInfo.HasHeader && strings.Contains(headerInfo.ExistingCopyright, "New Relic")
		correctSPDXIdentifier := strings.Contains(headerInfo.ExistingSPDXIdentifier, "New-Relic-Software-License")
		return headerInfo.HasHeader && correctCopyright && correctSPDXIdentifier, nil

	default:
		return false, nil
	}
}

// GenerateTopLevelLicense generates the top-level license file at the root directory
func GenerateTopLevelLicense(rootDir, description string) error {
	// Use template and replace placeholder
	licenseFileName := fmt.Sprintf("%s/LICENSING", rootDir)
	template := topLevelLicenseTemplate
	if description != "" {
		template = strings.Replace(template, "{{DESCRIPTION}}", description, 1)
	} else {
		// Remove the placeholder line if no custom description
		template = strings.Replace(template, "{{DESCRIPTION}}\n", "", 1)
	}
	err := os.WriteFile(licenseFileName, []byte(template), 0o600)
	if err != nil {
		return err
	}
	return nil
}

// CheckTopLevelLicense validates the top level licensing file.
func CheckTopLevelLicense(rootDir string) (bool, error) {
	licenseFileName := fmt.Sprintf("%s/LICENSING", rootDir)

	// Check the existence of the file
	matches, err := filepath.Glob(licenseFileName)
	if err != nil || len(matches) != 1 {
		return false, err
	}

	// Check that all directories listed in LICENSING exist w/ correct license
	content, err := os.ReadFile(licenseFileName)
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(content), "\n")

	validated := true
	for _, line := range lines[2:] {
		licensedDir := strings.TrimPrefix(line, "New Relic Software License -")
		licensedDir = strings.TrimSpace(licensedDir)
		path := fmt.Sprintf("%s/%s", rootDir, licensedDir)

		var matches []string
		matches, err = filepath.Glob(fmt.Sprintf("%s/LICENSE_NEWRELIC_*", path))
		if err != nil {
			return false, err
		}
		if len(matches) < 1 {
			validated = false
			fmt.Printf("Directory listed in LICENSING doesn't exist or contains missing or incorrect license file: %s\n", licensedDir)
		}
	}

	return validated, err
}

// ==================== Functions adapted from google/addlicense ====================

// Shebang prefix that should be preserved at the top of files.
// Adapted from google/addlicense, simplified for this repository.
var fileHeaderPrefixes = []string{
	"#!", // shell script shebang
}

// hashBang extracts the first line of a file if it starts with a special directive.
// These lines should be preserved before the license header.
// Adapted from google/addlicense.
func hashBang(content []byte) []byte {
	var line []byte
	for _, c := range content {
		line = append(line, c)
		if c == '\n' {
			break
		}
	}
	first := strings.ToLower(string(line))
	for _, prefix := range fileHeaderPrefixes {
		if strings.HasPrefix(first, prefix) {
			return line
		}
	}
	return nil
}

// Regular expressions for detecting generated files.
// Adapted from google/addlicense.
var (
	// go generate: ^// Code generated .* DO NOT EDIT\.$
	goGenerated = regexp.MustCompile(`(?m)^.{1,3} Code generated .* DO NOT EDIT\.$`)
	// cargo raze: ^DO NOT EDIT! Replaced on runs of cargo-raze$
	cargoRazeGenerated = regexp.MustCompile(`(?m)^DO NOT EDIT! Replaced on runs of cargo-raze$`)
)

// isGeneratedFile returns true if the content contains markers indicating
// the file was auto-generated and should not be modified.
// Adapted from google/addlicense.
func isGeneratedFile(content []byte) bool {
	return goGenerated.Match(content) || cargoRazeGenerated.Match(content)
}

// hasLicenseHeader checks if content contains a license header in the first 1000 bytes.
// This is much more efficient than scanning the entire file.
// Adapted from google/addlicense.
func hasLicenseHeader(content []byte) bool {
	n := 1000
	if len(content) < 1000 {
		n = len(content)
	}
	header := bytes.ToLower(content[:n])
	return bytes.Contains(header, []byte("copyright")) ||
		bytes.Contains(header, []byte("mozilla public")) ||
		bytes.Contains(header, []byte("spdx-license-identifier"))
}

// getCommentPrefix returns the appropriate comment style for a file based on its name.
// Returns three strings: top (opening delimiter), mid (line prefix), bot (closing delimiter).
// Adapted from google/addlicense, simplified for this repository's file types.
//
//nolint:unparam // Reason: Keep flexible for block comments
func getCommentPrefix(filename string) (top, mid, bot string) {
	ext := filepath.Ext(filename)

	switch ext {
	case ".go", ".proto":
		// Go and protobuf use C++ style comments
		return "", "// ", ""

	case ".sh", ".bash", ".zsh", ".yaml", ".yml":
		// Shell scripts and YAML use hash comments
		return "", "# ", ""

	default:
		// Default to Go-style comments for unknown types
		return "", "// ", ""
	}
}
