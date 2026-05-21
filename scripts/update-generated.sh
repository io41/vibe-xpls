#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

build_dir="${BUILD_DIR:-dist/local}"

echo "==> Regenerating built-in Crossplane schema bundle"
go run ./cmd/vibe-xpls-schema-gen \
  --config internal/analyzer/schemadata/config.json \
  --out internal/analyzer/schemadata

echo "==> Running schema generator and bundle checks"
go test ./internal/schemagen ./internal/analyzer

echo "==> Running full test suite"
go test ./...

echo "==> Building local CLIs into ${build_dir}"
mkdir -p "${build_dir}"
go build -o "${build_dir}/vibe-xpls" ./cmd/vibe-xpls
go build -o "${build_dir}/vibe-xpls-schema-gen" ./cmd/vibe-xpls-schema-gen

echo "==> Generated artifacts, tests, and local builds are up to date"
