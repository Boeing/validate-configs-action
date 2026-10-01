package reporter

import (
	cfvreporter "github.com/Boeing/config-file-validator/v2/pkg/reporter"
)

// CaptureReporter implements cfv's Reporter interface to accumulate reports
// for post-processing (annotations, job summary, outputs).
type CaptureReporter struct {
	Reports []cfvreporter.Report
}

// Print accumulates reports into the Reports slice.
func (c *CaptureReporter) Print(reports []cfvreporter.Report) error {
	c.Reports = append(c.Reports, reports...)
	return nil
}
