package filter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Boeing/config-file-validator/v2/pkg/finder"
)

// ChangedFilesFilter wraps a FileFinder and filters results to only changed files.
type ChangedFilesFilter struct {
	Inner   finder.FileFinder
	Changed map[string]struct{}
}

// Find returns only files that appear in the Changed set.
func (f *ChangedFilesFilter) Find() ([]finder.FileMetadata, error) {
	all, err := f.Inner.Find()
	if err != nil {
		return nil, err
	}
	var filtered []finder.FileMetadata
	for _, file := range all {
		rel := file.Path
		if abs, err := filepath.Abs(file.Path); err == nil {
			if wd, err := os.Getwd(); err == nil {
				if r, err := filepath.Rel(wd, abs); err == nil {
					rel = r
				}
			}
		}
		if _, ok := f.Changed[rel]; ok {
			filtered = append(filtered, file)
		}
	}
	return filtered, nil
}

// GetChangedFiles uses git to determine which files changed in the current PR.
func GetChangedFiles() (map[string]struct{}, error) {
	baseBranch := os.Getenv("GITHUB_BASE_REF")
	if baseBranch == "" {
		return nil, fmt.Errorf("GITHUB_BASE_REF not set (not a pull request?)")
	}

	// Docker containers run as root but the workspace is owned by the runner user
	safe := exec.Command("git", "config", "--global", "--add", "safe.directory", "/github/workspace")
	_ = safe.Run() // best-effort; non-fatal if it fails

	fetch := exec.Command("git", "fetch", "origin", baseBranch, "--depth=1")
	fetch.Stderr = os.Stderr
	if err := fetch.Run(); err != nil {
		return nil, fmt.Errorf("git fetch: %w", err)
	}

	cmd := exec.Command("git", "diff", "--name-only", "origin/"+baseBranch+"...HEAD")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	changed := make(map[string]struct{})
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			changed[line] = struct{}{}
		}
	}
	return changed, nil
}
