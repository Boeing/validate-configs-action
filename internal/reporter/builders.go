package reporter

import (
	"strings"

	cfvreporter "github.com/Boeing/config-file-validator/v2/pkg/reporter"
)

// BuildReporters parses a comma-separated reporter string into cfv Reporter instances.
// Format: "type:dest,type:dest" where :dest is optional.
func BuildReporters(arg string) []cfvreporter.Reporter {
	if arg == "" {
		return []cfvreporter.Reporter{cfvreporter.NewStdoutReporter("")}
	}
	var reporters []cfvreporter.Reporter
	for _, r := range strings.Split(arg, ",") {
		parts := strings.SplitN(r, ":", 2)
		name := parts[0]
		dest := ""
		if len(parts) == 2 {
			dest = parts[1]
		}
		switch name {
		case "json":
			reporters = append(reporters, cfvreporter.NewJSONReporter(dest))
		case "junit":
			reporters = append(reporters, cfvreporter.NewJunitReporter(dest))
		case "sarif":
			reporters = append(reporters, cfvreporter.NewSARIFReporter(dest))
		case "github":
			reporters = append(reporters, cfvreporter.NewGitHubReporter(dest))
		default:
			reporters = append(reporters, cfvreporter.NewStdoutReporter(dest))
		}
	}
	return reporters
}
