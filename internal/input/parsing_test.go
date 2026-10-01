package input

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Boeing/config-file-validator/v3/pkg/cli"
	"github.com/Boeing/config-file-validator/v3/pkg/filetype"
)

// ---------- ParseTypeMap ----------

func TestParseTypeMap(t *testing.T) {
	// Build a quick lookup so we can assert on the returned FileType.
	ftByName := make(map[string]filetype.FileType)
	for _, ft := range filetype.FileTypes {
		ftByName[ft.Name] = ft
	}

	tests := []struct {
		name      string
		input     string
		wantLen   int
		wantErr   bool
		checkFunc func(t *testing.T, got []interface{})
	}{
		{
			name:    "valid single mapping",
			input:   "**/inventory:ini",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "valid multiple mappings",
			input:   "**/a:json,**/b:yaml",
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "case insensitive type",
			input:   "**/foo:JSON",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "invalid format no colon",
			input:   "nocolon",
			wantErr: true,
		},
		{
			name:    "empty pattern",
			input:   ":json",
			wantErr: true,
		},
		{
			name:    "empty type",
			input:   "pattern:",
			wantErr: true,
		},
		{
			name:    "unknown file type",
			input:   "pattern:doesnotexist",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTypeMap(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tc.wantLen)
			}
		})
	}

	// Detailed assertions for specific cases.
	t.Run("single mapping fields", func(t *testing.T) {
		got, err := ParseTypeMap("**/inventory:ini")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got[0].Pattern != "**/inventory" {
			t.Errorf("Pattern = %q, want %q", got[0].Pattern, "**/inventory")
		}
		if got[0].FileType.Name != "ini" {
			t.Errorf("FileType.Name = %q, want %q", got[0].FileType.Name, "ini")
		}
	})

	t.Run("multiple mapping fields", func(t *testing.T) {
		got, err := ParseTypeMap("**/a:json,**/b:yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got[0].Pattern != "**/a" {
			t.Errorf("[0].Pattern = %q, want %q", got[0].Pattern, "**/a")
		}
		if got[0].FileType.Name != "json" {
			t.Errorf("[0].FileType.Name = %q, want %q", got[0].FileType.Name, "json")
		}
		if got[1].Pattern != "**/b" {
			t.Errorf("[1].Pattern = %q, want %q", got[1].Pattern, "**/b")
		}
		if got[1].FileType.Name != "yaml" {
			t.Errorf("[1].FileType.Name = %q, want %q", got[1].FileType.Name, "yaml")
		}
	})

	t.Run("case insensitive resolves to correct type", func(t *testing.T) {
		got, err := ParseTypeMap("**/foo:JSON")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got[0].FileType.Name != "json" {
			t.Errorf("FileType.Name = %q, want %q", got[0].FileType.Name, "json")
		}
	})
}

// ---------- ParseSchemaMap ----------

func TestParseSchemaMap(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []cli.SchemaMapping
		wantErr bool
	}{
		{
			name:  "valid single mapping",
			input: "**/pkg.json:schemas/pkg.schema.json",
			want:  []cli.SchemaMapping{{Pattern: "**/pkg.json", SchemaPath: "schemas/pkg.schema.json"}},
		},
		{
			name:  "valid multiple mappings",
			input: "**/a.json:schemas/a.json,**/b.yaml:schemas/b.yaml",
			want: []cli.SchemaMapping{
				{Pattern: "**/a.json", SchemaPath: "schemas/a.json"},
				{Pattern: "**/b.yaml", SchemaPath: "schemas/b.yaml"},
			},
		},
		{
			name:    "invalid format no colon",
			input:   "nocolon",
			wantErr: true,
		},
		{
			name:    "empty pattern",
			input:   ":schema.json",
			wantErr: true,
		},
		{
			name:    "empty schema",
			input:   "pattern:",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSchemaMap(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tc.want))
			}
			for i, wantM := range tc.want {
				if got[i].Pattern != wantM.Pattern {
					t.Errorf("[%d].Pattern = %q, want %q", i, got[i].Pattern, wantM.Pattern)
				}
				if got[i].SchemaPath != wantM.SchemaPath {
					t.Errorf("[%d].SchemaPath = %q, want %q", i, got[i].SchemaPath, wantM.SchemaPath)
				}
			}
		})
	}
}

// ---------- ExpandGlobs ----------

func TestExpandGlobs(t *testing.T) {
	t.Run("non-glob passes through", func(t *testing.T) {
		got, err := ExpandGlobs(os.DirFS("."), []string{"plain/path", "another"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"plain/path", "another"}
		if len(got) != len(want) {
			t.Fatalf("len = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("glob expands to matching files", func(t *testing.T) {
		dir := t.TempDir()

		// Create test files.
		for _, name := range []string{"a.json", "b.json", "c.yaml"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
		}

		got, err := ExpandGlobs(os.DirFS(dir), []string{"*.json"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		sort.Strings(got)
		want := []string{"a.json", "b.json"}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("invalid glob returns error", func(t *testing.T) {
		_, err := ExpandGlobs(os.DirFS("."), []string{"[invalid"})
		if err == nil {
			t.Fatal("expected error for invalid glob, got nil")
		}
	})
}

// ---------- ExpandFileTypes ----------

func TestExpandFileTypes(t *testing.T) {
	t.Run("known type expands to all extensions", func(t *testing.T) {
		got := ExpandFileTypes([]string{"yaml"})
		gotMap := make(map[string]struct{})
		for _, s := range got {
			gotMap[s] = struct{}{}
		}
		// yaml FileType has extensions: yaml, yml
		for _, want := range []string{"yaml", "yml"} {
			if _, ok := gotMap[want]; !ok {
				t.Errorf("missing expected extension %q in result %v", want, got)
			}
		}
	})

	t.Run("unknown type passes through unchanged", func(t *testing.T) {
		got := ExpandFileTypes([]string{"unknowntype"})
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1; got %v", len(got), got)
		}
		if got[0] != "unknowntype" {
			t.Errorf("got %q, want %q", got[0], "unknowntype")
		}
	})

	t.Run("multiple types", func(t *testing.T) {
		got := ExpandFileTypes([]string{"yaml", "hcl"})
		gotMap := make(map[string]struct{})
		for _, s := range got {
			gotMap[s] = struct{}{}
		}
		// yaml → yaml, yml; hcl → hcl, tf, tfvars
		for _, want := range []string{"yaml", "yml", "hcl", "tf", "tfvars"} {
			if _, ok := gotMap[want]; !ok {
				t.Errorf("missing expected extension %q in result %v", want, got)
			}
		}
	})

	t.Run("no duplicates when input overlaps with expanded", func(t *testing.T) {
		got := ExpandFileTypes([]string{"yaml", "yml"})
		gotMap := make(map[string]struct{})
		for _, s := range got {
			gotMap[s] = struct{}{}
		}
		if len(got) != len(gotMap) {
			t.Errorf("duplicate entries in result %v", got)
		}
		for _, want := range []string{"yaml", "yml"} {
			if _, ok := gotMap[want]; !ok {
				t.Errorf("missing expected extension %q in result %v", want, got)
			}
		}
	})
}
