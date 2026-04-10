# Validate Configs Github Action

<p>
  <a href="https://scorecard.dev/viewer/?uri=github.com/Boeing/validate-configs-action">
    <img src="https://api.scorecard.dev/projects/github.com/Boeing/validate-configs-action/badge" alt="OpenSSF Scorecard">
  </a>
  <a href="https://opensource.org/licenses/Apache-2.0">
    <img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="Apache 2 License">
  </a>
</p>

:octocat: Github Action to validate your config files using the [config-file-validator](https://github.com/Boeing/config-file-validator). The config-file-validator will recursively scan the provided search path for the following configuration file types:

* Apple PList XML
* CSV
* EDITORCONFIG
* ENV
* HCL
* HOCON
* INI
* JSONC
* JSON
* Properties
* SARIF
* TOML
* TOON
* XML
* YAML

Each file will get validated for the correct syntax and the results collected into a report showing the path of the file and if it is invalid or valid. If the file is invalid an error will be displayed along with the line number and column where the error ocurred. By default the `$GITHUB_WORKDIR` is scanned.

Files that declare a schema (JSON Schema for JSON/YAML/TOML/TOON, XSD for XML) are automatically validated against it.

## PR inline annotations

Validation errors automatically appear as inline annotations on pull request diffs. When a config file fails validation, the action emits GitHub Actions workflow commands that annotate the exact file and line where the error occurred. For config types that do not support line numbers in the error output, an annotation will be added at line 1.

## Inputs

| Input              | Required | Default Value | Description |
| ------------------ | -------- | ------------- | ----------- |
| search-paths       | false    | `"."`         | The path that will be recursively searched for configuration files |
| exclude-dirs       | false    | `""`          | A comma-separated list of subdirectories to exclude from validation |
| exclude-file-types | false    | `""`          | A comma-separated list of file extensions to exclude |
| file-types         | false    | `""`          | A comma-separated list of file types to validate. Cannot be used with exclude-file-types |
| depth              | false    | `""`          | An integer value limiting the depth of recursion for the search paths. Setting depth to 0 disables recursion |
| reporter           | false    | `"standard"`  | Comma-separated report formats with optional output paths. Format: `type:path`. Options are `standard`, `json`, `junit`, and `sarif` |
| group-by           | false    | `""`          | Group output by `filetype`, `directory`, or `pass-fail` |
| quiet              | false    | `"false"`     | If set to `true`, suppresses all output to stdout |
| globbing           | false    | `"false"`     | If set to `true`, enables glob pattern matching for search paths |
| require-schema     | false    | `"false"`     | If set to `true`, fail validation for files that support schema validation but do not declare a schema |
| no-schema          | false    | `"false"`     | If set to `true`, disable all schema validation (syntax-only). Cannot be used with `require-schema`, `schema-map`, or `schemastore` |
| schemastore        | false    | `"false"`     | If set to `true`, enables automatic schema lookup using the embedded SchemaStore catalog with remote fetching |
| schemastore-path   | false    | `""`          | Path to a local SchemaStore clone for automatic schema lookup. For air-gapped environments. Implies `schemastore` |
| type-map           | false    | `""`          | Comma-separated glob pattern to file type mappings. Format: `pattern:type` |
| schema-map         | false    | `""`          | Comma-separated glob pattern to schema file mappings. Format: `pattern:schema_path` |


## Outputs

N/A

## Example usage

### Standard Run

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
```

### Custom search path

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            search-paths: ./project/configs
```

### Multiple search paths

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            search-paths: ./project/configs ./project/devops
```

### Exclude a directory

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            exclude-dirs: "tests,vendor"
```

### Exclude file type

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            exclude-file-types: "json,xml"
```

### Include only specific file types

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            file-types: "json,yaml"
```

### Disable recursive scanning

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            depth: 0
```

### JSON Report

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            reporter: "json"
```

### Multiple reporters with output files

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            reporter: "json:output.json,junit:results.xml"
```

### Group By Pass/Fail

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            group-by: "pass-fail"
```

### Quiet mode

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            quiet: "true"
```

### Glob pattern matching

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            globbing: "true"
            search-paths: "**/*.json"
```

### Require schema declarations

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            require-schema: "true"
```

### Disable schema validation

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            no-schema: "true"
```

### Automatic schema lookup with SchemaStore

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            schemastore: "true"
```

### Automatic schema lookup with local SchemaStore clone

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: git clone --depth=1 https://github.com/SchemaStore/schemastore.git
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            schemastore-path: "./schemastore"
```

### Map file types with glob patterns

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            type-map: "**/inventory:ini,**/*.cfg:json"
```

### Map schemas to files

```yml
jobs:
  validate-config-files:
    runs-on: ubuntu-latest
    steps:
      - uses: Boeing/validate-configs-action@v2.0.0
        with:
            schema-map: "**/package.json:schemas/package.schema.json,**/config.xml:schemas/config.xsd"
```