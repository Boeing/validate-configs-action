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
  awk '
    /"path":/ {
      gsub(/.*"path": *"/, ""); gsub(/".*/, "");
      sub(/^\/github\/workspace\//, "");
      path = $0
      failed = 0
    }
    /"status": *"failed"/ { failed = 1 }
    /"status": *"passed"/ { failed = 0 }
    /"errors":/ && failed { in_errors = 1; next }
    in_errors && /\]/ { in_errors = 0; next }
    in_errors && /"/ {
      gsub(/^ *"/, ""); gsub(/"[,]?$/, "");
      error_msg = $0
      title = "Validation Error"
      clean_msg = error_msg
      if (match(error_msg, /^schema: /)) {
        title = "Schema Error"
        clean_msg = substr(error_msg, 9)
      } else if (match(error_msg, /^syntax: /)) {
        title = "Syntax Error"
        clean_msg = substr(error_msg, 9)
      }
      line = ""; col = ""
      if (match(clean_msg, /line [0-9]+/)) {
        line = substr(clean_msg, RSTART+5, RLENGTH-5)
      }
      if (match(clean_msg, /\(string\):[0-9]+:/) && line == "") {
        tmp = clean_msg
        sub(/.*\(string\):/, "", tmp)
        sub(/:.*/, "", tmp)
        line = tmp
      }
      if (match(clean_msg, /column [0-9]+/)) {
        col = substr(clean_msg, RSTART+7, RLENGTH-7)
      }
      annotation = "::error file=" path ",title=" title
      if (line != "") annotation = annotation ",line=" line
      else annotation = annotation ",line=1"
      if (col != "") annotation = annotation ",col=" col
      annotation = annotation "::" clean_msg
      print annotation
    }
  ' "$ANNOTATIONS_JSON"
  if [ -z "$HAS_JSON_FILE" ]; then
    rm -f "$ANNOTATIONS_JSON"
  fi
fi

exit $EXIT_CODE
