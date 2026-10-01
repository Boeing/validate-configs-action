package output

import (
	"fmt"
	"os"

	"github.com/Boeing/config-file-validator/v3/pkg/reporter"
)

// WriteOutputs writes action outputs (files-validated, files-failed, exit-code)
// to the GITHUB_OUTPUT file.
func WriteOutputs(reports []reporter.Report, exitCode int) {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		return
	}

	total := len(reports)
	failed := 0
	for i := range reports {
		if reports[i].Status == reporter.StatusFail {
			failed++
		}
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY, 0o644) //nolint:gosec // GITHUB_OUTPUT is a trusted path set by the GitHub runner
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck // best-effort file close

	_, _ = fmt.Fprintf(f, "files-validated=%d\n", total)
	_, _ = fmt.Fprintf(f, "files-failed=%d\n", failed)
	_, _ = fmt.Fprintf(f, "exit-code=%d\n", exitCode)
}
