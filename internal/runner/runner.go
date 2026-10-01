package runner

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Boeing/config-file-validator/v3/pkg/cli"
	"github.com/Boeing/config-file-validator/v3/pkg/configfile"
	"github.com/Boeing/config-file-validator/v3/pkg/filetype"
	"github.com/Boeing/config-file-validator/v3/pkg/finder"
	"github.com/Boeing/config-file-validator/v3/pkg/schemastore"
	"github.com/Boeing/config-file-validator/v3/pkg/tools"

	"github.com/Boeing/validate-configs-action/internal/annotation"
	"github.com/Boeing/validate-configs-action/internal/config"
	"github.com/Boeing/validate-configs-action/internal/filter"
	intformat "github.com/Boeing/validate-configs-action/internal/format"
	"github.com/Boeing/validate-configs-action/internal/input"
	"github.com/Boeing/validate-configs-action/internal/output"
	intreporter "github.com/Boeing/validate-configs-action/internal/reporter"
	"github.com/Boeing/validate-configs-action/internal/summary"
)

// Run executes the validation pipeline and returns the exit code.
func Run(cfg *config.Config) int {
	// TODO: refactor Run() into testable stages (buildFinderOpts, buildCLIOpts, postProcess) — see PLAN.md Phase 4.5

	// Validate format-check mode.
	formatCheckMode := cfg.FormatCheck
	switch formatCheckMode {
	case "warn", "strict", "off":
		// valid
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid format-check value %q (must be warn, strict, or off)\n", formatCheckMode)
		return 2
	}

	// Validate mutually exclusive inputs.
	if cfg.FileTypes != "" && cfg.ExcludeFileTypes != "" {
		fmt.Fprintf(os.Stderr, "Error: file-types and exclude-file-types are mutually exclusive\n")
		return 2
	}
	if cfg.NoSchema && cfg.RequireSchema {
		fmt.Fprintf(os.Stderr, "Error: no-schema and require-schema are mutually exclusive\n")
		return 2
	}
	if cfg.Globbing && (cfg.ExcludeDirs != "" || cfg.ExcludeFileTypes != "" || cfg.FileTypes != "") {
		fmt.Fprintf(os.Stderr, "Error: globbing cannot be combined with exclude-dirs, exclude-file-types, or file-types\n")
		return 2
	}

	// Discover/load .cfv.toml config file.
	var cfvCfg *configfile.Config
	var cfvCfgPath string
	if !cfg.NoConfig {
		if cfg.ConfigPath != "" {
			cfvCfgPath = cfg.ConfigPath
		} else {
			cfvCfgPath = configfile.Discover(".")
		}
		if cfvCfgPath != "" {
			loaded, err := configfile.Load(cfvCfgPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config file %s: %v\n", cfvCfgPath, err)
				return 2
			}
			cfvCfg = loaded
		}
	}

	// Build finder options
	var fsOpts []finder.FSFinderOptions

	paths := []string{"."}
	if cfg.SearchPaths != "" {
		paths = strings.Fields(cfg.SearchPaths)
	}
	if cfg.Globbing {
		expanded, err := input.ExpandGlobs(os.DirFS("."), paths)
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
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid depth value %q\n", cfg.Depth)
			return 2
		}
		fsOpts = append(fsOpts, finder.WithDepth(d))
	}

	if cfg.TypeMap != "" {
		overrides, err := input.ParseTypeMap(cfg.TypeMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing type-map: %v\n", err)
			return 2
		}
		fsOpts = append(fsOpts, finder.WithTypeOverrides(overrides))
	}

	if cfg.Gitignore {
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
	if cfg.OnlyChanged {
		changed, err := filter.GetChangedFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: only-changed is enabled but could not determine changed files: %v\n", err)
			return 2
		}
		if len(changed) > 0 {
			fileFinder = &filter.ChangedFilesFilter{Inner: fileFinder, Changed: changed}
		}
	}

	cliOpts = append(cliOpts, cli.WithFinder(fileFinder))

	if cfg.Quiet {
		cliOpts = append(cliOpts, cli.WithQuiet(true))
	}
	if cfg.RequireSchema {
		cliOpts = append(cliOpts, cli.WithRequireSchema(true))
	}
	if cfg.NoSchema {
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
	} else if cfg.SchemaStore {
		store, err := schemastore.OpenEmbedded()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening embedded schemastore: %v\n", err)
			return 2
		}
		cliOpts = append(cliOpts, cli.WithSchemaStore(store))
	}

	// Wire config file path (tells cfv to skip format-checking its own config).
	if cfvCfgPath != "" {
		cliOpts = append(cliOpts, cli.WithConfigFile(cfvCfgPath))
	}

	// Wire format checking (warn/strict enable it, off skips it).
	if formatCheckMode != "off" {
		optsFunc, formatIgnores := intformat.BuildFormatOptionsFunc(cfvCfg, cfg.NoConfig)
		cliOpts = append(cliOpts, cli.WithFormatOptions(optsFunc))
		if formatIgnores != nil {
			cliOpts = append(cliOpts, cli.WithFormatIgnores(formatIgnores))
		}
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

	// Post-processing: annotations, notes, outputs, summary.
	annotation.EmitAnnotations(capture.Reports, formatCheckMode)
	annotation.EmitNotes(capture.Reports)
	output.WriteOutputs(capture.Reports, exitStatus)
	summary.WriteJobSummary(capture.Reports)

	// Override exit code based on format-check mode.
	return intformat.ComputeExitCode(exitStatus, capture.Reports, formatCheckMode)
}
