# Validate Configs Github Action

<p>
  <a href="https://scorecard.dev/viewer/?uri=github.com/Boeing/validate-configs-action">
    <img src="https://api.scorecard.dev/projects/github.com/Boeing/validate-configs-action/badge" alt="OpenSSF Scorecard">
  </a>
  <a href="https://opensource.org/licenses/Apache-2.0">
    <img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="Apache 2 License">
  </a>
</p>

Validate every config file in your repo — JSON, YAML, TOML, XML, INI, HCL, and [more](#inputs). If something's wrong, you'll see the error right on the PR diff, exactly where it broke.

Files with a schema declaration (JSON Schema, XSD) get validated against it automatically. You can optionally add [SchemaStore](#schemastore) to get schema validation for hundreds of common files like `package.json`, `tsconfig.json`, and GitHub Actions workflows with no additional configuration.

## Quick Start

```yaml
- uses: Boeing/validate-configs-action@v2
```

## Usage

```yaml
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: Boeing/validate-configs-action@v2
```

### Only changed files

For large repos, you probably don't want to validate everything on every PR. This only checks files that were actually changed:

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0
- uses: Boeing/validate-configs-action@v2
  with:
    only-changed: "true"
```

`fetch-depth: 0` is required so git has the history to determine which files changed.

### Exclude directories

Skip directories you don't care about — vendored deps, test fixtures, generated files:

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    exclude-dirs: "vendor,testdata,node_modules"
```

### SchemaStore

[SchemaStore](https://www.schemastore.org/) is a community catalog of JSON Schemas for common config files. Turn it on and files like `package.json`, `tsconfig.json`, and GitHub Actions workflows get validated against their schema automatically — no `$schema` declarations needed in your files:

```yaml
- uses: Boeing/validate-configs-action@v2
  with:
    schemastore: "true"
```

### Outputs

```yaml
- uses: Boeing/validate-configs-action@v2
  id: validate
  continue-on-error: true
- run: |
    echo "Files validated: ${{ steps.validate.outputs.files-validated }}"
    echo "Files failed: ${{ steps.validate.outputs.files-failed }}"
```

## Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `search-paths` | `"."` | Space-separated list of directories or files to scan |
| `exclude-dirs` | `""` | Comma-separated list of subdirectories to exclude |
| `exclude-file-types` | `""` | Comma-separated list of file extensions to exclude |
| `file-types` | `""` | Comma-separated list of file types to validate. Cannot be used with `exclude-file-types` |
| `depth` | `""` | Recursion depth limit. `0` disables recursion |
| `reporter` | `"standard"` | Report format(s). Options: `standard`, `json`, `junit`, `sarif`. Supports `type:path` for file output. Multiple reporters can be comma-separated. If you only specify file reporters, add `standard` to keep console output |
| `group-by` | `""` | Group output by `filetype`, `directory`, or `pass-fail` |
| `quiet` | `"false"` | Suppress all output to stdout |
| `globbing` | `"false"` | Enable glob pattern matching for search paths. Cannot be used with `exclude-dirs`, `exclude-file-types`, or `file-types` |
| `require-schema` | `"false"` | Fail files that support schema validation but don't declare a schema. Cannot be used with `no-schema` |
| `no-schema` | `"false"` | Disable all schema validation (syntax-only). Cannot be used with `require-schema`, `schema-map`, or `schemastore` |
| `schemastore` | `"false"` | Enable automatic schema lookup using the embedded [SchemaStore](https://www.schemastore.org/) catalog |
| `schemastore-path` | `""` | Path to a local SchemaStore clone. For air-gapped environments. Implies `schemastore` |
| `type-map` | `""` | Map glob patterns to file types. Format: `pattern:type`. Valid types: `csv`, `editorconfig`, `env`, `hcl`, `hocon`, `ini`, `json`, `jsonc`, `plist`, `properties`, `sarif`, `toml`, `toon`, `xml`, `yaml` |
| `schema-map` | `""` | Map glob patterns to schema files. Format: `pattern:schema_path`. Use JSON Schema (`.json`) for JSON/JSONC/YAML/TOML/TOON, XSD (`.xsd`) for XML. Paths are relative to the repo root |
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

#### Combined options

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0
- uses: Boeing/validate-configs-action@v2
  id: validate
  with:
    only-changed: "true"
    exclude-dirs: "vendor,generated,testdata"
    schema-map: "**/app-config.json:schemas/app.schema.json,**/deploy.xml:schemas/deploy.xsd"
    type-map: "**/inventory:ini,**/.env.*:env"
    reporter: "standard,junit:results.xml"
    schemastore: "true"
```

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
