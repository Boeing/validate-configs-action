# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [3.0.0] - 2026-10-01

### Added

- **Format checking** — validates that config files are canonically formatted. Three modes:
  - `warn` (default): format issues appear as `::warning` annotations on the PR diff, exit code 0
  - `strict`: format issues appear as `::error` annotations, exit code 1
  - `off`: format checking disabled
- **Format config resolution** — respects `.cfv.toml`, `.prettierrc`, `taplo.toml`, `.yamlfmt`, and `.editorconfig`
- New input: `format-check` — controls format checking mode (default: `warn`)
- New input: `no-config` — disables all config file discovery
- New input: `config` — explicit path to `.cfv.toml` config file
- New output: `files-unformatted` — count of files that need formatting
- Three-status job summary table: ✅ passed, ❌ failed, ⚠️ needs formatting
- Formatting issues section in job summary with file list

### Changed

- **BREAKING:** `schemastore` now defaults to `"true"`. Set `schemastore: "false"` to restore previous behavior.
- **BREAKING:** Upgraded from cfv v2 to cfv v3. The v3 library provides structured error positions (line/column per issue), new status types (pass/fail/unformatted), and format checking capabilities.
- Switched from Docker action to composite action with pre-built binary. 10-20x faster startup (2-3s vs 30-60s). Now supports macOS runners.
- Inputs are now passed via environment variables instead of positional arguments.
- Annotations now emit one annotation per issue (v2 coalesced multiple errors at the same location).
- Annotation messages are cleaner — line/column info is in the annotation metadata, not duplicated in the message body.
- `%` characters in error messages are now properly escaped in annotations.

### Removed

- Docker-based execution (Dockerfile kept for local development only).
- Annotation coalescing (v3 provides per-issue positions, making coalescing unnecessary).
- `parseLine()` regex fallback (v3 provides structured line/column data directly).

### Fixed

- Invalid `depth` values are now rejected with a clear error (previously silently ignored).
- `git config --global` for safe.directory is now only set when running in GitHub Actions (previously polluted local git config).

[3.0.0]: https://github.com/Boeing/validate-configs-action/releases/tag/v3.0.0
