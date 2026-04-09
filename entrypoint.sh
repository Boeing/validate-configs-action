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
ANNOTATIONS_JSON=$(mktemp)
CMD="$CMD --reporter=json:$ANNOTATIONS_JSON"

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
      line = ""; col = ""
      if (match(error_msg, /line [0-9]+/)) {
        line = substr(error_msg, RSTART+5, RLENGTH-5)
      }
      if (match(error_msg, /column [0-9]+/)) {
        col = substr(error_msg, RSTART+7, RLENGTH-7)
      }
      annotation = "::error file=" path
      if (line != "") annotation = annotation ",line=" line
      else annotation = annotation ",line=1"
      if (col != "") annotation = annotation ",col=" col
      annotation = annotation "::" error_msg
      print annotation
    }
  ' "$ANNOTATIONS_JSON"
  rm -f "$ANNOTATIONS_JSON"
fi

exit $EXIT_CODE
