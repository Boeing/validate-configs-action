package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Boeing/config-file-validator/v3/pkg/reporter"
)

// helper creates a temp file for GITHUB_OUTPUT, sets the env var via t.Setenv
// (auto-restored on cleanup), calls WriteOutputs, and returns the file contents.
func runWriteOutputs(t *testing.T, reports []reporter.Report, exitCode int) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "github_output")
	// The file must exist because WriteOutputs opens with O_APPEND|O_WRONLY.
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("creating temp GITHUB_OUTPUT file: %v", err)
	}

	t.Setenv("GITHUB_OUTPUT", path)

	WriteOutputs(reports, exitCode)

	data, err := os.ReadFile(path) //nolint:gosec // test helper reading from temp dir
	if err != nil {
		t.Fatalf("reading GITHUB_OUTPUT file: %v", err)
	}
	return string(data)
}

// assertContains checks that got contains the expected substring.
func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("output missing %q\ngot:\n%s", want, got)
	}
}

func TestWriteOutputs_GithubOutputNotSet(t *testing.T) {
	// Ensure GITHUB_OUTPUT is unset; t.Setenv restores the original value.
	t.Setenv("GITHUB_OUTPUT", "")

	// Should not panic or create any file.
	WriteOutputs([]reporter.Report{{Status: reporter.StatusPass}}, 0)
}

func TestWriteOutputs_AllValid(t *testing.T) {
	reports := []reporter.Report{
		{FilePath: "a", FileName: "a.json", Status: reporter.StatusPass},
		{FilePath: "b", FileName: "b.yaml", Status: reporter.StatusPass},
		{FilePath: "c", FileName: "c.toml", Status: reporter.StatusPass},
	}

	got := runWriteOutputs(t, reports, 0)

	assertContains(t, got, "files-validated=3\n")
	assertContains(t, got, "files-failed=0\n")
	assertContains(t, got, "exit-code=0\n")
}

func TestWriteOutputs_MixedValidAndInvalid(t *testing.T) {
	reports := []reporter.Report{
		{FilePath: "a", FileName: "a.json", Status: reporter.StatusPass},
		{FilePath: "b", FileName: "b.yaml", Status: reporter.StatusFail, Issues: []reporter.Issue{{Type: reporter.IssueTypeSyntax, Message: "bad indent"}}},
		{FilePath: "c", FileName: "c.toml", Status: reporter.StatusPass},
		{FilePath: "d", FileName: "d.xml", Status: reporter.StatusFail, Issues: []reporter.Issue{{Type: reporter.IssueTypeSyntax, Message: "unclosed tag"}}},
		{FilePath: "e", FileName: "e.ini", Status: reporter.StatusPass},
	}

	got := runWriteOutputs(t, reports, 1)

	assertContains(t, got, "files-validated=5\n")
	assertContains(t, got, "files-failed=2\n")
	assertContains(t, got, "exit-code=1\n")
}

func TestWriteOutputs_AllInvalid(t *testing.T) {
	reports := []reporter.Report{
		{FilePath: "a", FileName: "a.json", Status: reporter.StatusFail, Issues: []reporter.Issue{{Type: reporter.IssueTypeSyntax, Message: "err1"}}},
		{FilePath: "b", FileName: "b.yaml", Status: reporter.StatusFail, Issues: []reporter.Issue{{Type: reporter.IssueTypeSyntax, Message: "err2"}}},
		{FilePath: "c", FileName: "c.toml", Status: reporter.StatusFail, Issues: []reporter.Issue{{Type: reporter.IssueTypeSyntax, Message: "err3"}}},
	}

	got := runWriteOutputs(t, reports, 1)

	assertContains(t, got, "files-validated=3\n")
	assertContains(t, got, "files-failed=3\n")
	assertContains(t, got, "exit-code=1\n")
}

func TestWriteOutputs_EmptyReports(t *testing.T) {
	got := runWriteOutputs(t, []reporter.Report{}, 0)

	assertContains(t, got, "files-validated=0\n")
	assertContains(t, got, "files-failed=0\n")
	assertContains(t, got, "exit-code=0\n")
}

func TestWriteOutputs_ExitCode1(t *testing.T) {
	reports := []reporter.Report{
		{FilePath: "a", FileName: "a.json", Status: reporter.StatusFail, Issues: []reporter.Issue{{Type: reporter.IssueTypeSyntax, Message: "syntax error"}}},
	}

	got := runWriteOutputs(t, reports, 1)

	assertContains(t, got, "exit-code=1\n")
}

func TestWriteOutputs_ExitCode2(t *testing.T) {
	// Exit code 2 signals a runtime error regardless of report content.
	got := runWriteOutputs(t, []reporter.Report{}, 2)

	assertContains(t, got, "exit-code=2\n")
}

func TestWriteOutputs_NilReports(t *testing.T) {
	got := runWriteOutputs(t, nil, 0)

	assertContains(t, got, "files-validated=0\n")
	assertContains(t, got, "files-failed=0\n")
	assertContains(t, got, "exit-code=0\n")
}

func TestWriteOutputs_ExactFormat(t *testing.T) {
	// Verify the entire output matches the expected format exactly —
	// three lines, each terminated by a newline.
	reports := []reporter.Report{
		{FilePath: "x", FileName: "x.json", Status: reporter.StatusPass},
		{FilePath: "y", FileName: "y.json", Status: reporter.StatusFail, Issues: []reporter.Issue{{Type: reporter.IssueTypeSyntax, Message: "bad"}}},
	}

	got := runWriteOutputs(t, reports, 1)

	want := "files-validated=2\nfiles-failed=1\nexit-code=1\n"
	if got != want {
		t.Errorf("exact output mismatch\nwant: %q\n got: %q", want, got)
	}
}
