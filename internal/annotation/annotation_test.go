package annotation

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Boeing/config-file-validator/v3/pkg/reporter"
)

// captureStdout redirects os.Stdout to a pipe, runs fn, and returns whatever
// was written to stdout as a string.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close() //nolint:errcheck,gosec // test helper, pipe close error is non-critical
	out, _ := io.ReadAll(r)
	os.Stdout = old
	return string(out)
}

// ---------------------------------------------------------------------------
// EscapeAnnotation
// ---------------------------------------------------------------------------

func TestEscapeAnnotation(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "newline", in: "line1\nline2", want: "line1%0Aline2"},
		{name: "carriage_return", in: "a\rb", want: "a%0Db"},
		{name: "crlf", in: "a\r\nb", want: "a%0D%0Ab"},
		{name: "clean_string", in: "no special chars", want: "no special chars"},
		{name: "empty_string", in: "", want: ""},
		{name: "multiple_newlines", in: "a\nb\nc", want: "a%0Ab%0Ac"},
		{name: "only_newline", in: "\n", want: "%0A"},
		{name: "only_cr", in: "\r", want: "%0D"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EscapeAnnotation(tt.in)
			if got != tt.want {
				t.Errorf("EscapeAnnotation(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FormatBody
// ---------------------------------------------------------------------------

func TestFormatBody(t *testing.T) {
	tests := []struct {
		name  string
		title string
		msgs  []string
		want  string
	}{
		{
			name:  "single_message",
			title: "Syntax Error",
			msgs:  []string{"unexpected '}'"},
			want:  "unexpected '}'",
		},
		{
			name:  "two_messages",
			title: "Syntax Error",
			msgs:  []string{"bad token", "missing comma"},
			want:  "2 syntax errors found:\n• bad token\n• missing comma",
		},
		{
			name:  "three_schema_errors",
			title: "Schema Error",
			msgs:  []string{"a", "b", "c"},
			want:  "3 schema errors found:\n• a\n• b\n• c",
		},
		{
			name:  "validation_error_title",
			title: "Validation Error",
			msgs:  []string{"err1", "err2"},
			want:  "2 validation errors found:\n• err1\n• err2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBody(tt.title, tt.msgs)
			if got != tt.want {
				t.Errorf("FormatBody(%q, %v) =\n  %q\nwant\n  %q",
					tt.title, tt.msgs, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// StripWorkspacePrefix
// ---------------------------------------------------------------------------

func TestStripWorkspacePrefix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "with_prefix",
			in:   "/github/workspace/src/config.json",
			want: "src/config.json",
		},
		{
			name: "without_prefix",
			in:   "src/config.json",
			want: "src/config.json",
		},
		{
			name: "exact_prefix",
			in:   "/github/workspace/",
			want: "",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "similar_but_not_prefix",
			in:   "/github/workspace-extra/foo.yaml",
			want: "/github/workspace-extra/foo.yaml",
		},
		{
			name: "double_prefix",
			in:   "/github/workspace//github/workspace/bar.json",
			want: "/github/workspace/bar.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripWorkspacePrefix(tt.in)
			if got != tt.want {
				t.Errorf("StripWorkspacePrefix(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EmitAnnotations
// ---------------------------------------------------------------------------

func TestEmitAnnotations(t *testing.T) {
	t.Run("single_syntax_error", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "/github/workspace/cfg/app.yaml",
				FileName: "app.yaml",
				Status:   reporter.StatusFail,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeSyntax, Message: "unexpected indent", Line: 3, Column: 14},
				},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		want := "::error file=cfg/app.yaml,title=Syntax Error,line=3,col=14::unexpected indent\n"
		if out != want {
			t.Errorf("EmitAnnotations() =\n  %q\nwant\n  %q", out, want)
		}
	})

	t.Run("schema_error", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "config.json",
				FileName: "config.json",
				Status:   reporter.StatusFail,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeSchema, Message: "missing required field 'name'", Line: 10, Column: 0},
				},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		// Column is 0 so no ,col= segment
		want := "::error file=config.json,title=Schema Error,line=10::missing required field 'name'\n"
		if out != want {
			t.Errorf("EmitAnnotations() =\n  %q\nwant\n  %q", out, want)
		}
	})

	t.Run("format_issue", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "style.json",
				FileName: "style.json",
				Status:   reporter.StatusUnformatted,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeFormat, Message: "incorrect indentation", Line: 5, Column: 0},
				},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		want := "::error file=style.json,title=Formatting,line=5::incorrect indentation\n"
		if out != want {
			t.Errorf("EmitAnnotations() =\n  %q\nwant\n  %q", out, want)
		}
	})

	t.Run("status_pass_skipped", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "good.json",
				FileName: "good.json",
				Status:   reporter.StatusPass,
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		if out != "" {
			t.Errorf("expected no output for passing report, got %q", out)
		}
	})

	t.Run("line_zero_defaults_to_1", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "bad.ini",
				FileName: "bad.ini",
				Status:   reporter.StatusFail,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeSyntax, Message: "unknown format", Line: 0, Column: 0},
				},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		want := "::error file=bad.ini,title=Syntax Error,line=1::unknown format\n"
		if out != want {
			t.Errorf("EmitAnnotations() =\n  %q\nwant\n  %q", out, want)
		}
	})

	t.Run("column_gt_zero_includes_col", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "x.json",
				FileName: "x.json",
				Status:   reporter.StatusFail,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeSyntax, Message: "bad token", Line: 7, Column: 12},
				},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		if !strings.Contains(out, ",col=12") {
			t.Errorf("expected col=12 in output, got:\n%s", out)
		}
	})

	t.Run("column_zero_omits_col", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "x.json",
				FileName: "x.json",
				Status:   reporter.StatusFail,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeSyntax, Message: "bad token", Line: 7, Column: 0},
				},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		if strings.Contains(out, "col=") {
			t.Errorf("expected no col= for Column=0, got:\n%s", out)
		}
	})

	t.Run("nil_reports_no_output", func(t *testing.T) {
		out := captureStdout(func() { EmitAnnotations(nil) })
		if out != "" {
			t.Errorf("expected no output for nil reports, got %q", out)
		}
	})

	t.Run("empty_reports_no_output", func(t *testing.T) {
		out := captureStdout(func() { EmitAnnotations([]reporter.Report{}) })
		if out != "" {
			t.Errorf("expected no output for empty reports, got %q", out)
		}
	})

	t.Run("multiple_issues_on_same_report", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "app.json",
				FileName: "app.json",
				Status:   reporter.StatusFail,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeSchema, Message: "type mismatch", Line: 4, Column: 12},
					{Type: reporter.IssueTypeSchema, Message: "min length", Line: 4, Column: 12},
					{Type: reporter.IssueTypeSyntax, Message: "trailing comma", Line: 10, Column: 0},
				},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		// Each issue produces its own ::error line — no coalescing in v3
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 3 {
			t.Errorf("expected 3 ::error lines, got %d:\n%s", len(lines), out)
		}
		if !strings.Contains(lines[0], "type mismatch") {
			t.Errorf("first line should contain 'type mismatch', got %q", lines[0])
		}
		if !strings.Contains(lines[1], "min length") {
			t.Errorf("second line should contain 'min length', got %q", lines[1])
		}
		if !strings.Contains(lines[2], "trailing comma") {
			t.Errorf("third line should contain 'trailing comma', got %q", lines[2])
		}
	})

	t.Run("workspace_prefix_stripped", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "/github/workspace/deep/config.yml",
				FileName: "config.yml",
				Status:   reporter.StatusFail,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeSyntax, Message: "oops", Line: 1, Column: 1},
				},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		if strings.Contains(out, "/github/workspace/") {
			t.Errorf("workspace prefix should be stripped, got:\n%s", out)
		}
		if !strings.Contains(out, "file=deep/config.yml") {
			t.Errorf("expected stripped path in output, got:\n%s", out)
		}
	})

	t.Run("message_newlines_escaped", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "f.json",
				FileName: "f.json",
				Status:   reporter.StatusFail,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeSyntax, Message: "line1\nline2", Line: 1, Column: 0},
				},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		if !strings.Contains(out, "line1%0Aline2") {
			t.Errorf("expected escaped newline in message, got:\n%s", out)
		}
		// The only raw newline should be the trailing \n from Println
		body := strings.TrimSuffix(out, "\n")
		if strings.Contains(body, "\n") {
			t.Errorf("raw newline found in annotation body:\n%s", out)
		}
	})

	t.Run("multiple_reports_mixed_statuses", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "a.json",
				FileName: "a.json",
				Status:   reporter.StatusFail,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeSyntax, Message: "bad token", Line: 1, Column: 5},
				},
			},
			{
				FilePath: "b.json",
				FileName: "b.json",
				Status:   reporter.StatusPass,
			},
			{
				FilePath: "c.yaml",
				FileName: "c.yaml",
				Status:   reporter.StatusFail,
				Issues: []reporter.Issue{
					{Type: reporter.IssueTypeSyntax, Message: "indent error", Line: 3, Column: 0},
				},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 2 {
			t.Errorf("expected 2 annotation lines, got %d:\n%s", len(lines), out)
		}
		if !strings.Contains(lines[0], "a.json") {
			t.Errorf("first annotation should be for a.json, got:\n%s", lines[0])
		}
		if !strings.Contains(lines[1], "c.yaml") {
			t.Errorf("second annotation should be for c.yaml, got:\n%s", lines[1])
		}
	})

	t.Run("fail_with_no_issues_emits_nothing", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "empty.json",
				FileName: "empty.json",
				Status:   reporter.StatusFail,
				Issues:   nil,
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		if out != "" {
			t.Errorf("expected no output for failed report with no issues, got %q", out)
		}
	})
}

// ---------------------------------------------------------------------------
// EmitNotes
// ---------------------------------------------------------------------------

func TestEmitNotes(t *testing.T) {
	t.Run("single_note", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "/github/workspace/pkg/app.json",
				FileName: "app.json",
				Status:   reporter.StatusPass,
				Notes:    []string{"schema validation skipped: no schema found"},
			},
		}
		out := captureStdout(func() { EmitNotes(reports) })

		want := "::notice file=pkg/app.json,title=Note::schema validation skipped: no schema found\n"
		if out != want {
			t.Errorf("EmitNotes() =\n  %q\nwant\n  %q", out, want)
		}
	})

	t.Run("no_notes_skipped", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "a.json",
				FileName: "a.json",
				Status:   reporter.StatusPass,
				Notes:    nil,
			},
			{
				FilePath: "b.yaml",
				FileName: "b.yaml",
				Status:   reporter.StatusFail,
				Notes:    []string{},
			},
		}
		out := captureStdout(func() { EmitNotes(reports) })

		if out != "" {
			t.Errorf("expected no output for reports without notes, got %q", out)
		}
	})

	t.Run("multiple_notes_multiple_reports", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "first.yml",
				FileName: "first.yml",
				Status:   reporter.StatusPass,
				Notes:    []string{"note A", "note B"},
			},
			{
				FilePath: "second.yml",
				FileName: "second.yml",
				Status:   reporter.StatusFail,
				Notes:    []string{"note C"},
			},
		}
		out := captureStdout(func() { EmitNotes(reports) })

		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 3 {
			t.Errorf("expected 3 notice lines, got %d:\n%s", len(lines), out)
		}
		if !strings.Contains(lines[0], "note A") {
			t.Errorf("first line should contain 'note A', got %q", lines[0])
		}
		if !strings.Contains(lines[1], "note B") {
			t.Errorf("second line should contain 'note B', got %q", lines[1])
		}
		if !strings.Contains(lines[2], "note C") {
			t.Errorf("third line should contain 'note C', got %q", lines[2])
		}
	})

	t.Run("note_with_newline_escaped", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "x.json",
				FileName: "x.json",
				Status:   reporter.StatusPass,
				Notes:    []string{"line1\nline2"},
			},
		}
		out := captureStdout(func() { EmitNotes(reports) })

		if !strings.Contains(out, "%0A") {
			t.Errorf("expected escaped newline in note, got:\n%s", out)
		}
		// Raw newline should not appear inside the annotation body (only the trailing \n from Println)
		body := strings.TrimSuffix(out, "\n")
		if strings.Contains(body, "\n") {
			t.Errorf("raw newline found in annotation body:\n%s", out)
		}
	})

	t.Run("nil_reports_no_output", func(t *testing.T) {
		out := captureStdout(func() { EmitNotes(nil) })
		if out != "" {
			t.Errorf("expected no output for nil reports, got %q", out)
		}
	})

	t.Run("workspace_prefix_stripped_in_notes", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "/github/workspace/sub/config.ini",
				FileName: "config.ini",
				Status:   reporter.StatusPass,
				Notes:    []string{"some note"},
			},
		}
		out := captureStdout(func() { EmitNotes(reports) })

		if strings.Contains(out, "/github/workspace/") {
			t.Errorf("workspace prefix should be stripped, got:\n%s", out)
		}
		if !strings.Contains(out, "file=sub/config.ini") {
			t.Errorf("expected stripped path, got:\n%s", out)
		}
	})
}
