package config

import "os"

// Config holds all action inputs.
type Config struct {
	SearchPaths      string
	ExcludeDirs      string
	ExcludeFileTypes string
	FileTypes        string
	Depth            string
	Reporter         string
	GroupBy          string
	Quiet            bool
	Globbing         bool
	RequireSchema    bool
	NoSchema         bool
	SchemaStore      bool
	SchemaStorePath  string
	TypeMap          string
	SchemaMap        string
	Gitignore        bool
	IgnoreFiles      string
	OnlyChanged      bool
	FormatCheck      string
	NoConfig         bool
	ConfigPath       string
}

// Load reads configuration from environment variables.
// GitHub Actions sets INPUT_<NAME> env vars from the action's inputs block.
func Load() Config {
	return Config{
		SearchPaths:      envDefault("INPUT_SEARCH_PATHS", "."),
		ExcludeDirs:      os.Getenv("INPUT_EXCLUDE_DIRS"),
		ExcludeFileTypes: os.Getenv("INPUT_EXCLUDE_FILE_TYPES"),
		FileTypes:        os.Getenv("INPUT_FILE_TYPES"),
		Depth:            os.Getenv("INPUT_DEPTH"),
		Reporter:         envDefault("INPUT_REPORTER", "standard"),
		GroupBy:          os.Getenv("INPUT_GROUP_BY"),
		Quiet:            envDefault("INPUT_QUIET", "false") == "true",
		Globbing:         envDefault("INPUT_GLOBBING", "false") == "true",
		RequireSchema:    envDefault("INPUT_REQUIRE_SCHEMA", "false") == "true",
		NoSchema:         envDefault("INPUT_NO_SCHEMA", "false") == "true",
		SchemaStore:      envDefault("INPUT_SCHEMASTORE", "true") == "true",
		SchemaStorePath:  os.Getenv("INPUT_SCHEMASTORE_PATH"),
		TypeMap:          os.Getenv("INPUT_TYPE_MAP"),
		SchemaMap:        os.Getenv("INPUT_SCHEMA_MAP"),
		Gitignore:        envDefault("INPUT_GITIGNORE", "false") == "true",
		IgnoreFiles:      os.Getenv("INPUT_IGNORE_FILES"),
		OnlyChanged:      envDefault("INPUT_ONLY_CHANGED", "false") == "true",
		FormatCheck:      envDefault("INPUT_FORMAT_CHECK", "warn"),
		NoConfig:         envDefault("INPUT_NO_CONFIG", "false") == "true",
		ConfigPath:       os.Getenv("INPUT_CONFIG"),
	}
}

// envDefault returns the value of the environment variable named by key,
// or fallback if the variable is empty or unset.
func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
