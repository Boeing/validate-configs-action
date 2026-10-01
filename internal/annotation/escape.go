package annotation

import "strings"

// EscapeAnnotation escapes newlines for GitHub Actions workflow commands.
func EscapeAnnotation(s string) string {
	s = strings.ReplaceAll(s, "\n", "%0A")
	s = strings.ReplaceAll(s, "\r", "%0D")
	return s
}
