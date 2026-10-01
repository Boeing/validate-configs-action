package reporter

import (
	"fmt"
	"testing"

	cfvreporter "github.com/Boeing/config-file-validator/v3/pkg/reporter"
)

func TestCaptureReporter_Print_AccumulatesReports(t *testing.T) {
	c := &CaptureReporter{}
	reports := []cfvreporter.Report{
		{FilePath: "/tmp", FileName: "a.json", Status: cfvreporter.StatusPass},
		{FilePath: "/tmp", FileName: "b.yaml", Status: cfvreporter.StatusFail, Issues: []cfvreporter.Issue{{Type: cfvreporter.IssueTypeSyntax, Message: "bad syntax"}}},
	}

	err := c.Print(reports)
	if err != nil {
		t.Fatalf("Print returned unexpected error: %v", err)
	}
	if len(c.Reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(c.Reports))
	}
	if c.Reports[0].FileName != "a.json" {
		t.Errorf("expected first report FileName 'a.json', got %q", c.Reports[0].FileName)
	}
	if c.Reports[1].FileName != "b.yaml" {
		t.Errorf("expected second report FileName 'b.yaml', got %q", c.Reports[1].FileName)
	}
}

func TestCaptureReporter_Print_MultipleCallsAppend(t *testing.T) {
	c := &CaptureReporter{}

	batch1 := []cfvreporter.Report{
		{FileName: "first.json", Status: cfvreporter.StatusPass},
	}
	batch2 := []cfvreporter.Report{
		{FileName: "second.yaml", Status: cfvreporter.StatusFail},
		{FileName: "third.toml", Status: cfvreporter.StatusPass},
	}

	if err := c.Print(batch1); err != nil {
		t.Fatalf("first Print returned error: %v", err)
	}
	if err := c.Print(batch2); err != nil {
		t.Fatalf("second Print returned error: %v", err)
	}

	if len(c.Reports) != 3 {
		t.Fatalf("expected 3 accumulated reports, got %d", len(c.Reports))
	}
	expectedNames := []string{"first.json", "second.yaml", "third.toml"}
	for i, want := range expectedNames {
		if c.Reports[i].FileName != want {
			t.Errorf("report[%d]: expected FileName %q, got %q", i, want, c.Reports[i].FileName)
		}
	}
}

func TestCaptureReporter_Print_EmptySlice(t *testing.T) {
	c := &CaptureReporter{}

	err := c.Print([]cfvreporter.Report{})
	if err != nil {
		t.Fatalf("Print with empty slice returned error: %v", err)
	}
	if len(c.Reports) != 0 {
		t.Fatalf("expected 0 reports after empty Print, got %d", len(c.Reports))
	}
}

func TestCaptureReporter_Print_NilSlice(t *testing.T) {
	c := &CaptureReporter{}

	err := c.Print(nil)
	if err != nil {
		t.Fatalf("Print with nil slice returned error: %v", err)
	}
	if len(c.Reports) != 0 {
		t.Fatalf("expected 0 reports after nil Print, got %d", len(c.Reports))
	}
}

func TestCaptureReporter_Print_PreservesAllFields(t *testing.T) {
	c := &CaptureReporter{}
	report := cfvreporter.Report{
		FilePath: "/path/to",
		FileName: "config.json",
		Status:   cfvreporter.StatusFail,
		Issues: []cfvreporter.Issue{
			{Type: cfvreporter.IssueTypeSyntax, Message: "error1", Line: 10, Column: 5},
			{Type: cfvreporter.IssueTypeSyntax, Message: "error2", Line: 20, Column: 15},
		},
		Notes: []string{"note1"},
	}

	if err := c.Print([]cfvreporter.Report{report}); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	got := c.Reports[0]
	if got.FilePath != "/path/to" {
		t.Errorf("FilePath: got %q, want %q", got.FilePath, "/path/to")
	}
	if got.FileName != "config.json" {
		t.Errorf("FileName: got %q, want %q", got.FileName, "config.json")
	}
	if got.Status != cfvreporter.StatusFail {
		t.Errorf("Status: got %v, want %v", got.Status, cfvreporter.StatusFail)
	}
	if len(got.Issues) != 2 {
		t.Fatalf("Issues: got %d, want 2", len(got.Issues))
	}
	if got.Issues[0].Type != cfvreporter.IssueTypeSyntax {
		t.Errorf("Issues[0].Type: got %v, want %v", got.Issues[0].Type, cfvreporter.IssueTypeSyntax)
	}
	if got.Issues[0].Message != "error1" {
		t.Errorf("Issues[0].Message: got %q, want %q", got.Issues[0].Message, "error1")
	}
	if got.Issues[0].Line != 10 {
		t.Errorf("Issues[0].Line: got %d, want 10", got.Issues[0].Line)
	}
	if got.Issues[0].Column != 5 {
		t.Errorf("Issues[0].Column: got %d, want 5", got.Issues[0].Column)
	}
	if got.Issues[1].Message != "error2" {
		t.Errorf("Issues[1].Message: got %q, want %q", got.Issues[1].Message, "error2")
	}
	if got.Issues[1].Line != 20 {
		t.Errorf("Issues[1].Line: got %d, want 20", got.Issues[1].Line)
	}
	if got.Issues[1].Column != 15 {
		t.Errorf("Issues[1].Column: got %d, want 15", got.Issues[1].Column)
	}
	if len(got.Notes) != 1 {
		t.Errorf("Notes: got %d, want 1", len(got.Notes))
	}
}

func TestBuildReporters(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		wantCount int
		wantTypes []string
	}{
		{
			name:      "empty string returns single stdout reporter",
			arg:       "",
			wantCount: 1,
			wantTypes: []string{"*reporter.StdoutReporter"},
		},
		{
			name:      "standard returns stdout reporter",
			arg:       "standard",
			wantCount: 1,
			wantTypes: []string{"*reporter.StdoutReporter"},
		},
		{
			name:      "json returns JSON reporter",
			arg:       "json",
			wantCount: 1,
			wantTypes: []string{"*reporter.JSONReporter"},
		},
		{
			name:      "junit returns JUnit reporter",
			arg:       "junit",
			wantCount: 1,
			wantTypes: []string{"*reporter.JunitReporter"},
		},
		{
			name:      "sarif returns SARIF reporter",
			arg:       "sarif",
			wantCount: 1,
			wantTypes: []string{"*reporter.SARIFReporter"},
		},
		{
			name:      "github returns GitHub reporter",
			arg:       "github",
			wantCount: 1,
			wantTypes: []string{"*reporter.GitHubReporter"},
		},
		{
			name:      "unknown name falls back to stdout reporter",
			arg:       "unknown",
			wantCount: 1,
			wantTypes: []string{"*reporter.StdoutReporter"},
		},
		{
			name:      "json with dest",
			arg:       "json:output.json",
			wantCount: 1,
			wantTypes: []string{"*reporter.JSONReporter"},
		},
		{
			name:      "multiple reporters without dest",
			arg:       "json,junit,sarif",
			wantCount: 3,
			wantTypes: []string{"*reporter.JSONReporter", "*reporter.JunitReporter", "*reporter.SARIFReporter"},
		},
		{
			name:      "multiple reporters with destinations",
			arg:       "json:report.json,junit:report.xml",
			wantCount: 2,
			wantTypes: []string{"*reporter.JSONReporter", "*reporter.JunitReporter"},
		},
		{
			name:      "standard with dest",
			arg:       "standard:output.txt",
			wantCount: 1,
			wantTypes: []string{"*reporter.StdoutReporter"},
		},
		{
			name:      "mixed reporters with and without dest",
			arg:       "standard,json:report.json,github",
			wantCount: 3,
			wantTypes: []string{"*reporter.StdoutReporter", "*reporter.JSONReporter", "*reporter.GitHubReporter"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporters := BuildReporters(tt.arg)
			if len(reporters) != tt.wantCount {
				t.Fatalf("BuildReporters(%q): got %d reporters, want %d", tt.arg, len(reporters), tt.wantCount)
			}
			for i, r := range reporters {
				gotType := fmt.Sprintf("%T", r)
				if gotType != tt.wantTypes[i] {
					t.Errorf("reporter[%d]: got type %s, want %s", i, gotType, tt.wantTypes[i])
				}
			}
		})
	}
}
