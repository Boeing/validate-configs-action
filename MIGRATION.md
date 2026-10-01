# Migrating from v2 to v3

## Version change

```yaml
# Before
- uses: Boeing/validate-configs-action@v2

# After  
- uses: Boeing/validate-configs-action@v3
```

## Breaking changes

### SchemaStore is now enabled by default

v2 required `schemastore: "true"` to enable automatic schema lookups. v3 enables it by default.

If you don't want SchemaStore validation:

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    schemastore: "false"
```

### Format checking is enabled by default

v3 adds format checking in `warn` mode by default. Config files that are syntactically valid but not canonically formatted will produce `::warning` annotations on the PR diff. This does NOT affect the exit code — your builds won't break.

If you don't want format checking:

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    format-check: "off"
```

To make format issues fail the build:

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    format-check: "strict"
```

### Composite action (no longer Docker)

v3 runs as a composite action with a pre-built binary instead of a Docker container. This means:

- **10-20x faster startup** (2-3s vs 30-60s)
- **macOS runner support** (Docker actions only ran on Linux)
- No change to your workflow — the `uses:` syntax is the same

## New inputs

| Input | Default | Description |
|-------|---------|-------------|
| `format-check` | `"warn"` | Format checking mode: `warn` (warnings, exit 0), `strict` (errors, exit 1), `off` (disabled) |
| `no-config` | `"false"` | Disable all config file discovery (`.cfv.toml`, `.editorconfig`, `.prettierrc`, etc.) |
| `config` | `""` | Explicit path to `.cfv.toml` config file. Skips auto-discovery |

## New output

| Output | Description |
|--------|-------------|
| `files-unformatted` | Number of files that need formatting |

## Changed defaults

| Input | v2 Default | v3 Default |
|-------|-----------|------------|
| `schemastore` | `"false"` | `"true"` |

## Format checking config resolution

Format checking respects your existing formatter configuration:

- **`.cfv.toml`** — If present, it's the sole authority for format settings
- **`.prettierrc`** — Used for JSON/JSONC/YAML formatting (when no `.cfv.toml`)
- **`taplo.toml`** — Used for TOML formatting (when no `.cfv.toml`)
- **`.yamlfmt`** — Used for YAML formatting, takes priority over `.prettierrc` (when no `.cfv.toml`)
- **`.editorconfig`** — Always applied as a base layer (when no `.cfv.toml`)

Use `no-config: "true"` to ignore all config files and use pure defaults.
