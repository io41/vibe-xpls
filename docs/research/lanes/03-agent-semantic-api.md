# Agent Semantic API Research

## Summary

As of 2026-05-12, `vibe-xpls` should treat AI agents as first-class users, but not by forcing agents to drive cursor-oriented LSP methods. Crossplane resources form a graph across package metadata, XRDs, Compositions, function pipelines, templates, schemas, and rendered resources. Agents need structured operations over that graph.

The best initial shape is an internal analyzer library with a read-only structured JSON CLI adapter. MCP and JSON-RPC should be evaluated after the analyzer and CLI contracts are stable.

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

JSON-RPC:

- Useful if long-running clients need persistent workspace state outside LSP.
- Should reuse analyzer contracts rather than inventing separate semantics.

MCP:

- Good outer layer for AI clients because MCP has explicit tools, resources, and prompts.
- Should not be the core semantic model.
- Should be introduced after CLI contracts prove stable and safe.

LSP:

- Remains the editor protocol.
- Should not be scraped by agents as the primary semantic API.
- Cursor-oriented methods are useful for editor UX, but awkward for repository-level agent tasks.

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

Design `vibe-xpls` around a transport-neutral analyzer. Implement the first agent API as CLI commands returning structured JSON. Evaluate MCP after the analyzer, CLI, and security boundaries are proven.

The first agent API spike should implement `list-compositions`, `find-schema`, `validate-workspace`, and a fixture-backed or simulated `render`. Keep it read-only by default, and require explicit trust gates before any implementation invokes Docker, Crossplane functions, package downloads, or cluster reads.

## Confidence

High that agents need graph-shaped operations beyond LSP cursor methods.

High that an analyzer-first design is the right way to avoid divergent LSP, CLI, and MCP semantics.

Medium that MCP belongs in the early roadmap. It is likely useful, but should not precede a stable analyzer contract.

## Evidence That Would Change This Recommendation

- Protocol tests show LSP workspace methods are sufficient for the agent workflows.
- Users do not want a separate CLI or agent API.
- MCP becomes the primary required integration target before the CLI contract is implemented.
- Security review finds that exposing render or schema operations to agents is too risky for the first scope.
