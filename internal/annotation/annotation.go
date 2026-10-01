package annotation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Boeing/config-file-validator/v2/pkg/reporter"
)

var (
	reLineNum = regexp.MustCompile(`line (\d+)`)
	reColNum  = regexp.MustCompile(`column (\d+)`)
)

type annotationGroup struct {
	file  string
	line  int
	col   int
	title string
	msgs  []string
}

// StripWorkspacePrefix removes the /github/workspace/ prefix from file paths.
func StripWorkspacePrefix(path string) string {
	return strings.TrimPrefix(path, "/github/workspace/")
}

// EmitAnnotations writes GitHub Actions error annotations for failed reports.
func EmitAnnotations(reports []reporter.Report) {
	groups := map[string]*annotationGroup{}
	var order []string

	for i := range reports {
		if reports[i].IsValid {
			continue
		}

		path := StripWorkspacePrefix(reports[i].FilePath)

		for j, errMsg := range reports[i].ValidationErrors {
			title := "Validation Error"
			msg := errMsg
			if strings.HasPrefix(errMsg, "schema: ") {
				title = "Schema Error"
				msg = errMsg[8:]
			} else if strings.HasPrefix(errMsg, "syntax: ") {
				title = "Syntax Error"
				msg = errMsg[8:]
			}

			var line, col int
			if j < len(reports[i].ErrorLines) && reports[i].ErrorLines[j] > 0 {
				line = reports[i].ErrorLines[j]
			}
			if j < len(reports[i].ErrorColumns) && reports[i].ErrorColumns[j] > 0 {
				col = reports[i].ErrorColumns[j]
			}
			if line == 0 {
				line, col = ParseLine(msg)
			}

			key := fmt.Sprintf("%s|%d|%d|%s", path, line, col, title)
			if a, ok := groups[key]; ok {
				a.msgs = append(a.msgs, msg)
			} else {
				groups[key] = &annotationGroup{
					file: path, line: line, col: col,
					title: title, msgs: []string{msg},
				}
				order = append(order, key)
			}
		}
	}

	for _, key := range order {
		a := groups[key]
		body := FormatBody(a.title, a.msgs)
		line := a.line
		if line == 0 {
			line = 1
		}
		cmd := fmt.Sprintf("::error file=%s,title=%s,line=%d", a.file, a.title, line)
		if a.col > 0 {
			cmd += fmt.Sprintf(",col=%d", a.col)
		}
		cmd += "::" + EscapeAnnotation(body)
		fmt.Println(cmd)
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

// ParseLine extracts line and column numbers from an error message using regex.
func ParseLine(msg string) (line, col int) {
	if m := reLineNum.FindStringSubmatch(msg); len(m) > 1 {
		line, _ = strconv.Atoi(m[1])
	}
	if m := reColNum.FindStringSubmatch(msg); len(m) > 1 {
		col, _ = strconv.Atoi(m[1])
	}
	return line, col
}

// FormatBody formats one or more messages into an annotation body.
func FormatBody(title string, msgs []string) string {
	if len(msgs) == 1 {
		return msgs[0]
	}
	lines := []string{fmt.Sprintf("%d %ss found:", len(msgs), strings.ToLower(title))}
	for _, m := range msgs {
		lines = append(lines, "• "+m)
	}
	return strings.Join(lines, "\n")
}
