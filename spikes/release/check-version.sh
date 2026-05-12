#!/bin/sh
set -eu

version="${1:-}"
case "$version" in
  v0.[0-9]*.[0-9]*|v0.[0-9]*.[0-9]*-*)
    exit 0
    ;;
  *)
    echo "release version must stay on v0.X.X before explicit pre-1.0 exit approval" >&2
    exit 1
    ;;
esac
