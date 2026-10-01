package format

import (
	"github.com/Boeing/config-file-validator/v3/pkg/reporter"
)

// ComputeExitCode adjusts cfv's exit code based on the format-check mode.
//
// In "warn" mode, if the only non-passing reports are StatusUnformatted
// (no StatusFail), the exit code is overridden to 0.
//
// In "strict" and "off" modes, cfv's exit code is used as-is.
func ComputeExitCode(cfvExitCode int, reports []reporter.Report, formatCheckMode string) int {
	// Runtime error — always propagate.
	if cfvExitCode == 2 {
		return 2
	}

	// If cfv says success, it's success.
	if cfvExitCode == 0 {
		return 0
	}

	// cfv returned 1 (error found). In warn mode, if no reports have StatusFail,
	// the error was caused solely by format issues. We downgrade to 0 because
	// warn mode treats format issues as non-blocking warnings.
	//
	// This also covers edge cases where cfv returns 1 with an empty or all-pass
	// report set — we trust that if there's no StatusFail, there's no real
	// error to surface.
	if formatCheckMode == "warn" {
		for i := range reports {
			if reports[i].Status == reporter.StatusFail {
				return 1
			}
		}
		// Only format issues — downgrade to success in warn mode.
		return 0
	}

	return 1
}
