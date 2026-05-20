## Decision.

GO for the next brainstorming/design phase. The current research base is strong enough to write the real Crossplane LSP brainstorming document and turn the spike findings into product requirements, architecture boundaries, and validation plans.

NO-GO for product implementation. The project should not yet start building the production analyzer, LSP server, Zed integration, schema index, render integration, or agent API as product code.

## Evidence.

- `docs/research/spikes/01-lsp-harness.md` proves a small Go stdio LSP loop can handle initialize, document sync, diagnostics, hover, completion, shutdown, and subprocess `Content-Length` framing, but it explicitly does not select the production LSP framework.
- The `crossplane-yaml` extension launches `vibe-xpls serve` for `Crossplane YAML` files and delegates package/no-root detection to the analyzer.
- Manual Zed UI validation remains required for startup logs, diagnostics, hover, completion, stale diagnostic clearing, and common repository shapes.
- `docs/research/spikes/03-yaml-template-mapping.md` proves same-length masking can preserve source coordinates across mixed YAML and Go-template files, while noting that production still needs explicit LSP encoding conversion and a real YAML parser.
- `docs/research/spikes/04-schema-index.md` proves local XRD, Composition, provider CRD, and package metadata lookup can support kind lookup and field documentation, but only through a narrow fixture parser without watchers, precedence, cache invalidation, or package discovery.
- `docs/research/spikes/05-render-validate.md` proves `crossplane render` and `crossplane beta validate` are valuable proof commands, but render crosses Docker permissions and validate can depend on cache, registry, credential-helper, or kubeconfig state; they are not hot-path LSP operations.
- `docs/research/spikes/06-agent-api.md` proves a read-only JSON command envelope for agents, including explicit security metadata, but it remains fixture-backed and does not scan real workspaces, execute Crossplane, download schemas, or read clusters.
- `docs/research/spikes/07-kubernetes-tooling.md` shows generic YAML and Kubernetes tools solve important infrastructure problems, while Crossplane still needs a Go-native semantic layer for XRD-to-Composition relationships, function pipelines, template source maps, render truth, provider discovery, and agent APIs.
- The Kubernetes reuse evidence is still provisional for implementation because embeddable Kubernetes/OpenAPI libraries have not been compared against the custom schema-index spike on the same fixture set.
- The agent API evidence proves a disk-backed JSON CLI shape, not editor-side agent support for unsaved buffers or multi-file drafts.
- `docs/research/lanes/11-security-reliability.md` identifies the trust model that product design must settle before implementation: untrusted workspace defaults, explicit gates for Docker, downloads, cluster reads, writes, and agent-triggered execution, plus sanitized diagnostics.
- `docs/research/lanes/10-release-phase-gates.md` requires each later implementation phase to define runnable evidence, tests, release checks, and v0 guardrails before merge or release.

## Alternatives Considered.

- Start product implementation now. Rejected because the research has deliberately narrow spikes and leaves unresolved production choices around parser selection, Zed UI behavior, schema/index freshness, performance, and trust boundaries.
- Continue open-ended research before writing a design artifact. Rejected because the spikes already identify the major architecture boundaries and blockers; the next useful step is a focused brainstorming/design document, not more disconnected probes.
- Treat generic YAML LS, Kubeconform, or `kubectl explain` as the product core. Rejected because `docs/research/spikes/07-kubernetes-tooling.md` shows those tools do not model Crossplane-specific relationships, function pipelines, render truth, or agent-facing operations.
- Treat `crossplane render` as the primary analyzer. Rejected for hot paths because `docs/research/spikes/05-render-validate.md` shows Docker, cache, registry, credential-helper, kubeconfig, latency, and source-mapping limits.

## Risks.

- The next design phase could overfit to fixture-backed evidence unless it calls out which results are proven only by spikes.
- Manual Zed validation may expose client behavior that changes the LSP adapter or extension contract.
- Parser choice can materially affect diagnostics, source maps, comments, anchors, tags, duplicate keys, performance, and LSP position handling.
- Real schema/index workloads may reveal latency, memory, freshness, conflict, or package-discovery issues not visible in fixture tests.
- Embeddable Kubernetes/OpenAPI libraries may change how much custom schema indexing should be built.
- Editor-embedded agents may need unsaved-overlay state that the file-backed CLI spike does not cover.
- Zed/analyzer behavior may still fail in nested packages, multi-package repositories, no-root-manifest repositories, or without user `file_types` mappings.
- Trust gates are central to the product boundary and still need concrete UX and policy decisions across CLI, Zed, and future agent adapters.

## What Would Change This Decision.

- A manual Zed run validates startup, diagnostics, hover, completion, missing-binary behavior, stale diagnostic clearing, and package/no-root analyzer behavior through `crossplane-yaml`.
- A production YAML/template parser choice is validated against mixed Crossplane fixtures, malformed input, UTF-8/UTF-16 position mapping, and source-map requirements.
- A real workspace schema/index prototype proves acceptable performance, freshness, conflict handling, and provider package discovery.
- A Kubernetes/OpenAPI library spike clarifies which upstream packages should be embedded versus which schema-index code remains custom.
- The agent API design specifies unsaved overlays and multi-file draft state for editor-side agents.
- The Zed manual gate proves launch and attach behavior across root packages, nested packages, multi-package workspaces, and `file_types` setups.
- The trust model is specified for Docker render, package/schema downloads, cluster reads, kubeconfig `exec` plugins, executable/image identity, write-producing tools, and agent-triggered operations.
- Broader user validation confirms the intended Crossplane workflows, editor commands, agent APIs, and release gates solve real authoring problems.
