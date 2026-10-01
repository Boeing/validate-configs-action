package annotation

import "strings"

// EscapeAnnotation escapes special characters for GitHub Actions workflow commands.
// Order matters: % must be escaped first so that the %0A and %0D replacements
// are not double-escaped.
func EscapeAnnotation(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\n", "%0A")
	s = strings.ReplaceAll(s, "\r", "%0D")
	return s
}
