package output

import (
	"fmt"
	"os"

	"github.com/Boeing/config-file-validator/v3/pkg/reporter"
)

// WriteOutputs writes action outputs (files-validated, files-failed,
// files-unformatted, exit-code) to the GITHUB_OUTPUT file.
func WriteOutputs(reports []reporter.Report, exitCode int) {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		return
	}

	total := len(reports)
	failed := 0
	unformatted := 0
	for i := range reports {
		switch reports[i].Status {
		case reporter.StatusFail:
			failed++
		case reporter.StatusUnformatted:
			unformatted++
		}
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY, 0o644) //nolint:gosec // GITHUB_OUTPUT is a trusted path set by the GitHub runner
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write to GITHUB_OUTPUT: %v\n", err)
		return
	}
	defer f.Close() //nolint:errcheck // best-effort file close

	_, _ = fmt.Fprintf(f, "files-validated=%d\n", total)
	_, _ = fmt.Fprintf(f, "files-failed=%d\n", failed)
	_, _ = fmt.Fprintf(f, "files-unformatted=%d\n", unformatted)
	_, _ = fmt.Fprintf(f, "exit-code=%d\n", exitCode)
}
