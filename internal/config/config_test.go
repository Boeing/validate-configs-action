package config

import (
	"os"
	"testing"
)

func TestLoad_AllFieldsPopulated(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{
		"cmd",                // [0]  program name
		"src docs",           // [1]  SearchPaths
		"vendor,node_modules", // [2]  ExcludeDirs
		"csv,xml",            // [3]  ExcludeFileTypes
		"json,yaml",          // [4]  FileTypes
		"5",                  // [5]  Depth
		"standard",           // [6]  Reporter
		"filetype",           // [7]  GroupBy
		"true",               // [8]  Quiet
		"false",              // [9]  Globbing
		"true",               // [10] RequireSchema
		"false",              // [11] NoSchema
		"true",               // [12] SchemaStore
		"/tmp/schemastore",   // [13] SchemaStorePath
		"**/inv:ini",         // [14] TypeMap
		"**/cfg:s.json",      // [15] SchemaMap
		"true",               // [16] Gitignore
		".dockerignore",      // [17] IgnoreFiles
		"true",               // [18] OnlyChanged
	}

	cfg := Load()

	tests := []struct {
		field string
		got   string
		want  string
	}{
		{"SearchPaths", cfg.SearchPaths, "src docs"},
		{"ExcludeDirs", cfg.ExcludeDirs, "vendor,node_modules"},
		{"ExcludeFileTypes", cfg.ExcludeFileTypes, "csv,xml"},
		{"FileTypes", cfg.FileTypes, "json,yaml"},
		{"Depth", cfg.Depth, "5"},
		{"Reporter", cfg.Reporter, "standard"},
		{"GroupBy", cfg.GroupBy, "filetype"},
		{"Quiet", cfg.Quiet, "true"},
		{"Globbing", cfg.Globbing, "false"},
		{"RequireSchema", cfg.RequireSchema, "true"},
		{"NoSchema", cfg.NoSchema, "false"},
		{"SchemaStore", cfg.SchemaStore, "true"},
		{"SchemaStorePath", cfg.SchemaStorePath, "/tmp/schemastore"},
		{"TypeMap", cfg.TypeMap, "**/inv:ini"},
		{"SchemaMap", cfg.SchemaMap, "**/cfg:s.json"},
		{"Gitignore", cfg.Gitignore, "true"},
		{"IgnoreFiles", cfg.IgnoreFiles, ".dockerignore"},
		{"OnlyChanged", cfg.OnlyChanged, "true"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.field, tt.got, tt.want)
		}
	}
}

func TestLoad_EmptyStrings(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{
		"cmd", // [0] program name
		"",    // [1]  SearchPaths
		"",    // [2]  ExcludeDirs
		"",    // [3]  ExcludeFileTypes
		"",    // [4]  FileTypes
		"",    // [5]  Depth
		"",    // [6]  Reporter
		"",    // [7]  GroupBy
		"",    // [8]  Quiet
		"",    // [9]  Globbing
		"",    // [10] RequireSchema
		"",    // [11] NoSchema
		"",    // [12] SchemaStore
		"",    // [13] SchemaStorePath
		"",    // [14] TypeMap
		"",    // [15] SchemaMap
		"",    // [16] Gitignore
		"",    // [17] IgnoreFiles
		"",    // [18] OnlyChanged
	}

	cfg := Load()

	tests := []struct {
		field string
		got   string
	}{
		{"SearchPaths", cfg.SearchPaths},
		{"ExcludeDirs", cfg.ExcludeDirs},
		{"ExcludeFileTypes", cfg.ExcludeFileTypes},
		{"FileTypes", cfg.FileTypes},
		{"Depth", cfg.Depth},
		{"Reporter", cfg.Reporter},
		{"GroupBy", cfg.GroupBy},
		{"Quiet", cfg.Quiet},
		{"Globbing", cfg.Globbing},
		{"RequireSchema", cfg.RequireSchema},
		{"NoSchema", cfg.NoSchema},
		{"SchemaStore", cfg.SchemaStore},
		{"SchemaStorePath", cfg.SchemaStorePath},
		{"TypeMap", cfg.TypeMap},
		{"SchemaMap", cfg.SchemaMap},
		{"Gitignore", cfg.Gitignore},
		{"IgnoreFiles", cfg.IgnoreFiles},
		{"OnlyChanged", cfg.OnlyChanged},
	}

	for _, tt := range tests {
		if tt.got != "" {
			t.Errorf("%s = %q, want empty string", tt.field, tt.got)
		}
	}
}
