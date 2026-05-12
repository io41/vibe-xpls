#!/bin/sh
set -eu

version="${1:-}"
version_re='^v0\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$'

if printf '%s\n' "$version" | grep -Eq "$version_re"; then
  exit 0
fi

echo "release version must stay on v0.X.X before explicit pre-1.0 exit approval" >&2
exit 1
