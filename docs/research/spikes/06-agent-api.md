# Agent API Spike

## Summary

This spike implements a dependency-light Go CLI under `spikes/agent-api` that exposes the first read-only agent API shape as structured JSON commands. It proves that agent-facing operations can share one envelope with an `ok` boolean, command name, structured data, diagnostics, errors, and explicit security metadata.

The implementation is intentionally fixture-backed. It does not scan the real workspace, invoke Docker, invoke the Crossplane CLI, download schemas, or read a Kubernetes cluster. The goal is to validate command shape and trust boundaries before building the analyzer that will provide real data.

The spike is also file-backed by design. It does not model editor-side agents working against unsaved buffers or multi-file drafts. The next design must choose an overlay path for those agents, such as LSP document state, a persistent JSON-RPC session, or explicit CLI overlay input.

## Commands

- `list-compositions` returns one fixture Composition summary with a `compositeTypeRef`, source file, mode, and pipeline steps.
- `find-schema --api-version platform.example.org/v1alpha1 --kind XBucket` resolves a fixture XRD schema and returns field metadata with provenance.
- `validate-workspace` returns fixture validation status, checked categories, and explicit static-analysis limits.
- `render` returns a fixture-backed render result with one composed bucket resource and function results.

All commands are read-only. Unsupported commands and invalid arguments also return JSON with `ok: false`.

## JSON Contracts

Every response uses this top-level envelope:

```json
{
  "ok": true,
  "command": "render",
  "data": {},
  "diagnostics": [],
  "errors": [],
  "security": {
    "readOnly": true,
    "fixtureBacked": true,
    "networkAccess": false,
    "dockerInvoked": false,
    "crossplaneCliInvoked": false,
    "clusterAccess": false,
    "writesWorkspace": false,
    "trustMode": "untrusted-workspace-safe",
    "externalExecutionMode": "disabled"
  }
}
```

The implementation normalizes omitted `data`, `diagnostics`, and `errors` fields at the JSON boundary, so empty values serialize as `{}` or `[]` instead of `null`.

Command-specific data contracts:

- `list-compositions`: `data.workspace` plus `data.compositions[]` with `id`, `name`, `file`, `mode`, `compositeTypeRef`, and `pipeline[]`.
- `find-schema`: `data.query`, `data.found`, optional `data.schema`, and `data.matches` when a schema is not found.
- `validate-workspace`: `data.workspace`, `data.valid`, `data.checked[]`, and `data.limits[]`.
- `render`: `data.fixtureBacked`, `data.authoritative`, `data.inputs`, `data.resources[]`, `data.functionResults[]`, and `data.execution`.

The `render` contract deliberately separates command success from authority. `ok: true` means the command returned a valid structured response. `data.authoritative: false` means the result is a fixture and must not be treated as actual Crossplane execution.

## Commands Run

In `spikes/agent-api`:

```text
gofmt -w main.go main_test.go
go test ./...
```

Successful output:

```text
ok  	github.com/io41/vibe-xpls/spikes/agent-api	1.529s
ok  	github.com/io41/vibe-xpls/spikes/agent-api	(cached)
ok  	github.com/io41/vibe-xpls/spikes/agent-api	0.717s
```

At the worktree root:

```text
git diff --check
```

Successful output: no output, exit code 0.

## Security Boundaries

The spike enforces the first-scope security posture in the command contract:

- Read-only operations only.
- No Docker invocation.
- No Crossplane CLI invocation.
- No network access.
- No Kubernetes cluster access.
- No workspace writes.
- No raw environment, kubeconfig, registry credential, or secret-bearing metadata in output.
- Render results are fixture-backed and marked non-authoritative.

The command contract leaves room for future trust gates by reporting `trustMode` and `externalExecutionMode`, but this spike does not implement any trusted execution path.

## Decision Impact

The spike supports the research recommendation that agents should use structured repository-level operations rather than scraping cursor-oriented LSP methods. A simple JSON CLI is enough to express the first useful operations and to make safety properties visible to agents.

The next implementation should move fixture data behind a transport-neutral analyzer library, then keep this CLI as a thin adapter for disk-backed terminal and CI agents. MCP, JSON-RPC, or CLI overlay inputs can be evaluated later once the analyzer contracts, read-only command set, unsaved-buffer model, and trust-gated execution model are stable.
