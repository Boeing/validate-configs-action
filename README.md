# Validate Configs Action

<p>
  <a href="https://scorecard.dev/viewer/?uri=github.com/Boeing/validate-configs-action">
    <img src="https://api.scorecard.dev/projects/github.com/Boeing/validate-configs-action/badge" alt="OpenSSF Scorecard">
  </a>
  <a href="https://opensource.org/licenses/Apache-2.0">
    <img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="Apache 2 License">
  </a>
</p>

A GitHub Action that catches broken config files in your PRs before they cause problems. Supports JSON, YAML, TOML, XML, INI, HCL, JSONC, KDL, Justfiles, and a bunch more. Errors show up as inline annotations on the PR diff so you know exactly what broke and where.

v3 also checks formatting out of the box — badly formatted files get `::warning` annotations so you can see them without failing the build. [SchemaStore](#schemastore) is now enabled by default, so common files like `package.json` and `tsconfig.json` get schema-validated automatically.

If a file has a `$schema` or XSD declaration, it gets schema-validated too.

## What's new in v3

- **Format checking** is on by default in `warn` mode. Format issues surface as warnings on the PR diff without failing the build. Set to `strict` to fail, or `off` to disable.
- **SchemaStore** is enabled by default. No need to opt in anymore.
- **Composite action** — runs natively on the runner (no Docker). Faster startup and supports macOS runners.
- **Config file support** — use a `.cfv.toml` file to configure the action instead of (or alongside) action inputs.

## Quick Start

```yaml
- uses: Boeing/validate-configs-action@v3
```

Add it to any job with `actions/checkout` and it'll validate every config file in the repo. SchemaStore lookups and format warnings are on by default.

## Usage

```yaml
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: Boeing/validate-configs-action@v3
```

Works on `ubuntu-latest`, `macos-latest`, and other GitHub-hosted runners.

### Only validate changed files

If you've got a big repo, you probably don't want to validate everything on every PR:

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0
- uses: Boeing/validate-configs-action@v3
  with:
    only-changed: "true"
```

You need `fetch-depth: 0` so git can figure out which files actually changed.

### Exclude directories

Skip the stuff you don't care about:

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    exclude-dirs: "vendor,testdata,node_modules"
```

### Respect ignore files

Already have a `.dockerignore` or `.prettierignore`? Use those same patterns to skip files:

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    ignore-files: ".dockerignore,.prettierignore"
```

### SchemaStore

[SchemaStore](https://www.schemastore.org/) is a community catalog of JSON Schemas for common config files. In v3, it's enabled by default — you get schema validation for hundreds of files out of the box with zero config.

To disable it:

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    schemastore: "false"
```

### Format Checking

Format checking catches inconsistent formatting (trailing whitespace, wrong indentation, missing final newlines, etc.) and shows them as annotations on the PR diff.

It's on by default in `warn` mode — format issues show as `::warning` annotations but don't fail the build. Three modes:

| Mode | Behavior |
|------|----------|
| `warn` (default) | Warning annotations, exit 0 |
| `strict` | Error annotations, exit 1 |
| `off` | No format checking |

#### Warn mode (default)

Nothing to configure. Format issues show up as warnings:

```yaml
- uses: Boeing/validate-configs-action@v3
```

#### Strict mode — fail on format issues

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    format-check: "strict"
```

#### Disable format checking

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    format-check: "off"
```

### Outputs

```yaml
- uses: Boeing/validate-configs-action@v3
  id: validate
  continue-on-error: true
- run: |
    echo "Files validated: ${{ steps.validate.outputs.files-validated }}"
    echo "Files failed: ${{ steps.validate.outputs.files-failed }}"
    echo "Files unformatted: ${{ steps.validate.outputs.files-unformatted }}"
```

## Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `search-paths` | `"."` | Space-separated dirs or files to scan |
| `exclude-dirs` | `""` | Comma-separated dirs to skip |
| `exclude-file-types` | `""` | Comma-separated file types to skip (e.g. `csv,xml`) |
| `file-types` | `""` | Only validate these file types. Can't use with `exclude-file-types` |
| `depth` | `""` | Recursion depth. `0` means don't recurse |
| `reporter` | `"standard"` | Output format. Options: `standard`, `json`, `junit`, `sarif`, `github`. Use `type:path` for file output, comma-separate for multiple |
| `group-by` | `""` | Group output: `filetype`, `directory`, or `pass-fail` |
| `quiet` | `"false"` | Suppress stdout |
| `globbing` | `"false"` | Use glob patterns in `search-paths`. Can't combine with `exclude-dirs`/`exclude-file-types`/`file-types` |
| `require-schema` | `"false"` | Fail files that could have a schema but don't declare one |
| `no-schema` | `"false"` | Skip all schema validation, syntax only |
| `schemastore` | `"true"` | Auto-lookup schemas from the embedded [SchemaStore](https://www.schemastore.org/) catalog |
| `schemastore-path` | `""` | Path to a local SchemaStore clone (for air-gapped envs). Implies `schemastore` |
| `type-map` | `""` | Map globs to file types: `pattern:type` (e.g. `**/inventory:ini`) |
| `schema-map` | `""` | Map globs to schemas: `pattern:schema_path` (e.g. `**/config.json:schemas/config.schema.json`) |
| `only-changed` | `"false"` | Only check files changed in the current PR |
| `gitignore` | `"false"` | Skip files matched by `.gitignore` |
| `ignore-files` | `""` | Comma-separated list of gitignore-style pattern files to use as filters (e.g. `.dockerignore,.prettierignore`) |
| `format-check` | `"warn"` | Format checking mode. `warn`: warning annotations, exit 0. `strict`: error annotations, exit 1. `off`: no format checking |
| `no-config` | `"false"` | Disable all config file discovery (`.cfv.toml`, `.editorconfig`, `.prettierrc`, etc.) |
| `config` | `""` | Explicit path to `.cfv.toml` config file. Skips auto-discovery |

### Supported file types

`csv`, `editorconfig`, `env`, `hcl` (`.tf`, `.tfvars`), `hocon`, `ini`, `json`, `jsonc`, `justfile`, `kdl`, `plist`, `properties`, `sarif`, `toml`, `toon`, `xml`, `yaml`

Plus ~90 well-known filenames (`.babelrc`, `tsconfig.json`, `Pipfile`, `pom.xml`, etc.) are auto-detected even without a file extension.

## Outputs

| Output | Description |
|--------|-------------|
| `files-validated` | Total files scanned |
| `files-failed` | Files that failed validation |
| `files-unformatted` | Files that need formatting |
| `exit-code` | `0` = success, `1` = validation errors, `2` = runtime error |

## Examples

<details>
<summary><b>Filtering</b></summary>

#### Only specific file types

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    file-types: "json,yaml"
```

#### Exclude file types

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    exclude-file-types: "csv,xml"
```

#### No recursion

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    depth: 0
```

#### Glob patterns

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    globbing: "true"
    search-paths: "**/*.json config/**/*.yaml"
```

#### Use existing ignore files

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    ignore-files: ".dockerignore,.prettierignore"
```

</details>

<details>
<summary><b>Reporters</b></summary>

#### GitHub annotations (inline on the PR diff)

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    reporter: "github"
```

#### JSON

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    reporter: "json"
```

#### Multiple reporters with file output

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    reporter: "standard,json:output.json,junit:results.xml,sarif:results.sarif"
```

#### Group by pass/fail

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    group-by: "pass-fail"
```

</details>

<details>
<summary><b>Schema Validation</b></summary>

#### Disable SchemaStore

SchemaStore is on by default in v3. To turn it off:

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    schemastore: "false"
```

#### Local SchemaStore (air-gapped)

```yaml
- run: git clone --depth=1 https://github.com/SchemaStore/schemastore.git
- uses: Boeing/validate-configs-action@v3
  with:
    schemastore-path: "./schemastore"
```

#### Require schemas

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    require-schema: "true"
```

#### Syntax only (no schema validation)

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    no-schema: "true"
```

#### Map custom schemas to files

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    schema-map: "**/package.json:schemas/package.schema.json,**/config.xml:schemas/config.xsd"
```

</details>

<details>
<summary><b>Format Checking</b></summary>

#### Warn mode (default — no config needed)

Format issues appear as `::warning` annotations on the PR diff. The build still passes:

```yaml
- uses: Boeing/validate-configs-action@v3
```

#### Strict mode — fail on format issues

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    format-check: "strict"
```

#### Disable format checking

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    format-check: "off"
```

#### Track unformatted files in outputs

```yaml
- uses: Boeing/validate-configs-action@v3
  id: validate
  continue-on-error: true
- run: |
    if [ "${{ steps.validate.outputs.files-unformatted }}" != "0" ]; then
      echo "⚠️ ${{ steps.validate.outputs.files-unformatted }} files need formatting"
    fi
```

</details>

<details>
<summary><b>Config File</b></summary>

#### Use a .cfv.toml config file

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    config: ".cfv.toml"
```

#### Disable config file discovery

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    no-config: "true"
```

</details>

<details>
<summary><b>Advanced</b></summary>

#### Kitchen sink

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0
- uses: Boeing/validate-configs-action@v3
  with:
    only-changed: "true"
    exclude-dirs: "vendor,generated,testdata"
    ignore-files: ".prettierignore"
    schema-map: "**/app-config.json:schemas/app.schema.json"
    type-map: "**/inventory:ini,**/.env.*:env"
    reporter: "standard,junit:results.xml"
    format-check: "strict"
```

#### Map unrecognized files to a type

```yaml
- uses: Boeing/validate-configs-action@v3
  with:
    type-map: "**/inventory:ini,**/*.cfg:json"
```

#### Use outputs for conditional logic

```yaml
- uses: Boeing/validate-configs-action@v3
  id: validate
  continue-on-error: true
- run: |
    if [ "${{ steps.validate.outputs.files-failed }}" != "0" ]; then
      echo "😬 ${{ steps.validate.outputs.files-failed }} files failed"
      exit 1
    fi
```

</details>
