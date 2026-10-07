#!/bin/sh

set -eu

if [ "$#" -ne 5 ]; then
  echo "usage: $0 <version> <sha256> <commit> <date> <output>" >&2
  exit 2
fi

version=$1
sha256=$2
commit=$3
date=$4
output=$5

require_match() {
  value=$1
  pattern=$2
  name=$3

  if ! printf '%s\n' "$value" | grep -Eq "$pattern"; then
    echo "invalid $name: $value" >&2
    exit 2
  fi
}

require_match "$version" '^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$' version
require_match "$sha256" '^[0-9a-f]{64}$' sha256
require_match "$commit" '^[0-9a-f]{40}$' commit
require_match "$date" '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(Z|[+-][0-9]{2}:[0-9]{2})$' date

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
template="$script_dir/nuke.rb.tmpl"
output_dir=$(dirname -- "$output")
temporary="$output.tmp"

mkdir -p "$output_dir"
trap 'rm -f "$temporary"' EXIT HUP INT TERM

sed \
  -e "s/__VERSION__/$version/g" \
  -e "s/__SHA256__/$sha256/g" \
  -e "s/__COMMIT__/$commit/g" \
  -e "s/__DATE__/$date/g" \
  "$template" >"$temporary"

if grep -Eq '__[A-Z0-9_]+__' "$temporary"; then
  echo "formula template contains unresolved placeholders" >&2
  exit 1
fi

mv "$temporary" "$output"
trap - EXIT HUP INT TERM
