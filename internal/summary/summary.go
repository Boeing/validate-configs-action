package summary

import (
	"fmt"
	"os"
	"strings"

	"github.com/Boeing/config-file-validator/v3/pkg/reporter"
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
	unformatted := 0
	var failedReports []reporter.Report
	var unformattedReports []reporter.Report
	for i := range reports {
		switch reports[i].Status {
		case reporter.StatusPass:
			passed++
		case reporter.StatusFail:
			failed++
			failedReports = append(failedReports, reports[i])
		case reporter.StatusUnformatted:
			unformatted++
			unformattedReports = append(unformattedReports, reports[i])
		}
	}

	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_WRONLY, 0o644) //nolint:gosec // GITHUB_STEP_SUMMARY is a trusted path set by the GitHub runner
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck // best-effort file close

	if failed == 0 && unformatted == 0 {
		_, _ = fmt.Fprintf(f, "### ✅ Config Validation Passed\n\n")
		_, _ = fmt.Fprintf(f, "All **%d** configuration files are valid.\n", total)
		return
	}

	if failed > 0 {
		_, _ = fmt.Fprintf(f, "### ❌ Config Validation Failed\n\n")
	} else {
		_, _ = fmt.Fprintf(f, "### ⚠️ Config Formatting Issues\n\n")
	}

	_, _ = fmt.Fprintf(f, "| | Count |\n|---|---|\n")
	_, _ = fmt.Fprintf(f, "| ✅ Passed | %d |\n", passed)
	if failed > 0 {
		_, _ = fmt.Fprintf(f, "| ❌ Failed | %d |\n", failed)
	}
	if unformatted > 0 {
		_, _ = fmt.Fprintf(f, "| ⚠️ Needs formatting | %d |\n", unformatted)
	}
	_, _ = fmt.Fprintf(f, "| **Total** | **%d** |\n\n", total)

	if failed > 0 {
		_, _ = fmt.Fprintf(f, "#### Failed Files\n\n")
		_, _ = fmt.Fprintf(f, "| File | Errors |\n|---|---|\n")
		for i := range failedReports {
			path := strings.TrimPrefix(failedReports[i].FilePath, "/github/workspace/")
			var msgs []string
			for _, issue := range failedReports[i].Issues {
				msgs = append(msgs, issue.Message)
			}
			errors := strings.Join(msgs, "<br>")
			_, _ = fmt.Fprintf(f, "| `%s` | %s |\n", path, errors)
		}
		_, _ = fmt.Fprintf(f, "\n")
	}

	if unformatted > 0 {
		_, _ = fmt.Fprintf(f, "#### Formatting Issues\n\n")
		_, _ = fmt.Fprintf(f, "| File |\n|---|\n")
		for i := range unformattedReports {
			path := strings.TrimPrefix(unformattedReports[i].FilePath, "/github/workspace/")
			_, _ = fmt.Fprintf(f, "| `%s` |\n", path)
		}
		_, _ = fmt.Fprintf(f, "\n")
	}
}
