#!/bin/sh

# Entrypoint wrapper for the validator executable
# Maps GitHub Action inputs to CLI arguments and CFV_* environment variables
# The validator natively reads CFV_* env vars, so we export simple flags
# as env vars and only build CLI args for search paths and repeatable flags.

set -e

SEARCH_PATHS=$1
EXCLUDE_DIRS=$2
EXCLUDE_FILE_TYPES=$3
FILE_TYPES=$4
DEPTH=$5
REPORTER=$6
GROUP_BY=$7
QUIET=$8
GLOBBING=$9
REQUIRE_SCHEMA=${10}
NO_SCHEMA=${11}
SCHEMASTORE=${12}
TYPE_MAP=${13}
SCHEMA_MAP=${14}

# Export simple flags as CFV_* env vars (validator reads these natively)
[ -n "$EXCLUDE_DIRS" ]       && export CFV_EXCLUDE_DIRS="$EXCLUDE_DIRS"
[ -n "$EXCLUDE_FILE_TYPES" ] && export CFV_EXCLUDE_FILE_TYPES="$EXCLUDE_FILE_TYPES"
[ -n "$FILE_TYPES" ]         && export CFV_FILE_TYPES="$FILE_TYPES"
[ -n "$DEPTH" ]              && export CFV_DEPTH="$DEPTH"
[ -n "$GROUP_BY" ]           && export CFV_GROUPBY="$GROUP_BY"
[ "$QUIET" = "true" ]        && export CFV_QUIET="true"
[ "$GLOBBING" = "true" ]     && export CFV_GLOBBING="true"
[ "$REQUIRE_SCHEMA" = "true" ] && export CFV_REQUIRE_SCHEMA="true"
[ "$NO_SCHEMA" = "true" ]   && export CFV_NO_SCHEMA="true"
[ -n "$SCHEMASTORE" ]        && export CFV_SCHEMASTORE="$SCHEMASTORE"

# Build CLI args for repeatable flags
CMD=validator

# Reporter supports multiple values and format:path syntax
if [ -n "$REPORTER" ]; then
  for r in $(echo "$REPORTER" | tr ',' ' '); do
    CMD="$CMD --reporter=$r"
  done
fi

# Always add a JSON reporter to a temp file for annotation parsing
# unless the user already specified a json reporter with a file output
ANNOTATIONS_JSON=$(mktemp)
HAS_JSON_FILE=""
if [ -n "$REPORTER" ]; then
  for r in $(echo "$REPORTER" | tr ',' ' '); do
    case "$r" in
      json:*) HAS_JSON_FILE=$(echo "$r" | cut -d: -f2) ;;
    esac
  done
fi
if [ -n "$HAS_JSON_FILE" ]; then
  ANNOTATIONS_JSON="$HAS_JSON_FILE"
else
  CMD="$CMD --reporter=json:$ANNOTATIONS_JSON"
fi

# type-map supports multiple comma-separated mappings
if [ -n "$TYPE_MAP" ]; then
  for t in $(echo "$TYPE_MAP" | tr ',' ' '); do
    CMD="$CMD --type-map=$t"
  done
fi

# schema-map supports multiple comma-separated mappings
if [ -n "$SCHEMA_MAP" ]; then
  for s in $(echo "$SCHEMA_MAP" | tr ',' ' '); do
    CMD="$CMD --schema-map=$s"
  done
fi

# add search paths
CMD="$CMD $SEARCH_PATHS"

# Run validator, capture exit code
set +e
( ${CMD} )
EXIT_CODE=$?
set -e

# Parse JSON output and emit GitHub Actions annotations for failures
if [ -f "$ANNOTATIONS_JSON" ]; then
  jq -r '
    .files[] | select(.status == "failed") |
    (.path | sub("^/github/workspace/"; "")) as $path |
    [.errors[] |
      (if startswith("schema: ") then {title: "Schema Error", msg: .[8:]}
       elif startswith("syntax: ") then {title: "Syntax Error", msg: .[8:]}
       else {title: "Validation Error", msg: .}
       end) |
      (.msg | capture("line (?<l>[0-9]+)") // {l: null}) as $lm |
      (.msg | capture("column (?<c>[0-9]+)") // {c: null}) as $cm |
      (.msg | capture("\\(string\\):(?<l>[0-9]+):") // {l: null}) as $xm |
      {title, msg, line: ($lm.l // $xm.l // "1"), col: $cm.c}
    ] |
    group_by([.line, .col, .title])[] |
    (.[0].title) as $title |
    (.[0].line) as $line |
    (.[0].col) as $col |
    [.[].msg] as $msgs |
    (if ($msgs | length) > 1 then
      "\($msgs | length) \($title | ascii_downcase)s found:\n" + ($msgs | map("\u2022 " + .) | join("\n"))
     else $msgs[0]
     end) as $body |
    "::error file=\($path),title=\($title),line=\($line)" +
    (if $col then ",col=\($col)" else "" end) +
    "::\($body | gsub("\n"; "%0A"))"
  ' "$ANNOTATIONS_JSON"
  if [ -z "$HAS_JSON_FILE" ]; then
    rm -f "$ANNOTATIONS_JSON"
  fi
fi

exit $EXIT_CODE
