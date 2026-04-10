# Validate Configs Github Action

> [GitHub Action](https://github.com/features/actions) for [config-file-validator](https://github.com/Boeing/config-file-validator)

<p>
  <a href="https://scorecard.dev/viewer/?uri=github.com/Boeing/validate-configs-action">
    <img src="https://api.scorecard.dev/projects/github.com/Boeing/validate-configs-action/badge" alt="OpenSSF Scorecard">
  </a>
  <a href="https://opensource.org/licenses/Apache-2.0">
    <img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="Apache 2 License">
  </a>
</p>

Validate configuration files in your repository and get inline PR annotations for errors. Supports syntax validation and JSON Schema validation for 16 file formats.

## Supported File Types

| Format | Syntax | Schema |
|--------|:------:|:------:|
| Apple PList XML | ✅ | |
| CSV | ✅ | |
| EDITORCONFIG | ✅ | |
| ENV | ✅ | |
| HCL | ✅ | |
| HOCON | ✅ | |
| INI | ✅ | |
| JSON | ✅ | ✅ |
| JSONC | ✅ | ✅ |
| Properties | ✅ | |
| SARIF | ✅ | ✅ |
| TOML | ✅ | ✅ |
| TOON | ✅ | ✅ |
| XML | ✅ | ✅ |
| YAML | ✅ | ✅ |

## Quick Start

```yaml
- uses: Boeing/validate-configs-action@v2
```

That's it. By default the action scans the entire repository for configuration files, validates them, and fails the workflow if any are invalid. Errors appear as inline annotations on the PR diff.

## Usage

### Basic

```yaml
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: Boeing/validate-configs-action@v2
```

### Validate specific paths

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    search-paths: ./configs ./deploy
```

### Validate only changed files in a PR

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0
- uses: Boeing/validate-configs-action@v2
  with:
    only-changed: "true"
```

### Schema validation with SchemaStore

Automatically validate files against [SchemaStore](https://www.schemastore.org/) schemas (e.g. `package.json`, `tsconfig.json`, GitHub Actions workflows):

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    schemastore: "true"
```

### Using outputs

```yaml
- uses: Boeing/validate-configs-action@v2
  id: validate
  continue-on-error: true
- run: |
    echo "Files validated: ${{ steps.validate.outputs.files-validated }}"
    echo "Files failed: ${{ steps.validate.outputs.files-failed }}"
```

## PR Inline Annotations

Validation errors automatically appear as inline annotations on pull request diffs. When a config file fails validation, the action emits GitHub Actions workflow commands that annotate the exact file and line where the error occurred. For config types that do not support line numbers in the error output, an annotation will be added at line 1.

## Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `search-paths` | `"."` | Space-separated list of directories or files to scan |
| `exclude-dirs` | `""` | Comma-separated list of subdirectories to exclude |
| `exclude-file-types` | `""` | Comma-separated list of file extensions to exclude |
| `file-types` | `""` | Comma-separated list of file types to validate. Cannot be used with `exclude-file-types` |
| `depth` | `""` | Recursion depth limit. `0` disables recursion |
| `reporter` | `"standard"` | Report format(s). Options: `standard`, `json`, `junit`, `sarif`. Supports `type:path` for file output |
| `group-by` | `""` | Group output by `filetype`, `directory`, or `pass-fail` |
| `quiet` | `"false"` | Suppress all output to stdout |
| `globbing` | `"false"` | Enable glob pattern matching for search paths |
| `require-schema` | `"false"` | Fail files that support schema validation but don't declare a schema |
| `no-schema` | `"false"` | Disable all schema validation (syntax-only) |
| `schemastore` | `"false"` | Enable automatic schema lookup using the embedded [SchemaStore](https://www.schemastore.org/) catalog |
| `schemastore-path` | `""` | Path to a local SchemaStore clone. For air-gapped environments. Implies `schemastore` |
| `type-map` | `""` | Map glob patterns to file types. Format: `pattern:type` |
| `schema-map` | `""` | Map glob patterns to schema files. Format: `pattern:schema_path` |
| `only-changed` | `"false"` | Only validate files changed in the current pull request |

## Outputs

| Output | Description |
|--------|-------------|
| `files-validated` | Total number of files scanned |
| `files-failed` | Number of files that failed validation |
| `exit-code` | Exit code from validation (0=success, 1=validation errors, 2=runtime error) |

## Examples

<details>
<summary><b>Filtering</b></summary>

#### Exclude directories

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    exclude-dirs: "tests,vendor,node_modules"
```

#### Exclude file types

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    exclude-file-types: "json,xml"
```

#### Include only specific file types

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    file-types: "json,yaml"
```

#### Disable recursive scanning

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    depth: 0
```

#### Glob pattern matching

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    globbing: "true"
    search-paths: "**/*.json"
```

</details>

<details>
<summary><b>Reporters</b></summary>

#### JSON report

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    reporter: "json"
```

#### Multiple reporters with file output

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    reporter: "json:output.json,junit:results.xml,sarif:results.sarif"
```

#### Group by pass/fail

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    group-by: "pass-fail"
```

#### Quiet mode

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    quiet: "true"
```

</details>

<details>
<summary><b>Schema Validation</b></summary>

#### Automatic schema lookup with SchemaStore

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    schemastore: "true"
```

#### Local SchemaStore clone (air-gapped)

```yaml
- run: git clone --depth=1 https://github.com/SchemaStore/schemastore.git
- uses: Boeing/validate-configs-action@v2
  with:
    schemastore-path: "./schemastore"
```

#### Require schema declarations

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    require-schema: "true"
```

#### Disable schema validation

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    no-schema: "true"
```

#### Map schemas to files

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    schema-map: "**/package.json:schemas/package.schema.json,**/config.xml:schemas/config.xsd"
```

</details>

<details>
<summary><b>Advanced</b></summary>

#### Map file types with glob patterns

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    type-map: "**/inventory:ini,**/*.cfg:json"
```

#### Validate only changed files in a PR

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0
- uses: Boeing/validate-configs-action@v2
  with:
    only-changed: "true"
```

#### Using outputs

```yaml
- uses: Boeing/validate-configs-action@v2
  id: validate
  continue-on-error: true
- run: |
    echo "Files validated: ${{ steps.validate.outputs.files-validated }}"
    echo "Files failed: ${{ steps.validate.outputs.files-failed }}"
```

</details>
