package reporter

import (
	"sync"

	cfvreporter "github.com/Boeing/config-file-validator/v3/pkg/reporter"
)

// CaptureReporter implements cfv's Reporter interface to accumulate reports
// for post-processing (annotations, job summary, outputs).
type CaptureReporter struct {
	mu      sync.Mutex
	Reports []cfvreporter.Report
}

// Print accumulates reports into the Reports slice.
func (c *CaptureReporter) Print(reports []cfvreporter.Report) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Reports = append(c.Reports, reports...)
	return nil
}
