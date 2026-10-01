package format

import (
	"github.com/Boeing/config-file-validator/v3/pkg/cli"
	"github.com/Boeing/config-file-validator/v3/pkg/configfile"
	"github.com/Boeing/config-file-validator/v3/pkg/formatter"
	"github.com/Boeing/config-file-validator/v3/pkg/formatter/envfmt"
	"github.com/Boeing/config-file-validator/v3/pkg/formatter/hclfmt"
	"github.com/Boeing/config-file-validator/v3/pkg/formatter/inifmt"
	"github.com/Boeing/config-file-validator/v3/pkg/formatter/jsoncfmt"
	"github.com/Boeing/config-file-validator/v3/pkg/formatter/jsonfmt"
	"github.com/Boeing/config-file-validator/v3/pkg/formatter/propfmt"
	"github.com/Boeing/config-file-validator/v3/pkg/formatter/tomlfmt"
	"github.com/Boeing/config-file-validator/v3/pkg/formatter/xmlfmt"
	"github.com/Boeing/config-file-validator/v3/pkg/formatter/yamlfmt"
)

// BuildFormatOptionsFunc builds the FormatOptionsFunc used by cfv's format
// checking pipeline. It implements a two-tier config resolution model:
//
// Tier 1 (.cfv.toml exists): .cfv.toml is sole authority. External tool
// configs (.prettierrc, taplo.toml, .yamlfmt) are NOT read.
//
// Tier 2 (no .cfv.toml): Per-format external tool config ownership.
// Resolution: defaults -> .editorconfig -> tool config (ONE per format).
//
// noConfig=true: Pure defaults only. No config files read.
func BuildFormatOptionsFunc(cfvCfg *configfile.Config, noConfig bool) (cli.FormatOptionsFunc, *formatter.FormatIgnores) {
	// --no-config: pure defaults only.
	if noConfig {
		return func(formatName, _ string) formatter.Options {
			return Defaults(formatName)
		}, nil
	}

	// Tier 1: .cfv.toml exists — sole authority.
	if cfvCfg != nil {
		globalCfg := &cfvCfg.Format.FormatOptions
		perFormatCfg := map[string]*configfile.FormatOptions{
			"json":       cfvCfg.Format.JSON,
			"jsonc":      cfvCfg.Format.JSONC,
			"yaml":       cfvCfg.Format.YAML,
			"hcl":        cfvCfg.Format.HCL,
			"toml":       cfvCfg.Format.TOML,
			"xml":        cfvCfg.Format.XML,
			"ini":        cfvCfg.Format.INI,
			"env":        cfvCfg.Format.ENV,
			"properties": cfvCfg.Format.Properties,
		}

		return func(formatName, _ string) formatter.Options {
			opts := Defaults(formatName)
			ApplyConfigFileOptions(&opts, globalCfg)
			if perFmt := perFormatCfg[formatName]; perFmt != nil {
				ApplyConfigFileOptions(&opts, perFmt)
			}
			return opts
		}, nil // no format ignores in tier 1
	}

	// Tier 2: no .cfv.toml — per-format external tool config ownership.
	editorCfg := formatter.NewEditorConfig()
	prettierCfg := formatter.NewPrettierConfig()
	taploCfg := formatter.LoadTaplo(".")
	yamlfmtCfg := formatter.LoadYamlfmt(".")

	// Load format-ignore patterns from external tool configs.
	formatIgnores := formatter.BuildFormatIgnores(".", taploCfg, yamlfmtCfg)

	optsFunc := func(formatName, path string) formatter.Options {
		opts := Defaults(formatName)

		// Base layer: .editorconfig (per-file glob matching)
		if editorCfg != nil {
			editorCfg.Apply(&opts, path)
		}

		// Tool config layer: ONE tool per format, no stacking.
		switch formatName {
		case "json", "jsonc":
			prettierCfg.Apply(&opts, path)
		case "yaml":
			if yamlfmtCfg != nil {
				yamlfmtCfg.Apply(&opts)
			} else {
				prettierCfg.Apply(&opts, path)
			}
		case "toml":
			if taploCfg != nil {
				taploCfg.Apply(&opts)
			}
		}

		return opts
	}

	return optsFunc, formatIgnores
}

// Defaults returns the canonical defaults for a format.
// Delegates to each format package's DefaultOptions() — single source of truth.
func Defaults(formatName string) formatter.Options {
	switch formatName {
	case "json":
		return jsonfmt.DefaultOptions()
	case "jsonc":
		return jsoncfmt.DefaultOptions()
	case "yaml":
		return yamlfmt.DefaultOptions()
	case "toml":
		return tomlfmt.DefaultOptions()
	case "xml":
		return xmlfmt.DefaultOptions()
	case "hcl":
		return hclfmt.DefaultOptions()
	case "ini":
		return inifmt.DefaultOptions()
	case "properties":
		return propfmt.DefaultOptions()
	case "env":
		return envfmt.DefaultOptions()
	default:
		return formatter.Options{
			IndentStyle:  formatter.IndentSpaces,
			IndentWidth:  2,
			FinalNewline: true,
			LineEnding:   formatter.LineEndingLF,
		}
	}
}

// ApplyConfigFileOptions overlays non-nil .cfv.toml values onto opts.
func ApplyConfigFileOptions(opts *formatter.Options, cfg *configfile.FormatOptions) {
	if cfg.Indent != nil {
		opts.IndentWidth = *cfg.Indent
	}
	if cfg.UseTabs != nil && *cfg.UseTabs {
		opts.IndentStyle = formatter.IndentTabs
	}
	if cfg.SortKeys != nil {
		opts.SortKeys = *cfg.SortKeys
	}
	if cfg.TrailingNewline != nil {
		opts.FinalNewline = *cfg.TrailingNewline
	}
	if cfg.LineEnding != nil {
		switch *cfg.LineEnding {
		case "crlf":
			opts.LineEnding = formatter.LineEndingCRLF
		default:
			opts.LineEnding = formatter.LineEndingLF
		}
	}
	if cfg.MaxLineWidth != nil {
		opts.MaxLineWidth = *cfg.MaxLineWidth
	}
	if cfg.QuoteStyle != nil {
		switch *cfg.QuoteStyle {
		case "double":
			opts.QuoteStyle = formatter.QuoteDouble
		case "single":
			opts.QuoteStyle = formatter.QuoteSingle
		default:
			opts.QuoteStyle = formatter.QuotePreserve
		}
	}
	if cfg.TrailingCommas != nil {
		switch *cfg.TrailingCommas {
		case "all":
			opts.TrailingCommas = formatter.TrailingCommasAll
		case "none":
			opts.TrailingCommas = formatter.TrailingCommasNone
		default:
			opts.TrailingCommas = formatter.TrailingCommasPreserve
		}
	}
	if cfg.IndentSequences != nil {
		if *cfg.IndentSequences {
			opts.IndentSequences = formatter.SequenceIndentEnabled
		} else {
			opts.IndentSequences = formatter.SequenceIndentDisabled
		}
	}
}
