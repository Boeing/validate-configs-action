package format

import (
	"testing"

	"github.com/Boeing/config-file-validator/v3/pkg/reporter"
)

func TestComputeExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfvExitCode int
		reports     []reporter.Report
		mode        string
		expected    int
	}{
		{
			name:        "runtime error always propagates",
			cfvExitCode: 2,
			reports:     []reporter.Report{{Status: reporter.StatusFail}},
			mode:        "warn",
			expected:    2,
		},
		{
			name:        "success always propagates",
			cfvExitCode: 0,
			reports:     []reporter.Report{{Status: reporter.StatusPass}},
			mode:        "strict",
			expected:    0,
		},
		{
			name:        "real errors in warn mode",
			cfvExitCode: 1,
			reports:     []reporter.Report{{Status: reporter.StatusFail}},
			mode:        "warn",
			expected:    1,
		},
		{
			name:        "only format issues in warn mode downgrade to success",
			cfvExitCode: 1,
			reports:     []reporter.Report{{Status: reporter.StatusUnformatted}},
			mode:        "warn",
			expected:    0,
		},
		{
			name:        "mixed: real error exists in warn mode",
			cfvExitCode: 1,
			reports: []reporter.Report{
				{Status: reporter.StatusFail},
				{Status: reporter.StatusUnformatted},
			},
			mode:     "warn",
			expected: 1,
		},
		{
			name:        "strict mode: unformatted is not overridden",
			cfvExitCode: 1,
			reports:     []reporter.Report{{Status: reporter.StatusUnformatted}},
			mode:        "strict",
			expected:    1,
		},
		{
			name:        "strict mode: fail is not overridden",
			cfvExitCode: 1,
			reports:     []reporter.Report{{Status: reporter.StatusFail}},
			mode:        "strict",
			expected:    1,
		},
		{
			name:        "all pass but cfv returned 1 in warn mode downgrade",
			cfvExitCode: 1,
			reports:     []reporter.Report{{Status: reporter.StatusPass}},
			mode:        "warn",
			expected:    0,
		},
		{
			name:        "empty reports with cfv exit 1 in warn mode downgrade",
			cfvExitCode: 1,
			reports:     []reporter.Report{},
			mode:        "warn",
			expected:    0,
		},
		{
			name:        "off mode: unformatted is not overridden",
			cfvExitCode: 1,
			reports:     []reporter.Report{{Status: reporter.StatusUnformatted}},
			mode:        "off",
			expected:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ComputeExitCode(tt.cfvExitCode, tt.reports, tt.mode)
			if got != tt.expected {
				t.Errorf("ComputeExitCode(%d, reports, %q) = %d, want %d",
					tt.cfvExitCode, tt.mode, got, tt.expected)
			}
		})
	}
}
