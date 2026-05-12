# Parser Decision For First Runnable Milestone

## Decision

Use `github.com/goccy/go-yaml` behind an internal parser facade for the first runnable product milestone.

Keep `go.yaml.in/yaml/v4` as a compatibility reference and fallback candidate, but do not expose either parser package outside `internal/analyzer/yaml.go`.

Pin parser dependencies for repeatable implementation:

- `github.com/goccy/go-yaml v1.19.2`
- `go.yaml.in/yaml/v4 v4.0.0-rc.4`

## Reasoning

The approved design requires source-aware YAML structure, mixed YAML/template masking, malformed-input resilience, hover/completion over stable YAML paths, and clear position conversion. `docs/research/lanes/05-yaml-template-parsing.md` identifies `goccy/go-yaml` as the strongest primary candidate because it exposes parser, lexer, AST, comment, and source-aware helpers.

The milestone still uses a parser facade because parser choice is a risk. Analyzer code consumes project-local types instead of parser-specific AST nodes.

## Parser Facade Contract

The parser facade returns:

- Parsed YAML documents with stable key paths.
- Diagnostics mapped to byte offsets in raw source.
- A flag for whether a path is eligible for schema-aware hover and completion.
- Best-effort results when input is malformed.

The facade must not:

- Execute templates.
- Invoke Crossplane CLI.
- Invoke Docker.
- Read the network.
- Read a Kubernetes cluster.

## Acceptance

This decision is accepted when product tests prove:

- Plain YAML parsing.
- Malformed YAML diagnostics.
- Mixed YAML/template masking.
- Stable-path eligibility.
- Source positions in raw byte offsets.
- No parser-specific types leak outside the analyzer package.
