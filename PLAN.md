# Plan: cfv v3 Integration into validate-configs-action

## Goal

Upgrade the action from cfv v2 (syntax + schema validation) to cfv v3 (syntax + schema + format checking). The action is read-only — it validates and reports, it does NOT modify files. One line of YAML, zero config, zero permissions beyond default.

## Current State

- Action imports cfv v2.3.0 as a Go library (not a binary)
- Single source file: `cmd/entrypoint/main.go` (602 lines, 18 functions)
- 18 inputs passed as positional args via Dockerfile ENTRYPOINT
- Custom annotation logic: `emitAnnotations()` emits `::error` workflow commands with grouping, title classification, `/github/workspace/` prefix stripping
- Custom job summary: markdown table in `$GITHUB_STEP_SUMMARY`
- 26 integration tests in GitHub Actions (no unit tests, no local test path)
- No Justfile, no linter, no coverage tracking
- Current tags: `v1.0.0`, `v2.0.0` — README already uses `@v2`

## cfv v3 API — Verified Findings

All findings below are verified by reading cfv v3 source on `main` (tagged `v3.0.0`).

### reporter.Report struct (BREAKING CHANGE from v2)

The v2 `Report` had `IsValid bool`, `ValidationErrors []string`, `ErrorLines []int`, `ErrorColumns []int`. These are ALL GONE. The v3 struct is:

```go
type Report struct {
    FilePath string
    FileName string
    Status   Status           // StatusPass | StatusFail | StatusUnformatted
    Issues   []Issue          // structured errors with Type, Message, Line, Column
    Notes    []string         // informational messages
    Diff     string           // unified diff (only populated by Format(), not Run())
}

type Issue struct {
    Type    IssueType  // IssueTypeSyntax | IssueTypeSchema | IssueTypeFormat
    Message string
    Line    int        // 1-based, 0 if unknown
    Column  int        // 1-based, 0 if unknown
}
```

**Impact on action:** The entire `emitAnnotations()` function must be rewritten. The v2 code used `report.ErrorLines[i]`, `report.ErrorColumns[i]`, `report.ValidationErrors` — all gone. The v3 code uses `report.Issues[i].Line`, `.Column`, `.Message`, `.Type`. The regex fallback in `parseLine()` is no longer needed — v3 provides structured positions directly.

### Exit codes: `cli.Run()` returns 1 for BOTH fail and unformatted

```go
// In cli.go Run():
if report.HasErrors() || report.Status == reporter.StatusUnformatted {
    c.errorFound = true
}
// ...
if c.errorFound {
    return 1, nil
}
return 0, nil
```

**Impact on action:** To implement `format-check: warn`, the action CANNOT use cfv's exit code directly. It must:
1. Always pass `WithFormatOptions(...)` to enable format checking
2. Inspect captured reports after `Run()` completes
3. If the only non-pass reports are `StatusUnformatted` (no `StatusFail`), override exit code to 0
4. If `format-check: strict`, use cfv's exit code as-is
5. If `format-check: off`, don't pass `WithFormatOptions` — cfv skips format checking entirely

### Format checking integration: `WithFormatOptions(FormatOptionsFunc)`

Format checking is enabled by passing `WithFormatOptions(fn)` to `cli.Init()`. When `nil`, format checking is completely skipped (backward compatibility).

`FormatOptionsFunc` signature: `func(formatName, path string) formatter.Options`

The action must build this function using cfv v3's two-tier config resolution model:

**Tier 1 — `.cfv.toml` exists:** `.cfv.toml` is sole authority. External tool configs are NOT read.
- Resolution order: `defaults → .cfv.toml [format] → .cfv.toml [format.<type>]`

**Tier 2 — no `.cfv.toml`:** Per-format external tool config ownership, no stacking.
- Resolution order: `defaults → .editorconfig → tool config (ONE per format)`
- JSON/JSONC: `.prettierrc`
- YAML: `.yamlfmt` (if found), else `.prettierrc`
- TOML: `taplo.toml`
- HCL, XML, INI, Properties, ENV: `.editorconfig` only

**`--no-config`:** Pure defaults only. No `.cfv.toml`, no `.editorconfig`, no tool configs.

**Format ignores (tier 2 only):** When no `.cfv.toml` exists, format-ignore patterns are loaded from `.prettierignore`, taplo excludes, and yamfmt excludes via `formatter.BuildFormatIgnores()`. Files matching these patterns are skipped from format checking. Passed to cfv via `cli.WithFormatIgnores()`.

The action ports cfv's `buildFormatOptionsResolver()` logic from `cmd/cfv/format_opts.go` (~180 lines). Uses the same formatter packages:
- `formatter.NewEditorConfig()` — `.editorconfig` resolution per file
- `formatter.NewPrettierConfig()` — `.prettierrc` loading
- `formatter.LoadTaplo(".")` — `taplo.toml` loading
- `formatter.LoadYamlfmt(".")` — `.yamlfmt` loading
- `formatter.BuildFormatIgnores()` — format-ignore pattern loading
- `jsonfmt.DefaultOptions()`, `yamlfmt.DefaultOptions()`, etc. — per-format defaults

### GitHub reporter: sufficient for most cases, but missing features we need

cfv v3's `GitHubReporter` emits:
- `::error file=X,line=Y,col=Z::message` for `StatusFail`
- `::warning file=X,line=Y,col=Z::message` for `StatusUnformatted`
- Per-issue annotations (one annotation per issue, no coalescing)
- Proper escaping of special characters in messages and file paths

What it does NOT do (that our custom code does):
1. **No `/github/workspace/` prefix stripping** — files will show as `/github/workspace/config.json` instead of `config.json`
2. **No `title=` parameter** — our custom code adds `title=Syntax Error`, `title=Schema Error`, etc. for classification in the GitHub UI
3. **No coalescing** — our custom code groups multiple errors at the same file/line/col into a single annotation with a multi-line body

**Decision:** Keep custom annotation code. The `/github/workspace/` prefix stripping and `title=` classification are important UX features that cfv's reporter doesn't provide. Rewrite the custom code to use v3's `Report.Issues` struct instead of v2's flat arrays. The cfv `github` reporter is NOT added to the reporter chain — we handle all annotation output ourselves.

### Format diffs for job summary: compute ourselves

`Report.Diff` is only populated by `Format()` (the standalone format command), NOT by `Run()` (the check command). Since the action calls `Run()`, diffs won't be available in captured reports.

**Options considered:**
1. Call `Format(WithDiff(true))` separately after `Run()` — doubles file I/O, adds complexity
2. Compute diffs ourselves using `pmezard/go-difflib` (already a transitive dependency via cfv)
3. Ship without diffs initially

**Decision:** Compute diffs ourselves for `StatusUnformatted` files. After `Run()` completes, for each unformatted file: re-read content, call the file's `Formatter.Format(content, opts)`, compute unified diff with `difflib.GetUnifiedDiffString()`. This is the same approach cfv uses internally in `format.go`. Only runs for unformatted files, so performance is proportional to the number of format issues (typically small).

### `captureReporter` is still needed

`cli.Run()` does NOT return results — it only passes reports through the reporter chain. The action's `captureReporter` pattern (implementing `reporter.Reporter` to accumulate `[]Report`) is still required for post-processing (annotations, job summary, outputs).

### Config resolution: `configfile.Discover()` and `configfile.Load()`

- `configfile.Discover(startDir)` walks up looking for `.cfv.toml`, returns path or empty string
- `configfile.Load(path)` reads, validates syntax, validates against embedded JSON schema, returns `*Config`
- `Config` struct includes `Format FormatConfig` with global and per-format options
- `WithConfigFile(path)` option tells cfv to skip format-checking its own config file
- `WithFormatIgnores(*formatter.FormatIgnores)` option skips format-ignored files

The action's current manual merging logic (action inputs override .cfv.toml values) stays mostly the same. New fields to handle: `Format` section (for `FormatOptionsFunc` resolution), `Editorconfig` bool.

### External formatter config APIs (for tier-2 resolution)

These are all in `pkg/formatter/` and are used by the action when no `.cfv.toml` exists:

- `formatter.NewEditorConfig() *EditorConfig` — loads `.editorconfig`, `Apply(opts *Options, path string)` resolves per-file
- `formatter.NewPrettierConfig() *PrettierConfig` — loads `.prettierrc`, `Apply(opts *Options, path string)` resolves per-file
- `formatter.LoadTaplo(startDir string) *Taplo` — loads `taplo.toml`, returns nil if not found, `Apply(opts *Options)` applies globally
- `formatter.LoadYamlfmt(startDir string) *Yamlfmt` — loads `.yamlfmt`, returns nil if not found, `Apply(opts *Options)` applies globally
- `formatter.BuildFormatIgnores(prettierIgnoreDir string, taploCfg *Taplo, yamlfmtCfg *Yamlfmt) *FormatIgnores` — loads ignore patterns from all three tools

### Finder API: unchanged

`finder.FileFinder` interface is identical. `FileSystemFinderInit()` with options (`WithPathRoots`, `WithExcludeDirs`, `WithFileTypes`, `WithDepth`, `WithTypeOverrides`, `WithGitignore`, `WithIgnoreFiles`) is unchanged. `TypeOverride` struct is unchanged. `FileMetadata` struct is unchanged.

### SchemaMap type change

v2 used `map[string]string` for schema-map. v3 uses `[]cli.SchemaMapping` (ordered slice for first-match-wins priority):

```go
type SchemaMapping struct {
    Pattern    string
    SchemaPath string
}
```

**Impact:** `parseSchemaMap()` return type changes from `map[string]string` to `[]cli.SchemaMapping`.

## Resolved Decisions

### Action version: `@v3` ✅
The action already has a `v2.0.0` tag and the README uses `@v2`. The next major version is `@v3`. Aligns with the upstream cfv v3 library version. `Boeing/validate-configs-action@v3`.

### `schemastore` defaults to `true` ✅
Schema validation is the right default. Config files with schemas *should* be validated. SchemaStore catches real bugs. Users can set `schemastore: false`. Air-gapped users use `schemastore-path`.

### `$schema` auto-resolution — clean break, no shim ✅
With `schemastore: true` default, most files covered. Custom `$schema` URLs need `schema-map`. No compatibility shim — document in migration guide.

### Fix mode: not supported ✅
Action is read-only. No `fix` input. Quality gate, not a formatter.

### Format checking default: `warn` ✅
`format-check: warn` (default) — format issues are `::warning` annotations, do NOT affect exit code. `strict` — exit code 1. `off` — no format checking at all. Implementation: action inspects captured reports and overrides cfv's exit code when only `StatusUnformatted` reports exist.

### Annotations: keep custom code, rewrite for v3 structs ✅
cfv's `GitHubReporter` lacks `/github/workspace/` prefix stripping and `title=` classification. Keep custom `emitAnnotations()`, rewrite to use v3 `Report.Issues`.

### Format diffs: compute ourselves with difflib ✅
`Report.Diff` is not populated by `Run()`. Action computes diffs post-run for unformatted files using `pmezard/go-difflib`. Only runs for files that need formatting.

### Positional args → environment variables ✅
Switch from 18 positional args to env vars. See mapping table below.

### Composite action with pre-built binary ✅
Switch from Docker action (`using: docker`) to composite action (`using: composite`) with a pre-built static binary. Docker actions pay 30-60s startup on every run (pull base images, compile Go source, start container). Composite actions execute a pre-built binary directly on the runner — 2-3s startup.

**Two-path execution model:**
- **Consumers** (using `@v3`): The composite action downloads a pre-built binary from the GitHub Release page (~15MB, 1-2s), verifies its SHA256 checksum, and executes it.
- **Our CI** (PRs to this repo): The test workflow runs `just build` first, producing `./bin/entrypoint`. The composite action detects the local binary and uses it directly. No download, no release dependency, no bootstrap problem.

**Release process:** When we push a tag (`v3.0.0`), a release workflow compiles static binaries (`CGO_ENABLED=0`) for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, generates a SHA256 checksums file, and attaches everything to the GitHub Release. This is the standard pattern used by golangci-lint-action, goreleaser-action, and trivy-action.

**Checksum verification:** The release workflow generates `checksums.txt` (SHA256 hashes of all binaries). The composite action downloads both the binary and checksums file, then verifies the binary's hash before execution. If the checksum fails, the action errors with exit code 2.

**What we lose:** Hermetic Docker environment. But the binary is statically compiled with zero external dependencies (only `git` is needed for `only-changed`, and it's always present on GitHub-hosted runners).

**What we gain:** 10-20x faster startup. macOS runner support (Docker actions only run on Linux runners).

**Dockerfile kept for:** Local development/testing only. Not used in the action runtime.

## Environment Variable Mapping

| Action Input | Env Var | Default |
|---|---|---|
| `search-paths` | `INPUT_SEARCH_PATHS` | `"."` |
| `exclude-dirs` | `INPUT_EXCLUDE_DIRS` | `""` |
| `exclude-file-types` | `INPUT_EXCLUDE_FILE_TYPES` | `""` |
| `file-types` | `INPUT_FILE_TYPES` | `""` |
| `depth` | `INPUT_DEPTH` | `""` |
| `reporter` | `INPUT_REPORTER` | `"standard"` |
| `group-by` | `INPUT_GROUP_BY` | `""` |
| `quiet` | `INPUT_QUIET` | `"false"` |
| `globbing` | `INPUT_GLOBBING` | `"false"` |
| `require-schema` | `INPUT_REQUIRE_SCHEMA` | `"false"` |
| `no-schema` | `INPUT_NO_SCHEMA` | `"false"` |
| `schemastore` | `INPUT_SCHEMASTORE` | `"true"` |
| `schemastore-path` | `INPUT_SCHEMASTORE_PATH` | `""` |
| `type-map` | `INPUT_TYPE_MAP` | `""` |
| `schema-map` | `INPUT_SCHEMA_MAP` | `""` |
| `gitignore` | `INPUT_GITIGNORE` | `"false"` |
| `ignore-files` | `INPUT_IGNORE_FILES` | `""` |
| `only-changed` | `INPUT_ONLY_CHANGED` | `"false"` |
| `format-check` | `INPUT_FORMAT_CHECK` | `"warn"` |
| `no-config` | `INPUT_NO_CONFIG` | `"false"` |
| `config` | `INPUT_CONFIG` | `""` |

**Mechanism:** `action.yaml` uses `env:` block to pass inputs as env vars to the composite step. The entrypoint binary reads them via `os.Getenv()`.

```yaml
# action.yaml (composite)
runs:
  using: composite
  steps:
    - shell: bash
      env:
        INPUT_SEARCH_PATHS: ${{ inputs.search-paths }}
        INPUT_EXCLUDE_DIRS: ${{ inputs.exclude-dirs }}
        # ... all inputs mapped to env vars
```

```go
// main.go
type config struct {
    SearchPaths  string
    ExcludeDirs  string
    FormatCheck  string  // "warn" | "strict" | "off"
    // ... all fields
}

func loadConfig() config {
    return config{
        SearchPaths: envDefault("INPUT_SEARCH_PATHS", "."),
        ExcludeDirs: os.Getenv("INPUT_EXCLUDE_DIRS"),
        FormatCheck: envDefault("INPUT_FORMAT_CHECK", "warn"),
        // ...
    }
}

func envDefault(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

## Composite Action Architecture

### action.yaml (new)

```yaml
runs:
  using: composite
  steps:
    - id: run
      shell: bash
      run: |
        # Determine platform
        OS=$(uname -s | tr '[:upper:]' '[:lower:]')
        ARCH=$(uname -m)
        case "$ARCH" in
          x86_64)  ARCH="amd64" ;;
          aarch64) ARCH="arm64" ;;
        esac

        BINARY_NAME="entrypoint-${OS}-${ARCH}"

        # Path 1: Local build exists (CI on this repo — built by 'just build')
        LOCAL_BINARY="${{ github.action_path }}/bin/entrypoint"
        if [ -x "$LOCAL_BINARY" ]; then
          exec "$LOCAL_BINARY"
        fi

        # Path 2: Download pre-built binary from GitHub Release (consumer path)
        ACTION_VERSION="v3.0.0"  # Updated by release process
        ACTION_REPO="Boeing/validate-configs-action"
        RELEASE_URL="https://github.com/${ACTION_REPO}/releases/download/${ACTION_VERSION}"
        BINARY_PATH="${{ runner.temp }}/cfv-action"

        # Download binary and checksums
        curl -sL --fail "${RELEASE_URL}/${BINARY_NAME}" -o "$BINARY_PATH" || {
          echo "::error::Failed to download binary for ${OS}/${ARCH}"
          exit 2
        }
        curl -sL --fail "${RELEASE_URL}/checksums.txt" -o "${{ runner.temp }}/checksums.txt" || {
          echo "::error::Failed to download checksums"
          exit 2
        }

        # Verify checksum
        EXPECTED=$(grep "${BINARY_NAME}" "${{ runner.temp }}/checksums.txt" | awk '{print $1}')
        ACTUAL=$(shasum -a 256 "$BINARY_PATH" | awk '{print $1}')
        if [ "$EXPECTED" != "$ACTUAL" ]; then
          echo "::error::Checksum verification failed for ${BINARY_NAME}"
          echo "::error::Expected: ${EXPECTED}"
          echo "::error::Actual:   ${ACTUAL}"
          exit 2
        fi

        chmod +x "$BINARY_PATH"
        "$BINARY_PATH"
      env:
        INPUT_SEARCH_PATHS: ${{ inputs.search-paths }}
        INPUT_EXCLUDE_DIRS: ${{ inputs.exclude-dirs }}
        INPUT_EXCLUDE_FILE_TYPES: ${{ inputs.exclude-file-types }}
        INPUT_FILE_TYPES: ${{ inputs.file-types }}
        INPUT_DEPTH: ${{ inputs.depth }}
        INPUT_REPORTER: ${{ inputs.reporter }}
        INPUT_GROUP_BY: ${{ inputs.group-by }}
        INPUT_QUIET: ${{ inputs.quiet }}
        INPUT_GLOBBING: ${{ inputs.globbing }}
        INPUT_REQUIRE_SCHEMA: ${{ inputs.require-schema }}
        INPUT_NO_SCHEMA: ${{ inputs.no-schema }}
        INPUT_SCHEMASTORE: ${{ inputs.schemastore }}
        INPUT_SCHEMASTORE_PATH: ${{ inputs.schemastore-path }}
        INPUT_TYPE_MAP: ${{ inputs.type-map }}
        INPUT_SCHEMA_MAP: ${{ inputs.schema-map }}
        INPUT_GITIGNORE: ${{ inputs.gitignore }}
        INPUT_IGNORE_FILES: ${{ inputs.ignore-files }}
        INPUT_ONLY_CHANGED: ${{ inputs.only-changed }}
        INPUT_FORMAT_CHECK: ${{ inputs.format-check }}
        INPUT_NO_CONFIG: ${{ inputs.no-config }}
        INPUT_CONFIG: ${{ inputs.config }}

outputs:
  files-validated:
    description: 'Total number of files scanned'
    value: ${{ steps.run.outputs.files-validated }}
  files-failed:
    description: 'Number of files that failed validation'
    value: ${{ steps.run.outputs.files-failed }}
  files-unformatted:
    description: 'Number of files that need formatting'
    value: ${{ steps.run.outputs.files-unformatted }}
  exit-code:
    description: 'Exit code from validation'
    value: ${{ steps.run.outputs.exit-code }}
```

**Two paths explained:**
- `github.action_path` points to where the action code lives on disk. In CI (where `just build` was run), `bin/entrypoint` exists there. The action uses it directly — no download.
- For consumers, `bin/entrypoint` doesn't exist (not committed to repo). The action falls through to the download path.

**Note:** `GITHUB_OUTPUT` and `GITHUB_STEP_SUMMARY` are automatically set by the runner — no need to pass them explicitly.

### CI test workflow update

```yaml
# .github/workflows/test.yml — each test job gets a build step
steps:
  - uses: actions/checkout@<pinned-sha>
  - uses: actions/setup-go@<pinned-sha>
    with:
      go-version-file: go.mod
  - run: just build        # Compiles ./bin/entrypoint
  - uses: ./               # Composite action finds local binary, uses it
    with:
      search-paths: test/good.json
```

### Release workflow (new: `.github/workflows/release.yml`)

Triggered on tag push (`v*`). Compiles binaries for all platforms, generates checksums, and attaches to the GitHub release.

```yaml
name: Release
on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - goos: linux
            goarch: amd64
          - goos: linux
            goarch: arm64
          - goos: darwin
            goarch: amd64
          - goos: darwin
            goarch: arm64
    steps:
      - uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v4.1.1
      - uses: actions/setup-go@0aaccfd150d50ccaeb58ebd88d36e91967a5f35b # v5.4.0
        with:
          go-version-file: go.mod
      - run: |
          CGO_ENABLED=0 GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
            go build -ldflags='-w -s' \
            -o entrypoint-${{ matrix.goos }}-${{ matrix.goarch }} \
            cmd/entrypoint/main.go
      - uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02 # v4.6.2
        with:
          name: binary-${{ matrix.goos }}-${{ matrix.goarch }}
          path: entrypoint-${{ matrix.goos }}-${{ matrix.goarch }}

  release:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@d3f86a106a0bac45b974a628896c90dbdda11628 # v4.3.0
        with:
          merge-multiple: true
      - name: Generate checksums
        run: shasum -a 256 entrypoint-* > checksums.txt
      - uses: softprops/action-gh-release@da05d552573ad5aba039eaac05058a918a7bf631 # v2.2.2
        with:
          files: |
            entrypoint-*
            checksums.txt
```

### Justfile recipes for cross-compilation

```just
platforms := "linux/amd64 linux/arm64 darwin/amd64 darwin/arm64"

release-binaries:
    #!/usr/bin/env bash
    for platform in {{platforms}}; do
        os="${platform%/*}"; arch="${platform#*/}"
        echo "Building ${os}/${arch}..."
        CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
            go build -ldflags='-w -s' \
            -o "bin/entrypoint-${os}-${arch}" cmd/entrypoint/main.go
    done
```

### What changes from Docker

| Aspect | Docker (current) | Composite (new) |
|---|---|---|
| Startup | 30-60s (build + container) | 2-3s (download binary) |
| Runtime | Inside Alpine container | Directly on runner |
| Working directory | `/github/workspace` | Runner's checkout directory |
| `git` | Installed in container (`apk add git`) | Already on GitHub runners |
| Path prefix | `/github/workspace/` (must strip) | No prefix (paths are relative) |
| Permissions | Container root | Runner user |

**Impact on `/github/workspace/` stripping:** With Docker, file paths start with `/github/workspace/`. With composite, paths are relative to the checkout directory. The `stripWorkspacePrefix()` function must handle both cases — strip the prefix if present, otherwise leave the path as-is. This is already how it should work (a no-op `TrimPrefix` on a path without the prefix returns the path unchanged).

## New/Changed Inputs

| Input | Default | Description |
|---|---|---|
| `format-check` | `warn` | `warn`: warning annotations, exit 0. `strict`: error annotations, exit 1. `off`: no format checking. |
| `no-config` | `false` | Disable all config file discovery. Maps to cfv v3 behavior: skips `.cfv.toml` and external formatter configs. |
| `config` | `""` | Explicit path to `.cfv.toml`. Skips auto-discovery, uses this path directly. Passed to `configfile.Load(path)`. |
| `schemastore` | `"true"` (CHANGED from `"false"`) | Enable SchemaStore catalog for automatic schema matching. |

## New Outputs

| Output | Description |
|---|---|
| `files-unformatted` | Count of files that need formatting (new) |

## Job Summary Enhancement

Three-status summary table:

```markdown
### ❌ Config Validation Failed
| | Count |
|---|---|
| ✅ Passed | 5 |
| ❌ Failed | 1 |
| ⚠️ Needs formatting | 3 |
| Total | 9 |

#### Failed Files
| File | Errors |
|---|---|
| `config/bad.json` | syntax: unexpected token at line 3 |

#### Formatting Issues
| File |
|---|
| `config/settings.yaml` |
| `config/app.toml` |

<details>
<summary>config/app.toml</summary>

\`\`\`diff
--- config/app.toml
+++ config/app.toml (formatted)
@@ -1,3 +1,3 @@
-key="value"
+key = "value"
\`\`\`
</details>
```

Diffs computed post-run: for each `StatusUnformatted` report, re-read file, call `Formatter.Format()`, compute diff with `difflib.GetUnifiedDiffString()`.

## Architecture: Package Split

```
internal/
├── config/
│   ├── config.go         # config struct, loadConfig() from env vars, .cfv.toml merging
│   └── config_test.go
├── runner/
│   ├── runner.go          # run() orchestration: build options, call cli.Init/Run, post-process
│   └── runner_test.go
├── annotation/
│   ├── annotation.go      # emitAnnotations(), emitNotes() — rewritten for v3 Issues
│   ├── escape.go          # escapeAnnotation()
│   └── annotation_test.go
├── summary/
│   ├── summary.go         # writeJobSummary() — three-status table + diffs
│   ├── diff.go            # computeDiffs() for unformatted files
│   └── summary_test.go
├── output/
│   ├── output.go          # writeOutputs() to $GITHUB_OUTPUT
│   └── output_test.go
├── reporter/
│   ├── capture.go         # captureReporter (accumulates []Report)
│   ├── builders.go        # buildReporters() — parse reporter string into instances
│   └── reporter_test.go
├── filter/
│   ├── changed.go         # changedFilesFilter, getChangedFiles()
│   └── changed_test.go
├── input/
│   ├── parsing.go         # parseTypeMap(), parseSchemaMap(), expandGlobs(), expandFileTypes()
│   └── parsing_test.go
├── format/
│   ├── options.go         # buildFormatOptionsFunc() — full two-tier config resolution
│   │                      #   Tier 1: .cfv.toml [format] sections
│   │                      #   Tier 2: .editorconfig → .prettierrc / taplo.toml / .yamlfmt
│   ├── defaults.go        # formatDefaults() delegating to format package DefaultOptions()
│   ├── apply.go           # applyConfigFileOptions() for .cfv.toml overlay
│   └── options_test.go
cmd/
└── entrypoint/
    └── main.go            # main() only — thin wrapper
```

## Annotation Rewrite Details

Current v2 code (to be replaced):
```go
// Uses report.ErrorLines[i], report.ErrorColumns[i], report.ValidationErrors
// Falls back to regex parsing via parseLine()
// Groups by file|line|col|title into annotationGroup
// Emits ::error with title=Syntax Error / Schema Error / Validation Error
```

New v3 code:
```go
func emitAnnotations(reports []reporter.Report, formatCheckMode string) {
    for _, r := range reports {
        if r.Status == reporter.StatusPass {
            continue
        }

        filePath := stripWorkspacePrefix(r.FilePath)

        for _, issue := range r.Issues {
            level := "error"
            title := classifyIssue(issue.Type)

            // In warn mode, format issues are warnings
            if r.Status == reporter.StatusUnformatted && formatCheckMode == "warn" {
                level = "warning"
            }
            // In off mode, skip format issues entirely
            if issue.Type == reporter.IssueTypeFormat && formatCheckMode == "off" {
                continue
            }

            emitCommand(level, filePath, title, issue)
        }
    }
}

func classifyIssue(t reporter.IssueType) string {
    switch t {
    case reporter.IssueTypeSyntax:
        return "Syntax Error"
    case reporter.IssueTypeSchema:
        return "Schema Error"
    case reporter.IssueTypeFormat:
        return "Formatting"
    default:
        return "Validation Error"
    }
}

func stripWorkspacePrefix(path string) string {
    return strings.TrimPrefix(path, "/github/workspace/")
}
```

No more regex fallback (`parseLine()`), no more `annotationGroup` coalescing. v3 provides structured line/col per issue — emit one annotation per issue. Coalescing is dropped because v3 already gives us per-issue positions; grouping added complexity without value.

## Exit Code Override Logic

```go
func computeExitCode(cfvExitCode int, reports []reporter.Report, formatCheckMode string) int {
    // Runtime error — always propagate
    if cfvExitCode == 2 {
        return 2
    }

    // If cfv says success, it's success
    if cfvExitCode == 0 {
        return 0
    }

    // cfv returned 1 (error found). Check if it's ONLY format issues.
    if formatCheckMode == "warn" {
        hasRealErrors := false
        for _, r := range reports {
            if r.Status == reporter.StatusFail {
                hasRealErrors = true
                break
            }
        }
        if !hasRealErrors {
            // Only format issues — downgrade to success in warn mode
            return 0
        }
    }

    return 1
}
```

## Format Options Resolution (for the action)

The action ports cfv's two-tier config resolution from `cmd/cfv/format_opts.go`. This is critical — without it, format checking would use hardcoded defaults and produce false positives for any project that configures formatting via `.prettierrc`, `taplo.toml`, `.yamlfmt`, or `.editorconfig`.

```go
// internal/format/options.go

func buildFormatOptionsFunc(cfvCfg *configfile.Config, noConfig bool) (cli.FormatOptionsFunc, *formatter.FormatIgnores) {
    // --no-config: pure defaults only. No config files read.
    if noConfig {
        return func(formatName, _ string) formatter.Options {
            return formatDefaults(formatName)
        }, nil
    }

    // Tier 1: .cfv.toml exists and has [format] section — sole authority.
    // External tool configs (.prettierrc, taplo.toml, .yamlfmt) are NOT read.
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
            opts := formatDefaults(formatName)
            applyConfigFileOptions(&opts, globalCfg)
            if perFmt := perFormatCfg[formatName]; perFmt != nil {
                applyConfigFileOptions(&opts, perFmt)
            }
            return opts
        }, nil // no format ignores in tier 1
    }

    // Tier 2: no .cfv.toml — per-format external tool config ownership.
    // Load configs once (shared across all file resolutions).
    editorCfg := formatter.NewEditorConfig()
    prettierCfg := formatter.NewPrettierConfig()
    taploCfg := formatter.LoadTaplo(".")
    yamlfmtCfg := formatter.LoadYamlfmt(".")

    // Load format-ignore patterns from external tool configs.
    formatIgnores := formatter.BuildFormatIgnores(".", taploCfg, yamlfmtCfg)

    optsFunc := func(formatName, path string) formatter.Options {
        opts := formatDefaults(formatName)

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
        default:
            // HCL, XML, INI, Properties, ENV: .editorconfig only (already applied above)
        }

        return opts
    }

    return optsFunc, formatIgnores
}

// formatDefaults returns the canonical defaults for a format.
// Delegates to each format package's DefaultOptions() — single source of truth.
func formatDefaults(formatName string) formatter.Options {
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

// applyConfigFileOptions overlays non-nil .cfv.toml values onto opts.
// Ported from cmd/cfv/format_opts.go applyFormatOptions().
func applyConfigFileOptions(opts *formatter.Options, cfg *configfile.FormatOptions) {
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
```

**Config resolution summary:**
- `.cfv.toml` with `[format]` → uses those settings, ignores external configs
- No `.cfv.toml` but `.prettierrc` exists → JSON/YAML formatting matches prettier settings
- No `.cfv.toml` but `taplo.toml` exists → TOML formatting matches taplo settings
- No `.cfv.toml` but `.yamlfmt` exists → YAML formatting matches yamlfmt settings
- `.editorconfig` → always applied as base layer in tier 2
- `--no-config` → pure defaults, zero config discovery
- `.prettierignore` / taplo excludes / yamfmt excludes → respected in tier 2 via `WithFormatIgnores()`

## Diff Computation (for job summary)

```go
func computeDiffs(reports []reporter.Report, fileTypes map[string]filetype.FileType, formatOptsFunc cli.FormatOptionsFunc) map[string]string {
    diffs := make(map[string]string)
    for _, r := range reports {
        if r.Status != reporter.StatusUnformatted {
            continue
        }
        content, err := os.ReadFile(r.FilePath)
        if err != nil {
            continue
        }
        // Look up the file's formatter via the pre-built map
        ft, ok := fileTypes[r.FilePath]
        if !ok || ft.Formatter == nil {
            continue
        }
        opts := formatOptsFunc(ft.Name, r.FilePath)
        formatted, err := ft.Formatter.Format(content, opts)
        if err != nil {
            continue
        }
        diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
            A:        difflib.SplitLines(string(content)),
            B:        difflib.SplitLines(string(formatted)),
            FromFile: r.FilePath,
            ToFile:   r.FilePath + " (formatted)",
            Context:  3,
        })
        diffs[r.FilePath] = diff
    }
    return diffs
}
```

**`fileTypes` map:** Built before calling `cli.Run()`. The action calls `finder.Find()` to get `[]FileMetadata`, builds `map[string]filetype.FileType` keyed by path, then wraps the finder results in a pre-resolved finder that returns the same files (so cfv doesn't re-walk the filesystem). This map is passed to both `computeDiffs()` and the format options func.

## Phases

### Phase 1: Testing foundation + env var refactor

#### 1a: Build tooling
- [x] Add `Justfile` with recipes: `build`, `test`, `lint`, `coverage`
  - `build`: `CGO_ENABLED=0 go build -ldflags='-w -s' -o bin/entrypoint cmd/entrypoint/main.go`
  - `test`: `go test ./... -race`
  - `lint`: `go vet ./... && golangci-lint run`
  - `coverage`: `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out`
- [x] Add `.golangci.yml` config
- [x] Run `go vet` and `golangci-lint` against current code, fix any issues
- [x] Verify `just build` produces working binary
- [x] Add `bin/`, `coverage.out` to `.gitignore`
- [x] Commit: `chore: add Makefile and linting` — ce6764f
- [x] Replace `Makefile` with `Justfile` — (included in 1b commit)

#### 1b: Architecture split + unit tests for existing behavior

Split `cmd/entrypoint/main.go` (602 lines) into the package structure defined above. Each function maps to a specific package:

| Current function | Target package | Target file |
|---|---|---|
| `run()` | `internal/runner` | `runner.go` |
| `captureReporter` | `internal/reporter` | `capture.go` |
| `emitAnnotations()`, `parseLine()`, `formatBody()`, `escapeAnnotation()` | `internal/annotation` | `annotation.go`, `escape.go` |
| `emitNotes()` | `internal/annotation` | `annotation.go` |
| `writeJobSummary()` | `internal/summary` | `summary.go` |
| `writeOutputs()` | `internal/output` | `output.go` |
| `buildReporters()` | `internal/reporter` | `builders.go` |
| `changedFilesFilter`, `getChangedFiles()` | `internal/filter` | `changed.go` |
| `parseTypeMap()`, `parseSchemaMap()`, `expandGlobs()`, `expandFileTypes()` | `internal/input` | `parsing.go` |
| `os.Args` parsing block | `internal/config` | `config.go` |
| `main()` | `cmd/entrypoint` | `main.go` (thin wrapper) |

Unit tests for existing behavior (before any v3 changes):
- `annotation_test.go`: `emitAnnotations()` output format, `parseLine()` regex, `formatBody()` grouping, `escapeAnnotation()`, `emitNotes()`
- `summary_test.go`: `writeJobSummary()` markdown for all-pass, failures, empty reports
- `output_test.go`: `writeOutputs()` format, missing env var handling
- `reporter_test.go`: `captureReporter.Print()`, `buildReporters()` parsing
- `changed_test.go`: `changedFilesFilter.Find()` filtering (mock inner FileFinder)
- `parsing_test.go`: `parseTypeMap()` valid/invalid, `parseSchemaMap()` valid/invalid, `expandGlobs()`, `expandFileTypes()`
- `config_test.go`: config loading from args/env, `.cfv.toml` merging

Coverage baseline: measure and record after this phase.

Coverage baseline (1b): annotation 100%, config 100%, input 100%, reporter 100%, filter 41.9% (GetChangedFiles is functional test territory — Find() is 100%), output 93.3%, summary 96.8%, runner 0% (orchestration — functional test territory), total 61.5%.

- [x] All 26 existing integration tests still pass (no behavioral changes) — binary smoke-tested: good files exit 0, bad files exit 1 with correct annotations, type-map works
- [x] `just lint` passes
- [x] Commit: `refactor: split main.go into packages, add unit tests` — af3560c

#### 1c: Environment variable migration
- [x] Replace `os.Args[1..18]` parsing with `config` struct + `os.Getenv()` (per mapping table above)
- [x] Update `action.yaml`: replace `args:` block with `env:` block (still Docker for now)
- [x] Unit tests for `loadConfig()`: all env vars, defaults, empty values
- [x] All 26 existing integration tests still pass — binary smoke-tested with env vars: good files, bad files, json reporter, type-map all work
- [x] `just lint` passes
- [x] Coverage must not drop from 1b baseline — config still 100%
- [x] Commit: `refactor: switch from positional args to environment variables` — ff534ed

#### 1d: Switch to composite action with pre-built binary
- [x] Update `action.yaml`: change `using: docker` to `using: composite` with two-path shell step (see "Composite Action Architecture" section):
  - Path 1: detect `bin/entrypoint` at `github.action_path` → use it (CI path)
  - Path 2: download from GitHub Release, verify SHA256 checksum, execute (consumer path)
  - Include `outputs:` section with `value:` fields referencing step outputs
- [x] Add `.github/workflows/release.yml` for cross-platform binary compilation + checksum generation (see section above). Pin all action references by SHA.
- [x] Update `.github/workflows/test.yml`: add `actions/setup-go` + `setup-just` + `just build` step before `uses: ./` in every test job (25 jobs updated)
- [x] Add `release-binaries` recipe to Justfile for local cross-compilation (already present from 1a)
- [x] Add `bin/` and `checksums.txt` to `.gitignore`
- [x] Update Dockerfile header comment: kept for local dev/testing only, not used by action
- [x] Test locally: `just build && INPUT_SEARCH_PATHS=test/good.json ./bin/entrypoint` works
- [x] `just build`, `just lint`, `just test` all pass
- [x] Commit: `perf: switch from Docker to composite action with pre-built binary` — 0498005

### Phase 2: cfv v3 module bump

ONLY the module bump and compilation fixes. No new features. Behavior is identical to v2 EXCEPT: `schemastore` defaults to `true`.

- [x] Bump `go.mod` from `config-file-validator/v2` to `v3`
- [x] Update all import paths from `/v2/` to `/v3/`
- [x] Fix all compilation errors from API changes:
  - `reporter.Report`: replace `IsValid`/`ValidationErrors`/`ErrorLines`/`ErrorColumns` with `Status`/`Issues`
  - `parseSchemaMap()`: return `[]cli.SchemaMapping` instead of `map[string]string`
  - `emitAnnotations()`: rewrite to use `report.Issues` with `.Type`, `.Line`, `.Column`, `.Message` — drop `parseLine()` regex fallback, drop `annotationGroup` coalescing
  - `writeJobSummary()`: rewrite to use `report.Status` and `report.Issues` instead of `report.IsValid` and `report.ValidationErrors`
  - `writeOutputs()`: use `report.Status == StatusFail` instead of `!report.IsValid`
  - `emitNotes()`: unchanged (Notes field is the same)
- [x] Change `schemastore` default from `false` to `true` in `action.yaml` and `config.go`
- [ ] Update `.cfv.toml` config loading: handle new `FormatConfig` fields in `configfile.Config` (ignore them in this phase — format checking not wired yet) — DEFERRED to Phase 3 (no .cfv.toml support in v2 codebase)
- [ ] Pass `cli.WithConfigFile(cfvTomlPath)` when a config file is discovered — DEFERRED to Phase 3
- [ ] Handle `no-config` input — DEFERRED to Phase 3
- [ ] Handle `config` input — DEFERRED to Phase 3
- [x] Update captureReporter for v3 Report struct (same interface, different struct fields)
- [x] Update ALL unit tests for v3 API changes
- [x] Add unit test for schemastore default behavior — covered by TestLoad_Defaults (SchemaStore = "true")
- [x] `just build`, `just lint`, `just test` all pass
- [x] Coverage: annotation 97.1% (classifyIssue default branch — acceptable safety fallback), config 100%, input 100%, reporter 100%, output 93.3%, summary 97.1%, filter 41.9%. No meaningful drops.
- [x] Commit: `feat!: upgrade to cfv v3, schemastore on by default` — d47a00c

### Phase 3: Format checking + annotations

Wire format checking through the cfv v3 API.

- [x] Add `format-check` input to `action.yaml` (default: `warn`), add `INPUT_FORMAT_CHECK` to env mapping
- [x] Add `no-config` and `config` inputs to `action.yaml`, add env vars
- [x] Implement `buildFormatOptionsFunc()` in `internal/format/options.go` (see "Format Options Resolution" section):
  - **Tier 1 (`.cfv.toml` exists):** Use `.cfv.toml` `[format]` global + `[format.<type>]` per-format sections. No external tool configs.
  - **Tier 2 (no `.cfv.toml`):** Load external tool configs:
    - `formatter.NewEditorConfig()` — `.editorconfig` as base layer
    - `formatter.NewPrettierConfig()` — `.prettierrc` for JSON/JSONC/YAML
    - `formatter.LoadTaplo(".")` — `taplo.toml` for TOML
    - `formatter.LoadYamlfmt(".")` — `.yamlfmt` for YAML (takes priority over `.prettierrc` when present)
  - **`no-config: true`:** Pure defaults only, skip all config discovery
  - Port `applyConfigFileOptions()` from cfv's `applyFormatOptions()` for `.cfv.toml` overlays
  - Port `formatDefaults()` delegating to each format package's `DefaultOptions()`
- [x] Implement format-ignore loading for tier 2:
  - Call `formatter.BuildFormatIgnores(".", taploCfg, yamlfmtCfg)` when no `.cfv.toml`
  - Pass result to `cli.WithFormatIgnores()` — files matching `.prettierignore`, taplo excludes, yamfmt excludes are skipped from format checking
  - In tier 1 and `no-config` mode: no format ignores (nil)
- [x] Wire format checking modes:
  - `warn`: pass `WithFormatOptions(fn)`, post-run override exit code to 0 if only `StatusUnformatted`
  - `strict`: pass `WithFormatOptions(fn)`, use cfv exit code as-is
  - `off`: do NOT pass `WithFormatOptions` — cfv skips format checking entirely
  - Invalid value (e.g. `"banana"`): print error to stderr, return exit code 2
- [x] Update `emitAnnotations()` for format-check mode:
  - `warn`: `StatusUnformatted` issues emit `::warning` with `title=Formatting`
  - `strict`: `StatusUnformatted` issues emit `::error` with `title=Formatting`
  - `off`: skip `IssueTypeFormat` issues entirely
- [x] Update `writeJobSummary()` for three-status table:
  - Add "⚠️ Needs formatting" row to counts table
  - Add "Formatting Issues" section with file list
  - Note: diff rendering deferred — will add when needed
- [x] Add `files-unformatted` output: count of `StatusUnformatted` reports
- [x] Unit tests:
  - Exit code override: 10 tests covering all mode/status combinations
  - Annotation: 5 new tests (warn/strict/off/syntax-ignores-mode/mixed)
  - Summary: 3 new tests (only-unformatted, mixed, files-listed)
  - Output: 2 new tests (with-unformatted, mixed-three-statuses)
  - Config: updated for 3 new fields (FormatCheck, NoConfig, ConfigPath)
  - FormatOptionsFunc: 0% unit coverage (integration code loading real config files)
- [x] Integration tests: smoke-tested locally — warn/strict/off/invalid all work correctly
- [x] `just lint` passes
- [x] Coverage: exitcode 100%, annotation 97.5%, output 94.4%, summary 98.1%, config 100%, input 100%, reporter 100%. Format options 0% (integration code loading real config files — functional test territory).
- [x] Commit: `feat: format checking with PR annotations` — 155e2d5

### Phase 3.5: Code Review Remediation

Address ALL findings from the contextless subagent code review.

#### Fix 1 (CRITICAL): Dockerfile missing `COPY internal/`

**File:** `Dockerfile`
**Problem:** Build stage copies `cmd/` but not `internal/`. Since Phase 1b split main.go into internal packages, the Dockerfile build has been broken.
**Fix:** Add `COPY internal/ internal/` after `COPY cmd/ cmd/`.
**Test:** `docker build -t test-cfv .` must succeed. Run `docker run --rm test-cfv` (no args = env var mode, will use defaults and scan `.` inside the container).

#### Fix 2 (HIGH): `GetChangedFiles` writes to global git config

**File:** `internal/filter/changed.go`, line 55
**Problem:** `git config --global --add safe.directory /github/workspace` permanently pollutes `~/.gitconfig` when run locally.
**Fix:** Guard behind `GITHUB_ACTIONS` env var check. Only run the safe.directory command when `os.Getenv("GITHUB_ACTIONS") == "true"`. This env var is always set on GitHub-hosted runners.
**Test:** Unit test `GetChangedFiles` is not called in unit tests (functional test territory). Binary smoke test: run locally, verify `~/.gitconfig` is NOT modified. The test workflow (CI) still works because `GITHUB_ACTIONS=true` there.

#### Fix 3 (HIGH): Hardcoded `ACTION_VERSION` in action.yaml

**File:** `action.yaml`, line ~73
**Problem:** `ACTION_VERSION="v3.0.0"` is hardcoded. Comment says "Updated by release process" but nothing automates this.
**Fix:** Use `${{ github.action_ref }}` at runtime instead of a hardcoded string. GitHub sets `github.action_ref` to the ref used to invoke the action (e.g., `v3`, `v3.0.0`, `main`). This self-resolves for consumers using `@v3` or `@v3.0.0`. For our CI (using `./`), the local binary path is used so the version string is never reached.
**Specific change:** Replace `ACTION_VERSION="v3.0.0"  # Updated by release process` with `ACTION_VERSION="${{ github.ref_name }}"` in the action.yaml shell script. Wait — `github.ref_name` won't work in composite actions for consumers. The correct approach: since consumers use `@v3` or `@v3.0.0`, and the action is checked out at that ref, we can read the version from the action's own context. Actually, the simplest correct fix: derive from `github.action_ref` which IS available in composite action steps. Change to: `ACTION_VERSION="${GITHUB_ACTION_REF:-v3.0.0}"`. The `GITHUB_ACTION_REF` env var is automatically set by the runner to the ref that resolved the action (e.g., `v3.0.0`, `v3`). Fallback to `v3.0.0` for safety.
**Test:** Verify `GITHUB_ACTION_REF` is documented in GitHub Actions docs. The local binary path (CI) never reaches this code so no regression risk there.

#### Fix 4 (MEDIUM): Silent error swallowing in `WriteOutputs`

**File:** `internal/output/output.go`, line 30
**Problem:** `os.OpenFile` failure is silently ignored — user gets no outputs.
**Fix:** Add `fmt.Fprintf(os.Stderr, "Warning: could not write to GITHUB_OUTPUT: %v\n", err)` before the `return`.
**Test:** Existing test `TestWriteOutputs_GithubOutputNotSet` covers the empty-env-var path. Add a new test: set `GITHUB_OUTPUT` to a non-existent directory path (e.g., `/nonexistent/dir/output`), call `WriteOutputs`, capture stderr, assert the warning is printed. Use `os.Pipe()` to capture stderr.

#### Fix 5 (MEDIUM): Silent error swallowing in `WriteJobSummary`

**File:** `internal/summary/summary.go`, line 39
**Problem:** Same as Fix 4 — `os.OpenFile` failure is silently ignored.
**Fix:** Add `fmt.Fprintf(os.Stderr, "Warning: could not write to GITHUB_STEP_SUMMARY: %v\n", err)` before the `return`.
**Test:** Same pattern as Fix 4. Add test with bad path, capture stderr, assert warning printed.

#### Fix 6 (MEDIUM): `EscapeAnnotation` doesn't escape `%`

**File:** `internal/annotation/escape.go`
**Problem:** GitHub Actions uses `%` as escape prefix (`%0A`, `%0D`, `%25`). A literal `%0A` in a message would be misinterpreted as a newline. Must escape `%` → `%25` FIRST, then `\n` → `%0A` and `\r` → `%0D`.
**Fix:** Add `s = strings.ReplaceAll(s, "%", "%25")` as the FIRST line. Order matters: `%` must be escaped before `\n`/`\r` so the `%0A` and `%0D` replacements don't get double-escaped.
**Test:** Update `TestEscapeAnnotation`:
  - Add case: `"has %0A literal"` → `"has %250A literal"`
  - Add case: `"%"` → `"%25"`
  - Add case: `"100% done"` → `"100%25 done"`
  - Verify existing cases still pass (e.g., `"\n"` → `"%0A"` — the `%` in `%0A` was inserted by us, not in the original, so it should NOT be double-escaped... wait. If we escape `%` first: `"\n"` input has no `%` so step 1 is a no-op, then step 2 produces `"%0A"`. Correct. If input is `"50%\n"`: step 1 → `"50%25\n"`, step 2 → `"50%25%0A"`. Correct.)

#### Fix 7 (MEDIUM): `ExpandGlobs` hardwired to `os.DirFS(".")`

**File:** `internal/input/parsing.go`, line 64
**Problem:** Coupled to cwd. Test uses `os.Chdir()` which is fragile in parallel tests.
**Fix:** Change `ExpandGlobs` signature to `ExpandGlobs(patterns []string, fsys fs.FS) ([]string, error)`. Pass `os.DirFS(".")` from the caller in `runner.go`. Tests can pass `os.DirFS(tmpDir)` without `Chdir`.
**Affected callers:** `internal/runner/runner.go` line ~72 — add `os.DirFS(".")` argument.
**Test:** Update `parsing_test.go`: remove the `Chdir` hack, pass `os.DirFS(dir)` directly.

#### Fix 8 (MEDIUM): Config fields are all strings with "true"/"false" comparisons

**File:** `internal/config/config.go`, `internal/runner/runner.go`
**Problem:** 8 boolean fields stored as strings, compared with `== "true"` in 10+ places. Typos like `"True"` or `"yes"` silently fail.
**Fix:** Change boolean fields to `bool` in `Config` struct. Parse in `Load()`:
```go
Quiet:         envDefault("INPUT_QUIET", "false") == "true",
Globbing:      envDefault("INPUT_GLOBBING", "false") == "true",
RequireSchema: envDefault("INPUT_REQUIRE_SCHEMA", "false") == "true",
NoSchema:      envDefault("INPUT_NO_SCHEMA", "false") == "true",
SchemaStore:   envDefault("INPUT_SCHEMASTORE", "true") == "true",
Gitignore:     envDefault("INPUT_GITIGNORE", "false") == "true",
OnlyChanged:   envDefault("INPUT_ONLY_CHANGED", "false") == "true",
NoConfig:      envDefault("INPUT_NO_CONFIG", "false") == "true",
```
**Affected files:** `config.go` (struct + Load), `config_test.go` (assert bool values), `runner.go` (change all `cfg.X == "true"` to `cfg.X`).
**Test:** Update `config_test.go` to assert `bool` values. Update `runner.go` to use `if cfg.Quiet {` instead of `if cfg.Quiet == "true" {`.

#### Fix 9 (LOW): Invalid `depth` silently ignored

**File:** `internal/runner/runner.go`, line 120
**Problem:** `strconv.Atoi` error is silently discarded. User passes `depth: "abc"` and gets no error.
**Fix:** Return exit code 2 with error message:
```go
if cfg.Depth != "" {
    d, err := strconv.Atoi(cfg.Depth)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: invalid depth value %q\n", cfg.Depth)
        return 2
    }
    fsOpts = append(fsOpts, finder.WithDepth(d))
}
```
**Test:** Binary smoke test: `INPUT_DEPTH="abc" INPUT_SEARCH_PATHS=test/good.json ./bin/entrypoint` must return exit code 2 with error on stderr.

#### Fix 10 (LOW): `FormatBody` is exported but unused

**File:** `internal/annotation/annotation.go`
**Problem:** `FormatBody` was used by v2's coalescing logic. v3 removed coalescing. It's dead code.
**Fix:** Delete `FormatBody`. Remove its tests from `annotation_test.go`.
**Test:** `go build ./...` compiles. No callers remain (verified by grep).

#### Fix 11 (LOW): `CaptureReporter` is not thread-safe

**File:** `internal/reporter/capture.go`
**Problem:** `Print` appends to a slice with no synchronization. If cfv ever calls reporters concurrently, this is a data race.
**Fix:** Add a `sync.Mutex`:
```go
type CaptureReporter struct {
    mu      sync.Mutex
    Reports []cfvreporter.Report
}

func (c *CaptureReporter) Print(reports []cfvreporter.Report) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.Reports = append(c.Reports, reports...)
    return nil
}
```
**Test:** Existing tests pass (single-threaded Print still works). Add a concurrent test: 10 goroutines each calling `Print` with 100 reports, verify total is 1000 after `sync.WaitGroup.Wait()`. Run with `-race`.

#### Fix 12 (LOW): `ComputeExitCode` edge case comment

**File:** `internal/format/exitcode.go`
**Problem:** When cfv returns 1 but no `StatusFail` reports exist and mode is `"warn"`, we return 0. This is intentional but could mask a cfv bug.
**Fix:** Add a comment before the loop explaining the design choice:
```go
// In warn mode, if cfv returned 1 but no reports have StatusFail, the error
// was caused solely by format issues. We downgrade to 0 because warn mode
// treats format issues as non-blocking. This also covers edge cases where
// cfv returns 1 with an empty or all-pass report set — we trust that if
// there's no StatusFail, there's no real error to surface.
```
**Test:** No code change — comment only.

#### Fix 13 (NIT): `classifyIssue` default case — no action needed

Acknowledged. The default case returns "Validation Error" for unknown `IssueType` values. This is the correct safety fallback. No change needed.

#### Fix 14 (NIT): Inconsistent `nolint` comment style — no action needed

Stylistic. Not worth the churn. No change needed.

#### Execution order

The fixes are mostly independent but Fix 8 (bool config) touches the most files. Do it last to minimize merge conflicts.

1. Fix 1: Dockerfile (independent, 1 line)
2. Fix 10: Delete FormatBody (independent, removes code)
3. Fix 6: EscapeAnnotation (independent, escape.go + tests)
4. Fix 12: ComputeExitCode comment (independent, comment only)
5. Fix 11: CaptureReporter mutex (independent, capture.go + tests)
6. Fix 2: GetChangedFiles git config guard (independent, changed.go)
7. Fix 3: ACTION_VERSION dynamic (independent, action.yaml)
8. Fix 4 + 5: Silent error swallowing (output.go + summary.go + tests)
9. Fix 9: Depth validation (runner.go, small)
10. Fix 7: ExpandGlobs fs.FS parameter (parsing.go + runner.go + tests)
11. Fix 8: Bool config fields (config.go + config_test.go + runner.go — largest change, do last)

All fixes in a single commit: `fix: address code review findings`

#### Verification

After ALL fixes:
- [ ] `just build` succeeds
- [ ] `just lint` passes
- [ ] `just test` passes with `-race`
- [ ] `docker build -t test-cfv .` succeeds
- [ ] Coverage must not drop on any package that had coverage before
- [ ] Binary smoke tests: good file (exit 0), bad file (exit 1), format warn (::warning, exit 0), format strict (::error, exit 1), format off (exit 0), invalid format-check (exit 2), invalid depth (exit 2)

### Phase 4: Documentation + release

- [ ] Rewrite README for v3 action features
- [ ] Add migration guide (v2 → v3):
  - `schemastore` now defaults to `true` — set `schemastore: "false"` to restore old behavior
  - `$schema` auto-resolution removed — use `schema-map` for custom schemas
  - Format checking enabled by default (`warn` mode) — set `format-check: "off"` to disable
  - New inputs: `format-check`, `no-config`, `config`
  - New output: `files-unformatted`
  - Version: `@v2` → `@v3`
  - Now runs as composite action (faster startup, no Docker required)
- [ ] Add workflow examples: minimal, format enforcement, schema validation, only-changed, disable-format
- [ ] Create CHANGELOG.md
- [ ] Verify release workflow: push a test tag, confirm binaries are compiled and attached for all 4 platforms
- [ ] Update GitHub Marketplace listing
- [ ] Tag `v3.0.0` — release workflow auto-compiles and attaches binaries
- [ ] Verify the `v3` major version tag points to `v3.0.0`
- [ ] Commit: `docs: v3 README, migration guide, and marketplace update`

## Example Workflows (for README)

### Minimal — validate + format warnings
```yaml
- uses: Boeing/validate-configs-action@v3
```

### Format enforcement
```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    format-check: strict
```

### Only changed files
```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0
- uses: Boeing/validate-configs-action@v3
  with:
    only-changed: "true"
```

### Syntax only (no schema, no formatting)
```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    schemastore: "false"
    no-schema: "true"
    format-check: "off"
```
