package summary

import (
	"fmt"
	"os"
	"strings"

	"github.com/Boeing/config-file-validator/v2/pkg/reporter"
)

// WriteJobSummary writes a markdown summary to GITHUB_STEP_SUMMARY.
func WriteJobSummary(reports []reporter.Report) {
	summaryFile := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryFile == "" {
		return
	}

	total := len(reports)
	passed := 0
	failed := 0
	var failedReports []reporter.Report
	for i := range reports {
		if reports[i].IsValid {
			passed++
		} else {
			failed++
			failedReports = append(failedReports, reports[i])
		}
	}

	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_WRONLY, 0o644) //nolint:gosec // GITHUB_STEP_SUMMARY is a trusted path set by the GitHub runner
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck // best-effort file close

	if failed == 0 {
		_, _ = fmt.Fprintf(f, "### ✅ Config Validation Passed\n\n")
		_, _ = fmt.Fprintf(f, "All **%d** configuration files are valid.\n", total)
		return
	}

	_, _ = fmt.Fprintf(f, "### ❌ Config Validation Failed\n\n")
	_, _ = fmt.Fprintf(f, "| | Count |\n|---|---|\n")
	_, _ = fmt.Fprintf(f, "| ✅ Passed | %d |\n", passed)
	_, _ = fmt.Fprintf(f, "| ❌ Failed | %d |\n", failed)
	_, _ = fmt.Fprintf(f, "| **Total** | **%d** |\n\n", total)

	_, _ = fmt.Fprintf(f, "#### Failed Files\n\n")
	_, _ = fmt.Fprintf(f, "| File | Errors |\n|---|---|\n")
	for i := range failedReports {
		path := strings.TrimPrefix(failedReports[i].FilePath, "/github/workspace/")
		errors := strings.Join(failedReports[i].ValidationErrors, "<br>")
		_, _ = fmt.Fprintf(f, "| `%s` | %s |\n", path, errors)
	}
}
