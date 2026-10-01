package filter

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Boeing/config-file-validator/v2/pkg/finder"
)

// mockFinder implements finder.FileFinder for testing.
type mockFinder struct {
	files []finder.FileMetadata
	err   error
}

func (m *mockFinder) Find() ([]finder.FileMetadata, error) {
	return m.files, m.err
}

// relPath returns an absolute-then-relative path rooted at cwd so that
// ChangedFilesFilter's filepath.Abs → filepath.Rel normalisation produces
// a predictable key that we can put into the Changed map.
func relPath(t *testing.T, elem ...string) string {
	t.Helper()
	p := filepath.Join(elem...)
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", p, err)
	}
	rel, err := filepath.Rel(mustGetwd(t), abs)
	if err != nil {
		t.Fatalf("filepath.Rel: %v", err)
	}
	return rel
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("filepath.Abs(.): %v", err)
	}
	return wd
}

func TestChangedFilesFilterFind_FiltersToOnlyChanged(t *testing.T) {
	// Inner returns 3 files, changed set has 1 → result has 1.
	pathA := relPath(t, "a.json")
	pathB := relPath(t, "b.yaml")
	pathC := relPath(t, "c.toml")

	inner := &mockFinder{
		files: []finder.FileMetadata{
			{Name: "a.json", Path: "a.json"},
			{Name: "b.yaml", Path: "b.yaml"},
			{Name: "c.toml", Path: "c.toml"},
		},
	}

	f := &ChangedFilesFilter{
		Inner: inner,
		Changed: map[string]struct{}{
			pathB: {},
		},
	}

	got, err := f.Find()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 file, got %d", len(got))
	}
	if got[0].Name != "b.yaml" {
		t.Errorf("expected b.yaml, got %s", got[0].Name)
	}

	// Verify the other files were indeed excluded.
	_ = pathA
	_ = pathC
}

func TestChangedFilesFilterFind_NoMatchingFiles(t *testing.T) {
	// Inner returns files, changed set has none of them → empty result.
	inner := &mockFinder{
		files: []finder.FileMetadata{
			{Name: "a.json", Path: "a.json"},
			{Name: "b.yaml", Path: "b.yaml"},
		},
	}

	f := &ChangedFilesFilter{
		Inner: inner,
		Changed: map[string]struct{}{
			relPath(t, "x.json"): {},
			relPath(t, "y.yaml"): {},
		},
	}

	got, err := f.Find()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 files, got %d", len(got))
	}
}

func TestChangedFilesFilterFind_AllMatching(t *testing.T) {
	// Inner returns files, changed set has all → result is all.
	pathA := relPath(t, "a.json")
	pathB := relPath(t, "b.yaml")
	pathC := relPath(t, "c.toml")

	inner := &mockFinder{
		files: []finder.FileMetadata{
			{Name: "a.json", Path: "a.json"},
			{Name: "b.yaml", Path: "b.yaml"},
			{Name: "c.toml", Path: "c.toml"},
		},
	}

	f := &ChangedFilesFilter{
		Inner: inner,
		Changed: map[string]struct{}{
			pathA: {},
			pathB: {},
			pathC: {},
		},
	}

	got, err := f.Find()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 files, got %d", len(got))
	}

	names := make(map[string]bool)
	for _, file := range got {
		names[file.Name] = true
	}
	for _, want := range []string{"a.json", "b.yaml", "c.toml"} {
		if !names[want] {
			t.Errorf("missing expected file %s in results", want)
		}
	}
}

func TestChangedFilesFilterFind_InnerErrorPropagates(t *testing.T) {
	// Inner returns an error → error propagates, nil result.
	innerErr := errors.New("disk on fire")
	inner := &mockFinder{
		err: innerErr,
	}

	f := &ChangedFilesFilter{
		Inner:   inner,
		Changed: map[string]struct{}{"a.json": {}},
	}

	got, err := f.Find()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, innerErr) {
		t.Errorf("expected inner error, got: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil result on error, got %d files", len(got))
	}
}

func TestChangedFilesFilterFind_EmptyChangedSet(t *testing.T) {
	// Changed set is empty → returns nothing.
	inner := &mockFinder{
		files: []finder.FileMetadata{
			{Name: "a.json", Path: "a.json"},
			{Name: "b.yaml", Path: "b.yaml"},
		},
	}

	f := &ChangedFilesFilter{
		Inner:   inner,
		Changed: map[string]struct{}{},
	}

	got, err := f.Find()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 files, got %d", len(got))
	}
}

func TestChangedFilesFilterFind_ChangedSetHasExtraFiles(t *testing.T) {
	// Changed set has files not in inner results → no error, returns intersection.
	pathA := relPath(t, "a.json")

	inner := &mockFinder{
		files: []finder.FileMetadata{
			{Name: "a.json", Path: "a.json"},
		},
	}

	f := &ChangedFilesFilter{
		Inner: inner,
		Changed: map[string]struct{}{
			pathA:                          {},
			relPath(t, "deleted.json"):     {},
			relPath(t, "not-found.yaml"):   {},
			relPath(t, "sub/dir/deep.xml"): {},
		},
	}

	got, err := f.Find()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 file (intersection), got %d", len(got))
	}
	if got[0].Name != "a.json" {
		t.Errorf("expected a.json, got %s", got[0].Name)
	}
}
