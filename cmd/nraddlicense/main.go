package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const tmpl_nr_apache = `Copyright New Relic, Inc.
SPDX-License-Identifier: Apache-2.0`

const tmpl_nr_proprietary = `Copyright New Relic, Inc.
SPDX-License-Identifier: New-Relic-Software`

const tmpl_otel_amended = `Copyright The OpenTelemetry Authors
Modifications copyright New Relic, Inc.

Modifications:
%s

SPDX-License-Identifier: Apache-2.0
`

// skipSlice stores repeated -skip flags in a format ready to be passed directly to CLI
type skipSlice []string

func (s *skipSlice) String() string {
	return fmt.Sprint(*s)
}

func (s *skipSlice) Set(value string) error {
	*s = append(*s, "-skip", value)
	return nil
}

// ignoreSlice stores repeated -ignroe flags in a format ready to be passed directly to CLI
type ignoreSlice []string

func (i *ignoreSlice) String() string {
	return fmt.Sprint(*i)
}

func (i *ignoreSlice) Set(value string) error {
	*i = append(*i, "-ignore", value)
	return nil
}

var (
	ROOT_DIR   string
	ADDLICENSE string
)

var (
	ignorePassthrough ignoreSlice
	skipPassthrough   skipSlice
	// Flags used in this program
	holderFlag  = flag.String("c", "", "copyright holder")
	licenseFlag = flag.String("l", "", "license type: apache, bsd, mit, mpl, newrelic")
	// These flags only exist to stop flag.Parse() from complaining, so they may be passed to addlicense.
	_ = flag.String("f", "", "license file")
	_ = flag.String("y", fmt.Sprint(time.Now().Year()), "copyright year(s)")
	_ = flag.Bool("v", false, "verbose mode: print the name of the files that are modified or were skipped")
	_ = flag.Bool("check", false, "check only mode: verify presence of license headers and exit with non-zero code if missing")
	_ = flag.String("s", "", "Include SPDX identifier in license header. Set -s=only to only include SPDX identifier.")
)

func init() {
	flag.Var(&skipPassthrough, "skip", "[deprecated: see -ignore] file extensions to skip, for example: -skip rb -skip go")
	flag.Var(&ignorePassthrough, "ignore", "file patterns to ignore, for example: -ignore **/*.go -ignore vendor/**")
	ROOT_DIR = execCommand("git", "rev-parse", "--show-toplevel")
	ADDLICENSE = fmt.Sprintf("%s/.tools/addlicense", ROOT_DIR)
}

func main() {
	flag.Parse()
	isNewRelic := (*holderFlag == "New Relic")

	file := flag.Arg(flag.NArg() - 1)
	content, _ := os.ReadFile(file)
	//otelAuthored := strings.Contains(string(content), "Copyright The OpenTelemetry Authors")
	// git diff with last pre-fork commit hash
	// diff := execCommand("git", "diff", "51061db5838300734ff23888e2396263f61146d9", "--", file)
	// hasDiff := len(diff) > 0


	// Skip generated files
	if strings.Contains(string(content), "GENERATED") || strings.Contains(string(content), "generated") {
		return
	}

	// Create temp template file with appropriate license
	tmpFile, _ := os.CreateTemp("", "license-*.tmpl")
	defer os.Remove(tmpFile.Name())
	if isNewRelic {
		template := ""
		switch *licenseFlag {
		case "newrelic":
			template = tmpl_nr_proprietary
		case "apache":
			template = tmpl_nr_apache
		default:
			log.Fatalf("Incorrect license type %s used with copyright holder \"New Relic\". Please use either \"newrelic\" or \"apache\".", *licenseFlag)
		}
		_, err := tmpFile.WriteString(template)
		if err != nil {
			log.Fatal(err)
		}
		tmpFile.Close()
	}

	// Pass on all flags, replacing -c and -l with new relic header template
	passedFlags := []string{}
	if isNewRelic {
		passedFlags = append(passedFlags, "-f", tmpFile.Name())
	}
	flag.Visit(func(f *flag.Flag) {
		// Skip custom slice types (handled separately)
		if f.Name == "skip" || f.Name == "ignore" {
			return
		}

		// Skip c, l, f flags when using New Relic license
		if isNewRelic && (f.Name == "c" || f.Name == "l" || f.Name == "f") {
			return
		}

		passedFlags = append(passedFlags, "-"+f.Name, f.Value.String())
	})
	passedFlags = append(passedFlags, ignorePassthrough...)
	passedFlags = append(passedFlags, skipPassthrough...)

	_ = execCommand(ADDLICENSE, append(passedFlags, flag.Args()...)...)
}

func removeCopyrightHeader(filepath string) error {
	content, _ := os.ReadFile(filepath)
	lines := strings.Split(string(content), "\n")
	start := -1
	end := -1
	for idx, line := range lines {
		lower := strings.ToLower(line)
		if start == -1 && strings.Contains(lower, "copyright") {
			start = idx
		} else if start != -1 && strings.Contains(lower, "spdx") {
			end = idx
			break
		}
	}

	var filtered []string
	if start != -1 && end != -1 {
		// Remove lines from start to end (inclusive)
		filtered = append(lines[:start], lines[end+1:]...)
	} else {
		filtered = lines
	}

	return os.WriteFile(filepath, []byte(strings.Join(filtered, "\n")), 0644)
}

func execCommand(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Error: %v\nOutput: %s\n", err, output)
	}
	return strings.TrimSpace(string(output))
}
