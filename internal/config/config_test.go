package config

import (
	"testing"
)

func TestLoad_AllFieldsFromEnv(t *testing.T) {
	// Set all env vars.
	t.Setenv("INPUT_SEARCH_PATHS", "src/ lib/")
	t.Setenv("INPUT_EXCLUDE_DIRS", "vendor,node_modules")
	t.Setenv("INPUT_EXCLUDE_FILE_TYPES", "csv,xml")
	t.Setenv("INPUT_FILE_TYPES", "json,yaml")
	t.Setenv("INPUT_DEPTH", "3")
	t.Setenv("INPUT_REPORTER", "json")
	t.Setenv("INPUT_GROUP_BY", "filetype")
	t.Setenv("INPUT_QUIET", "true")
	t.Setenv("INPUT_GLOBBING", "true")
	t.Setenv("INPUT_REQUIRE_SCHEMA", "true")
	t.Setenv("INPUT_NO_SCHEMA", "true")
	t.Setenv("INPUT_SCHEMASTORE", "true")
	t.Setenv("INPUT_SCHEMASTORE_PATH", "/path/to/store")
	t.Setenv("INPUT_TYPE_MAP", "**/inv:ini")
	t.Setenv("INPUT_SCHEMA_MAP", "**/pkg.json:schemas/pkg.json")
	t.Setenv("INPUT_GITIGNORE", "true")
	t.Setenv("INPUT_IGNORE_FILES", ".dockerignore")
	t.Setenv("INPUT_ONLY_CHANGED", "true")
	t.Setenv("INPUT_FORMAT_CHECK", "error")
	t.Setenv("INPUT_NO_CONFIG", "true")
	t.Setenv("INPUT_CONFIG", "/path/to/config.yaml")

	cfg := Load()

	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"SearchPaths", cfg.SearchPaths, "src/ lib/"},
		{"ExcludeDirs", cfg.ExcludeDirs, "vendor,node_modules"},
		{"ExcludeFileTypes", cfg.ExcludeFileTypes, "csv,xml"},
		{"FileTypes", cfg.FileTypes, "json,yaml"},
		{"Depth", cfg.Depth, "3"},
		{"Reporter", cfg.Reporter, "json"},
		{"GroupBy", cfg.GroupBy, "filetype"},
		{"Quiet", cfg.Quiet, "true"},
		{"Globbing", cfg.Globbing, "true"},
		{"RequireSchema", cfg.RequireSchema, "true"},
		{"NoSchema", cfg.NoSchema, "true"},
		{"SchemaStore", cfg.SchemaStore, "true"},
		{"SchemaStorePath", cfg.SchemaStorePath, "/path/to/store"},
		{"TypeMap", cfg.TypeMap, "**/inv:ini"},
		{"SchemaMap", cfg.SchemaMap, "**/pkg.json:schemas/pkg.json"},
		{"Gitignore", cfg.Gitignore, "true"},
		{"IgnoreFiles", cfg.IgnoreFiles, ".dockerignore"},
		{"OnlyChanged", cfg.OnlyChanged, "true"},
		{"FormatCheck", cfg.FormatCheck, "error"},
		{"NoConfig", cfg.NoConfig, "true"},
		{"ConfigPath", cfg.ConfigPath, "/path/to/config.yaml"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Clear all env vars — t.Setenv("", "") would set them to empty which
	// triggers the fallback in envDefault. But env vars may be inherited from
	// the parent process, so explicitly set them to empty.
	for _, key := range []string{
		"INPUT_SEARCH_PATHS", "INPUT_EXCLUDE_DIRS", "INPUT_EXCLUDE_FILE_TYPES",
		"INPUT_FILE_TYPES", "INPUT_DEPTH", "INPUT_REPORTER", "INPUT_GROUP_BY",
		"INPUT_QUIET", "INPUT_GLOBBING", "INPUT_REQUIRE_SCHEMA", "INPUT_NO_SCHEMA",
		"INPUT_SCHEMASTORE", "INPUT_SCHEMASTORE_PATH", "INPUT_TYPE_MAP",
		"INPUT_SCHEMA_MAP", "INPUT_GITIGNORE", "INPUT_IGNORE_FILES", "INPUT_ONLY_CHANGED",
		"INPUT_FORMAT_CHECK", "INPUT_NO_CONFIG", "INPUT_CONFIG",
	} {
		t.Setenv(key, "")
	}

	cfg := Load()

	// Fields with defaults via envDefault.
	defaults := []struct {
		field string
		got   string
		want  string
	}{
		{"SearchPaths", cfg.SearchPaths, "."},
		{"Reporter", cfg.Reporter, "standard"},
		{"Quiet", cfg.Quiet, "false"},
		{"Globbing", cfg.Globbing, "false"},
		{"RequireSchema", cfg.RequireSchema, "false"},
		{"NoSchema", cfg.NoSchema, "false"},
		{"SchemaStore", cfg.SchemaStore, "true"},
		{"Gitignore", cfg.Gitignore, "false"},
		{"OnlyChanged", cfg.OnlyChanged, "false"},
		{"FormatCheck", cfg.FormatCheck, "warn"},
		{"NoConfig", cfg.NoConfig, "false"},
	}
	for _, c := range defaults {
		if c.got != c.want {
			t.Errorf("%s = %q, want default %q", c.field, c.got, c.want)
		}
	}

	// Fields without defaults — should be empty.
	empties := []struct {
		field string
		got   string
	}{
		{"ExcludeDirs", cfg.ExcludeDirs},
		{"ExcludeFileTypes", cfg.ExcludeFileTypes},
		{"FileTypes", cfg.FileTypes},
		{"Depth", cfg.Depth},
		{"GroupBy", cfg.GroupBy},
		{"SchemaStorePath", cfg.SchemaStorePath},
		{"TypeMap", cfg.TypeMap},
		{"SchemaMap", cfg.SchemaMap},
		{"IgnoreFiles", cfg.IgnoreFiles},
		{"ConfigPath", cfg.ConfigPath},
	}
	for _, c := range empties {
		if c.got != "" {
			t.Errorf("%s = %q, want empty string", c.field, c.got)
		}
	}
}

func TestEnvDefault(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		t.Setenv("TEST_ENV_DEFAULT_KEY", "custom-value")
		got := envDefault("TEST_ENV_DEFAULT_KEY", "fallback")
		if got != "custom-value" {
			t.Errorf("envDefault = %q, want %q", got, "custom-value")
		}
	})

	t.Run("returns fallback when empty", func(t *testing.T) {
		t.Setenv("TEST_ENV_DEFAULT_KEY", "")
		got := envDefault("TEST_ENV_DEFAULT_KEY", "fallback")
		if got != "fallback" {
			t.Errorf("envDefault = %q, want %q", got, "fallback")
		}
	})

	t.Run("returns fallback when unset", func(t *testing.T) {
		// Don't set the env var at all — rely on it not existing.
		got := envDefault("TEST_ENV_DEFAULT_NONEXISTENT_KEY_12345", "fallback")
		if got != "fallback" {
			t.Errorf("envDefault = %q, want %q", got, "fallback")
		}
	})
}
