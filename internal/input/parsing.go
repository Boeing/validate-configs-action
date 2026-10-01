package input

import (
	"fmt"
	"os"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/Boeing/config-file-validator/v3/pkg/cli"
	"github.com/Boeing/config-file-validator/v3/pkg/filetype"
	"github.com/Boeing/config-file-validator/v3/pkg/finder"
	"github.com/Boeing/config-file-validator/v3/pkg/tools"
)

// ParseTypeMap parses a comma-separated list of pattern:type mappings into TypeOverrides.
func ParseTypeMap(input string) ([]finder.TypeOverride, error) {
	fileTypesByName := make(map[string]filetype.FileType)
	for _, ft := range filetype.FileTypes {
		fileTypesByName[ft.Name] = ft
	}
	mappings := strings.Split(input, ",")
	overrides := make([]finder.TypeOverride, 0, len(mappings))
	for _, mapping := range mappings {
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

// ParseSchemaMap parses a comma-separated list of pattern:schema mappings
// into an ordered slice of cli.SchemaMapping (first match wins).
func ParseSchemaMap(input string) ([]cli.SchemaMapping, error) {
	mappings := strings.Split(input, ",")
	result := make([]cli.SchemaMapping, 0, len(mappings))
	for _, mapping := range mappings {
		parts := strings.SplitN(mapping, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid schema-map format %q", mapping)
		}
		result = append(result, cli.SchemaMapping{Pattern: parts[0], SchemaPath: parts[1]})
	}
	return result, nil
}

// ExpandGlobs expands glob patterns in the given paths using doublestar.
func ExpandGlobs(patterns []string) ([]string, error) {
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

// ExpandFileTypes expands a list of file type names to include all extensions
// associated with matching file types.
func ExpandFileTypes(types []string) []string {
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
