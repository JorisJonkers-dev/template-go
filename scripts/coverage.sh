#!/bin/sh
# The Go coverage gates (workspace docs/testing-standards.md, section 6). `unit` runs the tests that
# need nothing outside the process and holds them to every statement, branch, condition and
# function; `integration` adds the tests behind the `integration` build tag and holds the merged
# run to every function. Go measures only statements, so branches and conditions come from gobco
# and functions from the profile. Totals land in <out>/coverage-totals*.json.
set -eu

bucket=${1:?usage: coverage.sh unit|integration [out]}
out=${2:-coverage}
mkdir -p "$out"

# main() is os.Exit around start, which the tests drive; its file is all it holds.
excluded='/cmd/[^/]+/main\.go:'

# The packages with source to measure, under the build tags given.
measured() {
  go list ${1:+-tags "$1"} -f '{{if .GoFiles}}{{.ImportPath}}{{end}}' ./...
}

joined() {
  tr '\n' ',' | sed 's/,$//'
}

# profile runs every test under the tags given and keeps the profile without the excluded files.
profile() {
  go test -race -count=1 ${1:+-tags "$1"} -covermode=atomic -coverpkg="$(measured "$1" | joined)" \
    -coverprofile="$out/$2.raw" ./...
  grep -Ev "$excluded" "$out/$2.raw" >"$out/$2.out"
}

functions_gate() {
  go tool cover -func="$1" | grep -v '^total:' >"$out/functions-$bucket.txt" || true
  uncovered=$(awk '$NF == "0.0%"' "$out/functions-$bucket.txt")
  functions=$(grep -c . "$out/functions-$bucket.txt")
  missed=$(printf '%s' "$uncovered" | grep -c . || true)
  echo "$((functions - missed)) $functions"
  if [ -n "$uncovered" ]; then
    echo "functions no test reaches:" >&2
    echo "$uncovered" >&2
    return 1
  fi
}

if [ "$bucket" = integration ]; then
  profile integration integration
  counted=$(functions_gate "$out/integration.out")
  printf '{"integration": {"functions": {"covered": %s, "total": %s}}}\n' "${counted% *}" "${counted#* }" \
    >"$out/coverage-totals-integration.json"
  cat "$out/coverage-totals-integration.json"
  exit 0
fi

profile "" unit
statements=$(go tool cover -func="$out/unit.out" | awk '/^total:/ { sub("%", "", $3); print $3 }')
counted=$(functions_gate "$out/unit.out") || exit 1
functions_run=${counted% *}
functions=${counted#* }

# Branch mode counts each branch taken; condition mode counts each operand of && and || both ways.
gobco_total() {
  mode=$1
  shift
  taken=0
  total=0
  module=$(go list -m)
  for package in $(measured); do
    report=$(go tool gobco "$@" ".${package#"$module"}" 2>&1) || true
    echo "$report" >&2
    counted=$(echo "$report" | sed -n "s/^$mode coverage: \([0-9]*\)\/\([0-9]*\)\$/\1 \2/p")
    if [ -z "$counted" ]; then
      echo "gobco reported no $mode coverage for $package" >&2
      return 1
    fi
    taken=$((taken + ${counted% *}))
    total=$((total + ${counted#* }))
  done
  echo "$taken $total"
}

counted=$(gobco_total Branch -branch) || exit 1
branches_taken=${counted% *}
branches=${counted#* }
counted=$(gobco_total Condition) || exit 1
conditions_taken=${counted% *}
conditions=${counted#* }

printf '{"statements": %s, "branches": {"covered": %s, "total": %s}, "conditions": {"covered": %s, "total": %s}, "functions": {"covered": %s, "total": %s}}\n' \
  "$statements" "$branches_taken" "$branches" "$conditions_taken" "$conditions" "$functions_run" "$functions" \
  >"$out/coverage-totals.json"
cat "$out/coverage-totals.json"

if [ "$statements" != "100.0" ] || [ "$branches_taken" != "$branches" ] ||
  [ "$conditions_taken" != "$conditions" ] || [ "$functions_run" != "$functions" ]; then
  echo "unit coverage must reach every statement, branch, condition and function" >&2
  exit 1
fi
