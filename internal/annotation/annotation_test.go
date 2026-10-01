package annotation

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Boeing/config-file-validator/v2/pkg/reporter"
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
// ParseLine
// ---------------------------------------------------------------------------

func TestParseLine(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		wantL   int
		wantC   int
	}{
		{
			name:  "line_and_column",
			msg:   "line 3, column 14: unexpected token",
			wantL: 3, wantC: 14,
		},
		{
			name:  "line_only",
			msg:   "line 7: missing closing bracket",
			wantL: 7, wantC: 0,
		},
		{
			name:  "no_line_no_column",
			msg:   "unexpected end of input",
			wantL: 0, wantC: 0,
		},
		{
			name:  "zero_line_zero_column",
			msg:   "line 0, column 0: something weird",
			wantL: 0, wantC: 0,
		},
		{
			name:  "large_numbers",
			msg:   "line 9999, column 512: way out there",
			wantL: 9999, wantC: 512,
		},
		{
			name:  "column_without_line",
			msg:   "column 5: orphan column",
			wantL: 0, wantC: 5,
		},
		{
			name:  "embedded_line_word_matches",
			msg:   "headline 4 problem",
			wantL: 4, wantC: 0, // regex `line (\d+)` matches "line" inside "headline"
		},
		{
			name:  "line_at_start",
			msg:   "line 1 something",
			wantL: 1, wantC: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotL, gotC := ParseLine(tt.msg)
			if gotL != tt.wantL || gotC != tt.wantC {
				t.Errorf("ParseLine(%q) = (%d, %d), want (%d, %d)",
					tt.msg, gotL, gotC, tt.wantL, tt.wantC)
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
	t.Run("single_syntax_error_with_error_lines", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath:         "/github/workspace/cfg/app.yaml",
				FileName:         "app.yaml",
				IsValid:          false,
				ValidationErrors: []string{"syntax: unexpected indent"},
				ErrorLines:       []int{5},
				ErrorColumns:     []int{3},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		want := "::error file=cfg/app.yaml,title=Syntax Error,line=5,col=3::unexpected indent\n"
		if out != want {
			t.Errorf("EmitAnnotations() =\n  %q\nwant\n  %q", out, want)
		}
	})

	t.Run("schema_error_prefix", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath:         "config.json",
				FileName:         "config.json",
				IsValid:          false,
				ValidationErrors: []string{"schema: missing required field 'name'"},
				ErrorLines:       []int{10},
				ErrorColumns:     []int{0},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		// col is 0 so no ,col= segment
		want := "::error file=config.json,title=Schema Error,line=10::missing required field 'name'\n"
		if out != want {
			t.Errorf("EmitAnnotations() =\n  %q\nwant\n  %q", out, want)
		}
	})

	t.Run("fallback_to_parse_line_when_error_lines_empty", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath:         "data.toml",
				FileName:         "data.toml",
				IsValid:          false,
				ValidationErrors: []string{"line 8, column 2: invalid key"},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		want := "::error file=data.toml,title=Validation Error,line=8,col=2::line 8, column 2: invalid key\n"
		if out != want {
			t.Errorf("EmitAnnotations() =\n  %q\nwant\n  %q", out, want)
		}
	})

	t.Run("valid_report_is_skipped", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath: "good.json",
				FileName: "good.json",
				IsValid:  true,
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		if out != "" {
			t.Errorf("expected no output for valid report, got %q", out)
		}
	})

	t.Run("coalesce_errors_at_same_file_line_col", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath:         "app.json",
				FileName:         "app.json",
				IsValid:          false,
				ValidationErrors: []string{"schema: type mismatch", "schema: min length"},
				ErrorLines:       []int{4, 4},
				ErrorColumns:     []int{12, 12},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		// Should be a single annotation with bullet body
		if strings.Count(out, "::error") != 1 {
			t.Errorf("expected 1 ::error annotation, got %d in:\n%s",
				strings.Count(out, "::error"), out)
		}
		if !strings.Contains(out, "2 schema errors found:") {
			t.Errorf("expected coalesced body with count header, got:\n%s", out)
		}
		if !strings.Contains(out, "type mismatch") || !strings.Contains(out, "min length") {
			t.Errorf("expected both messages in output, got:\n%s", out)
		}
	})

	t.Run("line_defaults_to_1_when_zero", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath:         "bad.ini",
				FileName:         "bad.ini",
				IsValid:          false,
				ValidationErrors: []string{"unknown format"},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		// No line info extractable => line defaults to 1
		if !strings.Contains(out, "line=1") {
			t.Errorf("expected line=1 fallback, got:\n%s", out)
		}
	})

	t.Run("empty_reports_no_output", func(t *testing.T) {
		out := captureStdout(func() { EmitAnnotations(nil) })
		if out != "" {
			t.Errorf("expected no output for nil reports, got %q", out)
		}
	})

	t.Run("multiple_reports_multiple_errors", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath:         "a.json",
				FileName:         "a.json",
				IsValid:          false,
				ValidationErrors: []string{"syntax: bad token"},
				ErrorLines:       []int{1},
				ErrorColumns:     []int{5},
			},
			{
				FilePath: "b.json",
				FileName: "b.json",
				IsValid:  true,
			},
			{
				FilePath:         "c.yaml",
				FileName:         "c.yaml",
				IsValid:          false,
				ValidationErrors: []string{"line 3: indent error"},
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

	t.Run("error_lines_zero_triggers_parse_line_fallback", func(t *testing.T) {
		// ErrorLines present but value is 0 — should fallback to ParseLine
		reports := []reporter.Report{
			{
				FilePath:         "x.json",
				FileName:         "x.json",
				IsValid:          false,
				ValidationErrors: []string{"line 15, column 4: bad value"},
				ErrorLines:       []int{0},
				ErrorColumns:     []int{0},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		if !strings.Contains(out, "line=15") {
			t.Errorf("expected ParseLine fallback to extract line 15, got:\n%s", out)
		}
		if !strings.Contains(out, "col=4") {
			t.Errorf("expected ParseLine fallback to extract col 4, got:\n%s", out)
		}
	})

	t.Run("workspace_prefix_stripped", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath:         "/github/workspace/deep/config.yml",
				FileName:         "config.yml",
				IsValid:          false,
				ValidationErrors: []string{"syntax: oops"},
				ErrorLines:       []int{1},
				ErrorColumns:     []int{1},
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

	t.Run("multiline_body_escaped", func(t *testing.T) {
		reports := []reporter.Report{
			{
				FilePath:         "f.json",
				FileName:         "f.json",
				IsValid:          false,
				ValidationErrors: []string{"schema: err1", "schema: err2"},
				ErrorLines:       []int{1, 1},
				ErrorColumns:     []int{1, 1},
			},
		}
		out := captureStdout(func() { EmitAnnotations(reports) })

		// The coalesced body has newlines that must be escaped as %0A
		if strings.Contains(out, "• ") && !strings.Contains(out, "%0A") {
			t.Errorf("expected newlines in body to be escaped, got:\n%s", out)
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
				Notes:    nil,
			},
			{
				FilePath: "b.yaml",
				FileName: "b.yaml",
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
				Notes:    []string{"note A", "note B"},
			},
			{
				FilePath: "second.yml",
				FileName: "second.yml",
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
