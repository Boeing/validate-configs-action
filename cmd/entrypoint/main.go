package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/Boeing/config-file-validator/v2/pkg/cli"
	"github.com/Boeing/config-file-validator/v2/pkg/filetype"
	"github.com/Boeing/config-file-validator/v2/pkg/finder"
	"github.com/Boeing/config-file-validator/v2/pkg/reporter"
	"github.com/Boeing/config-file-validator/v2/pkg/schemastore"
	"github.com/Boeing/config-file-validator/v2/pkg/tools"
)

var (
	reLineNum = regexp.MustCompile(`line (\d+)`)
	reColNum  = regexp.MustCompile(`column (\d+)`)
)

// captureReporter captures reports for annotation processing
type captureReporter struct {
	reports []reporter.Report
}

func (c *captureReporter) Print(reports []reporter.Report) error {
	c.reports = append(c.reports, reports...)
	return nil
}

func main() {
	os.Exit(run())
}

func run() int {
	searchPaths := os.Args[1]
	excludeDirs := os.Args[2]
	excludeFileTypes := os.Args[3]
	fileTypes := os.Args[4]
	depth := os.Args[5]
	reporterArg := os.Args[6]
	groupBy := os.Args[7]
	quiet := os.Args[8]
	globbing := os.Args[9]
	requireSchema := os.Args[10]
	noSchema := os.Args[11]
	schemaStorePath := os.Args[12]
	typeMap := os.Args[13]
	schemaMap := os.Args[14]

	// Build finder options
	var fsOpts []finder.FSFinderOptions

	paths := []string{"."}
	if searchPaths != "" {
		paths = strings.Fields(searchPaths)
	}
	if globbing == "true" {
		expanded, err := expandGlobs(paths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error expanding globs: %v\n", err)
			return 1
		}
		paths = expanded
	}
	fsOpts = append(fsOpts, finder.WithPathRoots(paths...))

	if excludeDirs != "" {
		fsOpts = append(fsOpts, finder.WithExcludeDirs(strings.Split(excludeDirs, ",")))
	}

	if excludeFileTypes != "" {
		lower := strings.ToLower(excludeFileTypes)
		excludeTypes := expandFileTypes(strings.Split(lower, ","))
		fsOpts = append(fsOpts, finder.WithExcludeFileTypes(excludeTypes))
	}

	if fileTypes != "" {
		includeTypes := tools.ArrToMap(strings.Split(strings.ToLower(fileTypes), ",")...)
		var fileTypeFilter []filetype.FileType
		for _, ft := range filetype.FileTypes {
			for ext := range ft.Extensions {
				if _, ok := includeTypes[ext]; ok {
					fileTypeFilter = append(fileTypeFilter, ft)
					break
				}
			}
		}
		fsOpts = append(fsOpts, finder.WithFileTypes(fileTypeFilter))
	}

	if depth != "" {
		d, err := strconv.Atoi(depth)
		if err == nil {
			fsOpts = append(fsOpts, finder.WithDepth(d))
		}
	}

	if typeMap != "" {
		overrides, err := parseTypeMap(typeMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing type-map: %v\n", err)
			return 1
		}
		fsOpts = append(fsOpts, finder.WithTypeOverrides(overrides))
	}

	// Build CLI options
	var cliOpts []cli.Option

	fileFinder := finder.FileSystemFinderInit(fsOpts...)
	cliOpts = append(cliOpts, cli.WithFinder(fileFinder))

	if quiet == "true" {
		cliOpts = append(cliOpts, cli.WithQuiet(true))
	}
	if requireSchema == "true" {
		cliOpts = append(cliOpts, cli.WithRequireSchema(true))
	}
	if noSchema == "true" {
		cliOpts = append(cliOpts, cli.WithNoSchema(true))
	}
	if groupBy != "" {
		cliOpts = append(cliOpts, cli.WithGroupOutput(strings.Split(groupBy, ",")))
	}
	if schemaMap != "" {
		sm, err := parseSchemaMap(schemaMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing schema-map: %v\n", err)
			return 1
		}
		cliOpts = append(cliOpts, cli.WithSchemaMap(sm))
	}
	if schemaStorePath != "" {
		store, err := schemastore.Open(schemaStorePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening schemastore: %v\n", err)
			return 1
		}
		cliOpts = append(cliOpts, cli.WithSchemaStore(store))
	}

	// Build reporters — user-specified plus a capture reporter for annotations
	capture := &captureReporter{}
	reporters := buildReporters(reporterArg)
	reporters = append(reporters, capture)
	cliOpts = append(cliOpts, cli.WithReporters(reporters...))

	c := cli.Init(cliOpts...)
	exitStatus, err := c.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	emitAnnotations(capture.reports)

	return exitStatus
}

func buildReporters(arg string) []reporter.Reporter {
	if arg == "" {
		return []reporter.Reporter{reporter.NewStdoutReporter("")}
	}
	var reporters []reporter.Reporter
	for _, r := range strings.Split(arg, ",") {
		parts := strings.SplitN(r, ":", 2)
		name := parts[0]
		dest := ""
		if len(parts) == 2 {
			dest = parts[1]
		}
		switch name {
		case "json":
			reporters = append(reporters, reporter.NewJSONReporter(dest))
		case "junit":
			reporters = append(reporters, reporter.NewJunitReporter(dest))
		case "sarif":
			reporters = append(reporters, reporter.NewSARIFReporter(dest))
		default:
			reporters = append(reporters, reporter.NewStdoutReporter(dest))
		}
	}
	return reporters
}

func parseTypeMap(input string) ([]finder.TypeOverride, error) {
	fileTypesByName := make(map[string]filetype.FileType)
	for _, ft := range filetype.FileTypes {
		fileTypesByName[ft.Name] = ft
	}
	var overrides []finder.TypeOverride
	for _, mapping := range strings.Split(input, ",") {
		parts := strings.SplitN(mapping, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid type-map format %q", mapping)
		}
		ft, ok := fileTypesByName[strings.ToLower(parts[1])]
		if !ok {
			return nil, fmt.Errorf("unknown file type %q", parts[1])
		}
		overrides = append(overrides, finder.TypeOverride{Pattern: parts[0], FileType: ft})
	}
	return overrides, nil
}

func parseSchemaMap(input string) (map[string]string, error) {
	result := make(map[string]string)
	for _, mapping := range strings.Split(input, ",") {
		parts := strings.SplitN(mapping, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid schema-map format %q", mapping)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
}

func expandGlobs(patterns []string) ([]string, error) {
	var result []string
	for _, p := range patterns {
		if strings.ContainsAny(p, "*?[]") {
			matches, err := doublestar.Glob(os.DirFS("."), p)
			if err != nil {
				return nil, fmt.Errorf("glob error for %q: %w", p, err)
			}
			result = append(result, matches...)
		} else {
			result = append(result, p)
		}
	}
	return result, nil
}

func expandFileTypes(types []string) []string {
	unique := tools.ArrToMap(types...)
	for _, ft := range filetype.FileTypes {
		for ext := range ft.Extensions {
			if _, ok := unique[ext]; !ok {
				continue
			}
			for ext := range ft.Extensions {
				unique[ext] = struct{}{}
			}
			break
		}
	}
	result := make([]string, 0, len(unique))
	for k := range unique {
		result = append(result, k)
	}
	return result
}

type annotationGroup struct {
	file  string
	line  int
	col   int
	title string
	msgs  []string
}

func emitAnnotations(reports []reporter.Report) {
	const workspacePrefix = "/github/workspace/"
	groups := map[string]*annotationGroup{}
	var order []string

	for _, r := range reports {
		if r.IsValid {
			continue
		}

		path := r.FilePath
		if strings.HasPrefix(path, workspacePrefix) {
			path = path[len(workspacePrefix):]
		}

		for _, errMsg := range r.ValidationErrors {
			title := "Validation Error"
			msg := errMsg
			if strings.HasPrefix(errMsg, "schema: ") {
				title = "Schema Error"
				msg = errMsg[8:]
			} else if strings.HasPrefix(errMsg, "syntax: ") {
				title = "Syntax Error"
				msg = errMsg[8:]
			}

			line, col := parseLine(msg)
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
		body := formatBody(a.title, a.msgs)
		line := a.line
		if line == 0 {
			line = 1
		}
		cmd := fmt.Sprintf("::error file=%s,title=%s,line=%d", a.file, a.title, line)
		if a.col > 0 {
			cmd += fmt.Sprintf(",col=%d", a.col)
		}
		cmd += "::" + escapeAnnotation(body)
		fmt.Println(cmd)
	}
}

func parseLine(msg string) (int, int) {
	var line, col int
	if m := reLineNum.FindStringSubmatch(msg); len(m) > 1 {
		line, _ = strconv.Atoi(m[1])
	}
	if m := reColNum.FindStringSubmatch(msg); len(m) > 1 {
		col, _ = strconv.Atoi(m[1])
	}
	return line, col
}

func formatBody(title string, msgs []string) string {
	if len(msgs) == 1 {
		return msgs[0]
	}
	lines := []string{fmt.Sprintf("%d %ss found:", len(msgs), strings.ToLower(title))}
	for _, m := range msgs {
		lines = append(lines, "• "+m)
	}
	return strings.Join(lines, "\n")
}

func escapeAnnotation(s string) string {
	s = strings.ReplaceAll(s, "\n", "%0A")
	s = strings.ReplaceAll(s, "\r", "%0D")
	return s
}
