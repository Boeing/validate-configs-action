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
	Quiet            string
	Globbing         string
	RequireSchema    string
	NoSchema         string
	SchemaStore      string
	SchemaStorePath  string
	TypeMap          string
	SchemaMap        string
	Gitignore        string
	IgnoreFiles      string
	OnlyChanged      string
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
		Quiet:            envDefault("INPUT_QUIET", "false"),
		Globbing:         envDefault("INPUT_GLOBBING", "false"),
		RequireSchema:    envDefault("INPUT_REQUIRE_SCHEMA", "false"),
		NoSchema:         envDefault("INPUT_NO_SCHEMA", "false"),
		SchemaStore:      envDefault("INPUT_SCHEMASTORE", "false"),
		SchemaStorePath:  os.Getenv("INPUT_SCHEMASTORE_PATH"),
		TypeMap:          os.Getenv("INPUT_TYPE_MAP"),
		SchemaMap:        os.Getenv("INPUT_SCHEMA_MAP"),
		Gitignore:        envDefault("INPUT_GITIGNORE", "false"),
		IgnoreFiles:      os.Getenv("INPUT_IGNORE_FILES"),
		OnlyChanged:      envDefault("INPUT_ONLY_CHANGED", "false"),
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
