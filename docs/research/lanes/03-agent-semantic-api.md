# Agent Semantic API Research

## Summary

As of 2026-05-12, `vibe-xpls` should treat AI agents as first-class users, but not by forcing agents to drive cursor-oriented LSP methods. Crossplane resources form a graph across package metadata, XRDs, Compositions, function pipelines, templates, schemas, and rendered resources. Agents need structured operations over that graph.

The best initial shape is an internal analyzer library with a read-only structured JSON CLI adapter for terminal, CI, and repository-scanning agents. Editor-embedded agents are a separate class because they may need unsaved buffer overlays and multi-file draft state. MCP and JSON-RPC should be evaluated after the analyzer, CLI contracts, overlay model, and security boundaries are stable.

## Sources

- Language Server Protocol 3.17: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
- Model Context Protocol specification: https://modelcontextprotocol.io/specification/
- MCP tools: https://modelcontextprotocol.io/specification/2025-11-25/server/tools
- MCP resources: https://modelcontextprotocol.io/specification/2025-11-25/server/resources
- MCP prompts: https://modelcontextprotocol.io/specification/2025-11-25/server/prompts
- Crossplane Compositions: https://docs.crossplane.io/latest/composition/compositions/
- Crossplane CLI command reference: https://docs.crossplane.io/latest/cli/command-reference/
- Crossplane XRDs: https://docs.crossplane.io/latest/composition/composite-resource-definitions/
- Sourcegraph precise code navigation: https://sourcegraph.com/docs/code-navigation/precise-code-navigation
- Sourcegraph search-based code navigation: https://sourcegraph.com/docs/code-search/code-navigation/search_based_code_navigation

## Operations

High-value operations:

- `list-compositions`: return Compositions, their composite type refs, pipeline steps, and source files.
- `explain-template`: explain the template at a file and position, including available root objects, helper functions, and schema-derived context.
- `find-schema`: resolve `apiVersion` and `kind` to schema source, field docs, and version metadata.
- `render`: run or simulate the render workflow and return structured resources, function results, and diagnostics.
- `validate-workspace`: return workspace-level diagnostics grouped by package, XRD, Composition, template, and schema source.
- `list-generated-resources`: describe rendered or statically inferred composed resources.
- `suggest-fix`: produce fix candidates from diagnostics and schema data, not from unstructured text scraping.

These operations should include stable object identifiers, source file paths, ranges when known, confidence levels, and clear limits when information is inferred.

## Agent Classes

Terminal and CI agents:

- Operate mostly on files that exist on disk.
- Can use one-shot CLI commands such as `list-compositions`, `find-schema`, `validate-workspace`, and trust-gated `render`.
- Should receive stable object identifiers and source ranges so their patches can be checked against current files.

Editor-embedded agents:

- May operate on unsaved buffers, multi-file drafts, and editor session state that is not yet present on disk.
- Should not be forced to call a file-only CLI and receive stale answers.
- Need an overlay-aware path through LSP session state, a future JSON-RPC session API, or explicit CLI overlay inputs before the product can claim full editor-agent support.

First scope may ship the file-backed JSON CLI for terminal and CI agents, but the design document must explicitly state that unsaved editor-agent overlays remain unresolved until one of those overlay paths is proven.

## Interface Options

Internal analyzer library:

- Best place for Crossplane semantic state.
- Should be transport-neutral and testable without an editor.
- Should power LSP, CLI, and later MCP/JSON-RPC adapters.

CLI with structured JSON:

- Best first agent-facing interface.
- Easy to test in fixtures and shell workflows.
- Good fit for commands such as `list-compositions`, `find-schema`, `validate-workspace`, and `render`.
- Should use explicit flags and return `ok`, `diagnostics`, `data`, and `errors` fields.
- Needs an overlay story before it can serve editor-embedded agents editing unsaved files. Possible options are stdin overlay bundles, a persistent JSON-RPC session, or delegation to the LSP document store.

JSON-RPC:

- Useful if long-running clients need persistent workspace state outside LSP.
- Useful if editor-side agents need unsaved buffers and multi-file draft state without scraping LSP cursor methods.
- Should reuse analyzer contracts rather than inventing separate semantics.

MCP:

- Good outer layer for AI clients because MCP has explicit tools, resources, and prompts.
- Should not be the core semantic model.
- Should be introduced after CLI contracts prove stable and safe.

LSP:

- Remains the editor protocol.
- Should not be scraped by agents as the primary semantic API.
- Cursor-oriented methods are useful for editor UX, but awkward for repository-level agent tasks.
- The LSP document store is still the most natural source of truth for editor-agent unsaved overlays unless a separate session API is introduced.

## Security Boundaries

Agent-facing commands must be conservative by default:

- Read-only by default.
- No cluster reads unless explicitly enabled.
- No Docker execution unless explicitly enabled.
- No package/schema downloads without a clear cache policy.
- Structured JSON output only; avoid asking agents to parse human logs.
- Source paths must be normalized to the workspace.
- Render and validation commands must report sanitized execution metadata such as tool path, version, timeout, exit code, and allowed feature gates. They must not report raw environment variables or secret-bearing configuration.
- First-scope `render` should be fixture-backed or simulated by default. Real `crossplane render`, Docker, function execution, package downloads, and cluster reads require explicit trusted-workspace and trusted-execution gates.
- Any future `suggest-fix` operation should return proposed edits, not apply them by default.

## Recommendation

Design `vibe-xpls` around a transport-neutral analyzer. Implement the first agent API as file-backed CLI commands returning structured JSON for terminal and CI agents. Evaluate MCP, JSON-RPC sessions, and overlay-aware CLI inputs after the analyzer, CLI, and security boundaries are proven.

The first agent API spike should implement `list-compositions`, `find-schema`, `validate-workspace`, and a fixture-backed or simulated `render`. Keep it read-only by default, and require explicit trust gates before any implementation invokes Docker, Crossplane functions, package downloads, or cluster reads.

Overlay acceptance criteria for the next design:

- Define whether first-scope editor agents read unsaved state from LSP document state, a JSON-RPC session, or explicit CLI overlay files/stdin.
- Prove multi-file draft state can be analyzed without mixing stale disk content and unsaved editor content.
- Preserve stable object identifiers across edits so agent plans and diagnostics can survive incremental changes.

## Confidence

High that agents need graph-shaped operations beyond LSP cursor methods.

High that an analyzer-first design is the right way to avoid divergent LSP, CLI, and MCP semantics.

Medium that MCP belongs in the early roadmap. It is likely useful, but should not precede a stable analyzer contract.

Medium-low that the first JSON CLI alone is enough for editor-embedded agents, because unsaved overlay behavior has not been proven.

## Evidence That Would Change This Recommendation

- Protocol tests show LSP workspace methods are sufficient for the agent workflows.
- Users do not want a separate CLI or agent API.
- MCP becomes the primary required integration target before the CLI contract is implemented.
- Security review finds that exposing render or schema operations to agents is too risky for the first scope.
- Editor-agent validation shows unsaved overlays are the dominant agent workflow, making an overlay-aware session API necessary before a file-backed CLI.
