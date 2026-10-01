package summary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Boeing/config-file-validator/v2/pkg/reporter"
)

// helper creates a temp file for GITHUB_STEP_SUMMARY and sets the env var.
// Returns the path so callers can read the file after WriteJobSummary runs.
func setupSummaryFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "summary.md")
	f, err := os.Create(path) //nolint:gosec // test helper writing to temp dir
	if err != nil {
		t.Fatalf("create summary file: %v", err)
	}
	f.Close() //nolint:errcheck,gosec // test helper, close error is non-critical
	t.Setenv("GITHUB_STEP_SUMMARY", path)
	return path
}

func readSummary(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // test helper reading from temp dir
	if err != nil {
		t.Fatalf("read summary file: %v", err)
	}
	return string(b)
}

func TestWriteJobSummary_EnvNotSet(t *testing.T) {
	// Ensure the env var is explicitly unset.
	t.Setenv("GITHUB_STEP_SUMMARY", "")

	// Should return immediately without panic or error.
	WriteJobSummary([]reporter.Report{
		{FilePath: "/some/file.json", FileName: "file.json", IsValid: true},
	})
}

func TestWriteJobSummary_AllPassing(t *testing.T) {
	path := setupSummaryFile(t)

	reports := []reporter.Report{
		{FilePath: "/repo/config.json", FileName: "config.json", IsValid: true},
		{FilePath: "/repo/settings.yaml", FileName: "settings.yaml", IsValid: true},
		{FilePath: "/repo/data.toml", FileName: "data.toml", IsValid: true},
	}

	WriteJobSummary(reports)

	got := readSummary(t, path)

	if !strings.Contains(got, "Config Validation Passed") {
		t.Error("expected 'Config Validation Passed' header")
	}
	if !strings.Contains(got, "**3**") {
		t.Error("expected count of 3 valid files")
	}
	if strings.Contains(got, "Config Validation Failed") {
		t.Error("should not contain failure header when all pass")
	}
	if strings.Contains(got, "Failed Files") {
		t.Error("should not contain failed files table when all pass")
	}
}

func TestWriteJobSummary_SomeFailures(t *testing.T) {
	path := setupSummaryFile(t)

	reports := []reporter.Report{
		{FilePath: "/repo/good.json", FileName: "good.json", IsValid: true},
		{
			FilePath:         "/repo/bad.yaml",
			FileName:         "bad.yaml",
			IsValid:          false,
			ValidationErrors: []string{"syntax error at line 5"},
		},
		{
			FilePath:         "/repo/broken.toml",
			FileName:         "broken.toml",
			IsValid:          false,
			ValidationErrors: []string{"unexpected key"},
		},
	}

	WriteJobSummary(reports)

	got := readSummary(t, path)

	if !strings.Contains(got, "Config Validation Failed") {
		t.Error("expected 'Config Validation Failed' header")
	}
	// Summary counts table
	if !strings.Contains(got, "Passed | 1") {
		t.Error("expected 1 passed in summary table")
	}
	if !strings.Contains(got, "Failed | 2") {
		t.Error("expected 2 failed in summary table")
	}
	if !strings.Contains(got, "**Total** | **3**") {
		t.Error("expected total of 3 in summary table")
	}
	// Failed files table
	if !strings.Contains(got, "Failed Files") {
		t.Error("expected 'Failed Files' section")
	}
	if !strings.Contains(got, "`/repo/bad.yaml`") {
		t.Error("expected bad.yaml in failed files table")
	}
	if !strings.Contains(got, "syntax error at line 5") {
		t.Error("expected validation error for bad.yaml")
	}
	if !strings.Contains(got, "`/repo/broken.toml`") {
		t.Error("expected broken.toml in failed files table")
	}
	if !strings.Contains(got, "unexpected key") {
		t.Error("expected validation error for broken.toml")
	}
	// Should not contain the all-pass message
	if strings.Contains(got, "Config Validation Passed") {
		t.Error("should not contain passed header when there are failures")
	}
}

func TestWriteJobSummary_WorkspacePathStripped(t *testing.T) {
	path := setupSummaryFile(t)

	reports := []reporter.Report{
		{
			FilePath:         "/github/workspace/src/config.json",
			FileName:         "config.json",
			IsValid:          false,
			ValidationErrors: []string{"invalid JSON"},
		},
		{
			FilePath:         "/github/workspace/deep/nested/file.yaml",
			FileName:         "file.yaml",
			IsValid:          false,
			ValidationErrors: []string{"bad indent"},
		},
		{
			// Path that does NOT have the workspace prefix should be kept as-is.
			FilePath:         "/other/path/file.toml",
			FileName:         "file.toml",
			IsValid:          false,
			ValidationErrors: []string{"missing value"},
		},
	}

	WriteJobSummary(reports)

	got := readSummary(t, path)

	// The /github/workspace/ prefix should be stripped.
	if !strings.Contains(got, "`src/config.json`") {
		t.Error("expected /github/workspace/ prefix stripped from src/config.json")
	}
	if !strings.Contains(got, "`deep/nested/file.yaml`") {
		t.Error("expected /github/workspace/ prefix stripped from deep/nested/file.yaml")
	}
	// The full original prefix should not appear.
	if strings.Contains(got, "/github/workspace/") {
		t.Error("workspace prefix should be stripped from all paths that have it")
	}
	// Non-workspace path should remain unchanged.
	if !strings.Contains(got, "`/other/path/file.toml`") {
		t.Error("expected non-workspace path to remain unchanged")
	}
}

func TestWriteJobSummary_EmptyReports(t *testing.T) {
	path := setupSummaryFile(t)

	WriteJobSummary([]reporter.Report{})

	got := readSummary(t, path)

	if !strings.Contains(got, "Config Validation Passed") {
		t.Error("expected 'Config Validation Passed' for empty reports")
	}
	if !strings.Contains(got, "**0**") {
		t.Error("expected count of 0 for empty reports")
	}
}

func TestWriteJobSummary_MultipleErrorsJoinedWithBr(t *testing.T) {
	path := setupSummaryFile(t)

	reports := []reporter.Report{
		{
			FilePath: "/repo/multi-err.json",
			FileName: "multi-err.json",
			IsValid:  false,
			ValidationErrors: []string{
				"missing required field 'name'",
				"invalid type for 'version'",
				"additional property 'foo' not allowed",
			},
		},
	}

	WriteJobSummary(reports)

	got := readSummary(t, path)

	// Errors should be joined with <br>
	if !strings.Contains(got, "missing required field 'name'<br>invalid type for 'version'<br>additional property 'foo' not allowed") {
		t.Error("expected multiple errors joined with <br>")
	}
	// Each individual error should be present
	for _, e := range reports[0].ValidationErrors {
		if !strings.Contains(got, e) {
			t.Errorf("expected error %q in output", e)
		}
	}
}

func TestWriteJobSummary_NilReports(t *testing.T) {
	path := setupSummaryFile(t)

	WriteJobSummary(nil)

	got := readSummary(t, path)

	if !strings.Contains(got, "Config Validation Passed") {
		t.Error("expected 'Config Validation Passed' for nil reports")
	}
	if !strings.Contains(got, "**0**") {
		t.Error("expected count of 0 for nil reports")
	}
}
