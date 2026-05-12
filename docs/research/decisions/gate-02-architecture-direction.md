## Decision.

The architecture is analyzer-first: a Go-native static semantic core owns Crossplane workspace understanding, schema lookup, source mapping, diagnostics, hovers, completions, and command planning. The LSP layer is a thin adapter around that core. Mixed YAML/template files use raw, masked, and rendered views with explicit source maps. Crossplane CLI render and validate are optional proof paths for explicit, on-save, or manual commands, not the hot path for interactive diagnostics.

## Evidence.

- `docs/research/lanes/04-lsp-framework.md` recommends a thin LSP adapter around an internal analyzer core and says semantics should stay out of the transport layer.
- `docs/research/spikes/01-lsp-harness.md` proves document sync, diagnostics, hover, completion, shutdown, and stdio JSON-RPC framing can be exercised with deterministic handlers that delegate future semantics elsewhere.
- `docs/research/lanes/05-yaml-template-parsing.md` recommends a dual-view analyzer with raw file text, masked YAML/template structure, and rendered output only for later fixture validation.
- `docs/research/spikes/03-yaml-template-mapping.md` proves same-length masking can preserve byte offsets, line, and column positions while separating YAML diagnostics from template diagnostics in mixed YAML/template files.
- `docs/research/lanes/07-schema-workspace-indexing.md` recommends local-first indexing of built-in Crossplane APIs, workspace XRDs, Compositions, provider CRDs, package metadata, and user schema directories.
- `docs/research/spikes/04-schema-index.md` proves a local Go index can resolve XRDs, Compositions, provider CRDs, package metadata, and OpenAPI-backed field documentation without network access.
- `docs/research/spikes/05-render-validate.md` shows warm render and validate are useful but environment-sensitive; render crossed Docker permission boundaries, validate cache misses could hit registry and credential-helper behavior, and neither produced complete source-span mappings.

## Alternatives Considered.

- LSP-framework-first architecture: can speed editor plumbing, but it risks coupling Crossplane semantics to protocol handlers and making CLI or agent reuse harder.
- Render-first architecture: uses authoritative Crossplane execution, but it is too slow and environment-dependent for per-keystroke diagnostics and does not solve source mapping by itself.
- YAML-parser-only architecture: handles syntax and schema basics, but it cannot model template spans, function pipelines, XRD-to-Composition relationships, or rendered truth.
- Non-Go language server stack: mature options exist, but they add runtime and dependency cost without improving fit for a Go-native analyzer.

## Risks.

- The static analyzer can diverge from Crossplane runtime behavior if render/validate proof paths are not used in fixtures and explicit commands.
- Source maps across raw, masked, and rendered views can become complex once templates emit multiple resources or move fields across documents.
- A thin LSP adapter still needs careful UTF-8, UTF-16, byte, and rune conversion at protocol boundaries.
- Local schema indexing must handle large provider CRD sets, conflicting schemas, stale package metadata, and missing dependencies without blocking editing.

## What Would Change This Decision.

- Crossplane exposes a stable local semantic API that provides fast diagnostics, schema lookup, render-aware source spans, and workspace graph facts without Docker, cluster, or network dependencies.
- A production LSP framework proves it can host the analyzer with less risk while keeping semantics fully transport-independent.
- Real fixtures show source-mapped static analysis cannot provide useful diagnostics before render.
- Render or validate becomes fast, hermetic, source-mapped, and permission-safe enough for interactive editor use.
