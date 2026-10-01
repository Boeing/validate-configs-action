package annotation

import (
	"fmt"
	"strings"

	"github.com/Boeing/config-file-validator/v3/pkg/reporter"
)

// StripWorkspacePrefix removes the /github/workspace/ prefix from file paths.
func StripWorkspacePrefix(path string) string {
	return strings.TrimPrefix(path, "/github/workspace/")
}

// EmitAnnotations writes GitHub Actions annotations for non-passing reports.
// formatCheckMode controls how format issues are reported:
//   - "warn": format issues emit ::warning
//   - "strict": format issues emit ::error
//   - "off": format issues are skipped entirely
func EmitAnnotations(reports []reporter.Report, formatCheckMode string) {
	for i := range reports {
		if reports[i].Status == reporter.StatusPass {
			continue
		}

		path := StripWorkspacePrefix(reports[i].FilePath)

		for _, issue := range reports[i].Issues {
			// In off mode, skip format issues entirely.
			if issue.Type == reporter.IssueTypeFormat && formatCheckMode == "off" {
				continue
			}

			level := "error"
			if issue.Type == reporter.IssueTypeFormat && formatCheckMode == "warn" {
				level = "warning"
			}

			title := classifyIssue(issue.Type)
			line := issue.Line
			if line == 0 {
				line = 1
			}

			cmd := fmt.Sprintf("::%s file=%s,title=%s,line=%d", level, path, title, line)
			if issue.Column > 0 {
				cmd += fmt.Sprintf(",col=%d", issue.Column)
			}
			cmd += "::" + EscapeAnnotation(issue.Message)
			fmt.Println(cmd)
		}
	}
}

// EmitNotes writes GitHub Actions notice annotations for report notes.
func EmitNotes(reports []reporter.Report) {
	for i := range reports {
		if len(reports[i].Notes) == 0 {
			continue
		}
		path := StripWorkspacePrefix(reports[i].FilePath)
		for _, note := range reports[i].Notes {
			fmt.Printf("::notice file=%s,title=Note::%s\n", path, EscapeAnnotation(note))
		}
	}
}

// FormatBody formats one or more messages into an annotation body.
func FormatBody(title string, msgs []string) string {
	if len(msgs) == 1 {
		return msgs[0]
	}
	lines := []string{fmt.Sprintf("%d %ss found:", len(msgs), strings.ToLower(title))}
	for _, m := range msgs {
		lines = append(lines, "\u2022 "+m)
	}
	return strings.Join(lines, "\n")
}

// classifyIssue maps a v3 IssueType to a human-readable annotation title.
func classifyIssue(t reporter.IssueType) string {
	switch t {
	case reporter.IssueTypeSyntax:
		return "Syntax Error"
	case reporter.IssueTypeSchema:
		return "Schema Error"
	case reporter.IssueTypeFormat:
		return "Formatting"
	default:
		return "Validation Error"
	}
}
