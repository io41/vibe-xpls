# Security And Reliability Research

## Summary

`vibe-xpls` will sit on a sensitive boundary: it reads untrusted workspace files, may execute external CLIs, may trigger Docker-backed Crossplane functions, may download schemas or packages, may read Kubernetes clusters, and may expose agent-facing tools. The default posture must be local, read-only, non-executing, and deterministic.

The first implementation should treat render, downloads, cluster discovery, and write-producing agent tools as privileged operations behind explicit trust gates. Diagnostics should degrade gracefully when data is untrusted or unavailable rather than silently executing tools.

## Sources

- Crossplane CLI command reference: https://docs.crossplane.io/master/cli/command-reference/
- Crossplane Compositions: https://docs.crossplane.io/latest/composition/compositions/
- function-go-templating: https://github.com/crossplane-contrib/function-go-templating
- MCP security best practices: https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices
- MCP authorization: https://modelcontextprotocol.io/docs/tutorials/security/authorization
- VS Code Workspace Trust: https://code.visualstudio.com/api/extension-guides/workspace-trust
- Docker Engine security: https://docs.docker.com/engine/security/
- Docker Compose trust model: https://docs.docker.com/compose/trust-model/
- OWASP Path Traversal: https://owasp.org/www-community/attacks/Path_Traversal
- Kubernetes CRDs: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/

## Risk Register

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Docker execution from `crossplane render` | Runs package-defined function containers and may pull images | Off by default; require trusted workspace, explicit command, timeout, and sanitized logs |
| Function development runtime | Can connect to local endpoints controlled by the workspace | Require explicit trusted-execution gate and disclose target host/port |
| Package/schema downloads | Remote content can poison cache or change results | Off by default; require content-addressed storage or verified checksums before cross-workspace reuse; keep tag-only downloads untrusted |
| Live-cluster discovery | Reads kubeconfig-selected cluster state and may trigger kubeconfig `exec` auth plugins before any intended read | Off by default; no kubeconfig-touching CLI in untrusted mode; require explicit context/path display, env scrubbing, timeout, and separate approval for `exec` auth |
| Untrusted templates | Go templates can be malformed, expensive, or emit misleading YAML | Parse without executing template logic; execute only through trusted render path |
| Path traversal in FileSystem templates | Template paths may escape workspace through `../`, symlinks, or time-of-check/time-of-use races | Resolve workspace and target realpaths, reject escapes after symlink evaluation, prefer no-follow or fd-relative opens, and re-check identity after open |
| Agent tool permissions | Agent may invoke render, downloads, or cluster reads indirectly | Read-only default; separate capabilities for discovery, execution, network, cluster, and writes |
| Raw environment leakage | Diagnostics could expose credentials | Never report raw environment; report sanitized tool path/version/exit code only |
| Cache poisoning | Stale or malicious schemas produce wrong completions/diagnostics | Hash cache entries, record source, separate trusted and untrusted caches, require immutable identity before reuse, add refresh command |
| LSP crashes | Bad YAML/template input can terminate editor intelligence | Panic recovery around document analysis, bounded diagnostics, and test malformed fixtures |
| Diagnostic noise | Incomplete edits can generate stale or unactionable errors | Debounce, classify confidence, clear diagnostics on close, suppress lower-confidence errors while typing, and fence async results by document/workspace generation |
| Large workspaces and provider CRDs | Indexing can block editor startup | Incremental indexing, size limits, cancellation, progress reporting, and per-source disable switches |

## Required Mitigations

- Default `validate-workspace`, `list-compositions`, `find-schema`, and template explanation to read-only static analysis.
- Make real render execution opt-in. The first agent API render should be simulated or fixture-backed unless the user explicitly enables trusted execution.
- Never pass unchecked workspace paths to external commands. Resolve paths against the workspace root, reject escapes after `EvalSymlinks`, define symlink policy, prefer no-follow or fd-relative opens where possible, and re-check target identity after opening.
- Use timeouts and cancellation for all parsing, indexing, external commands, and cluster calls.
- Attach a monotonic document/workspace generation to every async parse, index, render, or validation task, and drop results whose generation no longer matches current state.
- Store schema and package caches with provenance: URL, registry, digest or version, retrieval time, and trust level. Do not promote tag-only downloads into a trusted cross-workspace cache without a digest, checksum, or signature. Separate negative-cache entries from positive verified artifacts.
- Sanitize all diagnostics and structured JSON outputs for secrets, environment variables, kubeconfig paths, and registry credentials.
- Treat cluster discovery as an execution boundary, not just a read boundary. In untrusted mode, do not call CLIs that may load kubeconfig. In trusted mode, show kubeconfig path and context, detect or require approval for `exec` auth plugins, scrub inherited environment, and isolate subprocess execution with hard timeouts.
- Bind trust grants to immutable subjects: workspace realpath, operation class, canonical executable path or image reference, content hash or digest when available, and configuration source. Invalidate grants when the executable path, symlink target, mtime+size, image digest, or config source changes.
- Treat MCP as an outer adapter with least-privilege scopes, not as the core authorization model.
- Avoid broad wildcard permissions in agent tools. Split low-risk discovery tools from execution and write tools.

## Review Gates

- Security review before enabling Docker-backed `crossplane render`.
- Security review before enabling package or schema downloads by default.
- Security review before live-cluster discovery leaves experimental status.
- Security review before any MCP adapter exposes tools beyond read-only discovery.
- Reliability review before indexing large provider CRD sets by default.
- Regression fixtures for malformed YAML, unterminated templates, template path traversal, symlink escapes, huge documents, stale diagnostics, async generation fencing, and external command timeouts.

## Recommendation

Adopt a trust-gated execution model from the start. The analyzer should be safe to run on untrusted repositories: no Docker, no network, no cluster reads, no writes, and no raw environment reporting unless explicitly enabled by a trusted workspace policy.

Implement render and cluster-backed validation as explicit commands with clear status, sanitized metadata, and timeouts. Add MCP only after the analyzer and CLI can express low-risk discovery separately from privileged execution.

## Confidence

High that Docker, downloads, cluster reads, and agent tool permissions are the main security boundaries.

High that static analysis must remain useful without those privileged features.

Medium on the exact trust UX because Zed, CLI, and future MCP clients will expose trust and approvals differently.

## Evidence That Would Change This Recommendation

- Zed or another first editor supplies a strong built-in workspace trust and command approval model that `vibe-xpls` can reuse directly.
- Crossplane CLI adds a no-Docker, no-network render mode that is safe enough for more frequent validation.
- User research shows teams require cluster discovery by default, and they accept kubeconfig read risk.
- Security review finds that even simulated render outputs are likely to mislead agents unless clearly marked as non-authoritative.
