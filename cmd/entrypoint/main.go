package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	schemaStoreEnabled := os.Args[12]
	schemaStorePath := os.Args[13]
	typeMap := os.Args[14]
	schemaMap := os.Args[15]
	gitignoreEnabled := os.Args[16]
	onlyChanged := os.Args[17]

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
			return 2
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
			return 2
		}
		fsOpts = append(fsOpts, finder.WithTypeOverrides(overrides))
	}

	if gitignoreEnabled == "true" {
		fsOpts = append(fsOpts, finder.WithGitignore(true))
	}

	// Build CLI options
	var cliOpts []cli.Option

	var fileFinder finder.FileFinder
	fileFinder = finder.FileSystemFinderInit(fsOpts...)

	// Filter to only changed files if requested
	if onlyChanged == "true" {
		changed, err := getChangedFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not determine changed files: %v\n", err)
		} else if len(changed) > 0 {
			fileFinder = &changedFilesFilter{inner: fileFinder, changed: changed}
		}
	}

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
			return 2
		}
		cliOpts = append(cliOpts, cli.WithSchemaMap(sm))
	}
	if schemaStorePath != "" {
		store, err := schemastore.Open(schemaStorePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening schemastore: %v\n", err)
			return 2
		}
		cliOpts = append(cliOpts, cli.WithSchemaStore(store))
	} else if schemaStoreEnabled == "true" {
		store, err := schemastore.OpenEmbedded()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening embedded schemastore: %v\n", err)
			return 2
		}
		cliOpts = append(cliOpts, cli.WithSchemaStore(store))
	}

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
	emitNotes(capture.reports)
	writeOutputs(capture.reports, exitStatus)
	writeJobSummary(capture.reports)

	return exitStatus
}

// changedFilesFilter wraps a FileFinder and filters results to only changed files.
type changedFilesFilter struct {
	inner   finder.FileFinder
	changed map[string]struct{}
}

func (f *changedFilesFilter) Find() ([]finder.FileMetadata, error) {
	all, err := f.inner.Find()
	if err != nil {
		return nil, err
	}
	var filtered []finder.FileMetadata
	for _, file := range all {
		rel := file.Path
		if abs, err := filepath.Abs(file.Path); err == nil {
			if wd, err := os.Getwd(); err == nil {
				if r, err := filepath.Rel(wd, abs); err == nil {
					rel = r
				}
			}
		}
		if _, ok := f.changed[rel]; ok {
			filtered = append(filtered, file)
		}
	}
	return filtered, nil
}

func getChangedFiles() (map[string]struct{}, error) {
	baseBranch := os.Getenv("GITHUB_BASE_REF")
	if baseBranch == "" {
		return nil, fmt.Errorf("GITHUB_BASE_REF not set (not a pull request?)")
	}

	// Docker containers run as root but the workspace is owned by the runner user
	safe := exec.Command("git", "config", "--global", "--add", "safe.directory", "/github/workspace")
	safe.Run()

	fetch := exec.Command("git", "fetch", "origin", baseBranch, "--depth=1")
	fetch.Stderr = os.Stderr
	if err := fetch.Run(); err != nil {
		return nil, fmt.Errorf("git fetch: %w", err)
	}

	cmd := exec.Command("git", "diff", "--name-only", "origin/"+baseBranch+"...HEAD")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	changed := make(map[string]struct{})
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			changed[line] = struct{}{}
		}
	}
	return changed, nil
}

func writeOutputs(reports []reporter.Report, exitCode int) {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		return
	}

	total := len(reports)
	failed := 0
	for _, r := range reports {
		if !r.IsValid {
			failed++
		}
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "files-validated=%d\n", total)
	fmt.Fprintf(f, "files-failed=%d\n", failed)
	fmt.Fprintf(f, "exit-code=%d\n", exitCode)
}

func writeJobSummary(reports []reporter.Report) {
	summaryFile := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryFile == "" {
		return
	}

	total := len(reports)
	passed := 0
	failed := 0
	var failedReports []reporter.Report
	for _, r := range reports {
		if r.IsValid {
			passed++
		} else {
			failed++
			failedReports = append(failedReports, r)
		}
	}

	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	if failed == 0 {
		fmt.Fprintf(f, "### ✅ Config Validation Passed\n\n")
		fmt.Fprintf(f, "All **%d** configuration files are valid.\n", total)
		return
	}

	fmt.Fprintf(f, "### ❌ Config Validation Failed\n\n")
	fmt.Fprintf(f, "| | Count |\n|---|---|\n")
	fmt.Fprintf(f, "| ✅ Passed | %d |\n", passed)
	fmt.Fprintf(f, "| ❌ Failed | %d |\n", failed)
	fmt.Fprintf(f, "| **Total** | **%d** |\n\n", total)

	fmt.Fprintf(f, "#### Failed Files\n\n")
	fmt.Fprintf(f, "| File | Errors |\n|---|---|\n")
	for _, r := range failedReports {
		path := r.FilePath
		if strings.HasPrefix(path, "/github/workspace/") {
			path = path[len("/github/workspace/"):]
		}
		errors := strings.Join(r.ValidationErrors, "<br>")
		fmt.Fprintf(f, "| `%s` | %s |\n", path, errors)
	}
}

func emitNotes(reports []reporter.Report) {
	const workspacePrefix = "/github/workspace/"
	for _, r := range reports {
		if len(r.Notes) == 0 {
			continue
		}
		path := r.FilePath
		if strings.HasPrefix(path, workspacePrefix) {
			path = path[len(workspacePrefix):]
		}
		for _, note := range r.Notes {
			fmt.Printf("::notice file=%s,title=Note::%s\n", path, escapeAnnotation(note))
		}
	}
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

		for i, errMsg := range r.ValidationErrors {
			title := "Validation Error"
			msg := errMsg
			if strings.HasPrefix(errMsg, "schema: ") {
				title = "Schema Error"
				msg = errMsg[8:]
			} else if strings.HasPrefix(errMsg, "syntax: ") {
				title = "Syntax Error"
				msg = errMsg[8:]
			}

			// Use per-error positions from report when available,
			// fall back to regex parsing for compatibility.
			var line, col int
			if i < len(r.ErrorLines) && r.ErrorLines[i] > 0 {
				line = r.ErrorLines[i]
			}
			if i < len(r.ErrorColumns) && r.ErrorColumns[i] > 0 {
				col = r.ErrorColumns[i]
			}
			if line == 0 {
				line, col = parseLine(msg)
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
