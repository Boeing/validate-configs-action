package runner

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Boeing/config-file-validator/v3/pkg/cli"
	"github.com/Boeing/config-file-validator/v3/pkg/filetype"
	"github.com/Boeing/config-file-validator/v3/pkg/finder"
	"github.com/Boeing/config-file-validator/v3/pkg/schemastore"
	"github.com/Boeing/config-file-validator/v3/pkg/tools"

	"github.com/Boeing/validate-configs-action/internal/annotation"
	"github.com/Boeing/validate-configs-action/internal/config"
	"github.com/Boeing/validate-configs-action/internal/filter"
	"github.com/Boeing/validate-configs-action/internal/input"
	"github.com/Boeing/validate-configs-action/internal/output"
	intreporter "github.com/Boeing/validate-configs-action/internal/reporter"
	"github.com/Boeing/validate-configs-action/internal/summary"
)

// Run executes the validation pipeline and returns the exit code.
func Run(cfg *config.Config) int {
	// Build finder options
	var fsOpts []finder.FSFinderOptions

	paths := []string{"."}
	if cfg.SearchPaths != "" {
		paths = strings.Fields(cfg.SearchPaths)
	}
	if cfg.Globbing == "true" {
		expanded, err := input.ExpandGlobs(paths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error expanding globs: %v\n", err)
			return 2
		}
		paths = expanded
	}
	fsOpts = append(fsOpts, finder.WithPathRoots(paths...))

	if cfg.ExcludeDirs != "" {
		fsOpts = append(fsOpts, finder.WithExcludeDirs(strings.Split(cfg.ExcludeDirs, ",")))
	}

	if cfg.ExcludeFileTypes != "" {
		lower := strings.ToLower(cfg.ExcludeFileTypes)
		excludeTypes := input.ExpandFileTypes(strings.Split(lower, ","))
		fsOpts = append(fsOpts, finder.WithExcludeFileTypes(excludeTypes))
	}

	if cfg.FileTypes != "" {
		includeTypes := tools.ArrToMap(strings.Split(strings.ToLower(cfg.FileTypes), ",")...)
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

	if cfg.Depth != "" {
		d, err := strconv.Atoi(cfg.Depth)
		if err == nil {
			fsOpts = append(fsOpts, finder.WithDepth(d))
		}
	}

	if cfg.TypeMap != "" {
		overrides, err := input.ParseTypeMap(cfg.TypeMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing type-map: %v\n", err)
			return 2
		}
		fsOpts = append(fsOpts, finder.WithTypeOverrides(overrides))
	}

	if cfg.Gitignore == "true" {
		fsOpts = append(fsOpts, finder.WithGitignore(true))
	}

	if cfg.IgnoreFiles != "" {
		fsOpts = append(fsOpts, finder.WithIgnoreFiles(strings.Split(cfg.IgnoreFiles, ",")))
	}

	// Build CLI options
	var cliOpts []cli.Option

	var fileFinder finder.FileFinder
	fileFinder = finder.FileSystemFinderInit(fsOpts...)

	// Filter to only changed files if requested
	if cfg.OnlyChanged == "true" {
		changed, err := filter.GetChangedFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not determine changed files: %v\n", err)
		} else if len(changed) > 0 {
			fileFinder = &filter.ChangedFilesFilter{Inner: fileFinder, Changed: changed}
		}
	}

	cliOpts = append(cliOpts, cli.WithFinder(fileFinder))

	if cfg.Quiet == "true" {
		cliOpts = append(cliOpts, cli.WithQuiet(true))
	}
	if cfg.RequireSchema == "true" {
		cliOpts = append(cliOpts, cli.WithRequireSchema(true))
	}
	if cfg.NoSchema == "true" {
		cliOpts = append(cliOpts, cli.WithNoSchema(true))
	}
	if cfg.GroupBy != "" {
		cliOpts = append(cliOpts, cli.WithGroupOutput(strings.Split(cfg.GroupBy, ",")))
	}
	if cfg.SchemaMap != "" {
		sm, err := input.ParseSchemaMap(cfg.SchemaMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing schema-map: %v\n", err)
			return 2
		}
		cliOpts = append(cliOpts, cli.WithSchemaMap(sm))
	}
	if cfg.SchemaStorePath != "" {
		store, err := schemastore.Open(cfg.SchemaStorePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening schemastore: %v\n", err)
			return 2
		}
		cliOpts = append(cliOpts, cli.WithSchemaStore(store))
	} else if cfg.SchemaStore == "true" {
		store, err := schemastore.OpenEmbedded()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening embedded schemastore: %v\n", err)
			return 2
		}
		cliOpts = append(cliOpts, cli.WithSchemaStore(store))
	}

	capture := &intreporter.CaptureReporter{}
	reporters := intreporter.BuildReporters(cfg.Reporter)
	reporters = append(reporters, capture)
	cliOpts = append(cliOpts, cli.WithReporters(reporters...))

	c := cli.Init(cliOpts...)
	exitStatus, err := c.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	annotation.EmitAnnotations(capture.Reports)
	annotation.EmitNotes(capture.Reports)
	output.WriteOutputs(capture.Reports, exitStatus)
	summary.WriteJobSummary(capture.Reports)

	return exitStatus
}
