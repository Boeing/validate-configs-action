package reporter

import (
	"fmt"
	"testing"

	cfvreporter "github.com/Boeing/config-file-validator/v2/pkg/reporter"
)

func TestCaptureReporter_Print_AccumulatesReports(t *testing.T) {
	c := &CaptureReporter{}
	reports := []cfvreporter.Report{
		{FilePath: "/tmp", FileName: "a.json", IsValid: true},
		{FilePath: "/tmp", FileName: "b.yaml", IsValid: false, ValidationErrors: []string{"bad syntax"}},
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
		{FileName: "first.json", IsValid: true},
	}
	batch2 := []cfvreporter.Report{
		{FileName: "second.yaml", IsValid: false},
		{FileName: "third.toml", IsValid: true},
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
		FilePath:         "/path/to",
		FileName:         "config.json",
		IsValid:          false,
		ValidationErrors: []string{"error1", "error2"},
		ErrorLines:       []int{10, 20},
		ErrorColumns:     []int{5, 15},
		Notes:            []string{"note1"},
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
	if got.IsValid {
		t.Error("IsValid: got true, want false")
	}
	if len(got.ValidationErrors) != 2 {
		t.Errorf("ValidationErrors: got %d, want 2", len(got.ValidationErrors))
	}
	if len(got.ErrorLines) != 2 {
		t.Errorf("ErrorLines: got %d, want 2", len(got.ErrorLines))
	}
	if len(got.ErrorColumns) != 2 {
		t.Errorf("ErrorColumns: got %d, want 2", len(got.ErrorColumns))
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
