## Decision.

Make AI agents first-class through a read-only structured JSON CLI first, backed by the shared analyzer. Do not require agents to scrape cursor-oriented LSP methods. Evaluate MCP after analyzer contracts, CLI response shapes, and trust boundaries stabilize. Keep first-scope render fixture-backed or behind explicit trust gates.

## Evidence.

- `docs/research/lanes/03-agent-semantic-api.md` says agents need graph-shaped operations over package metadata, XRDs, Compositions, function pipelines, templates, schemas, and rendered resources, not cursor-oriented LSP scraping.
- `docs/research/spikes/06-agent-api.md` validates a dependency-light JSON CLI envelope with `ok`, command, structured data, diagnostics, errors, and security metadata.
- The agent API spike covers `list-compositions`, `find-schema`, `validate-workspace`, and `render` as read-only commands, with no Docker, Crossplane CLI, network, cluster access, or workspace writes.
- `docs/research/spikes/06-agent-api.md` marks render output as fixture-backed and non-authoritative, proving command shape and safety metadata without pretending to execute Crossplane.
- `docs/research/lanes/11-security-reliability.md` requires agent-facing commands to default to read-only static analysis and puts Docker render, downloads, cluster reads, and writes behind explicit trust gates.
- `docs/research/spikes/05-render-validate.md` shows render crosses Docker permissions and validate can depend on cache, network, credential-helper, and kubeconfig state, making those operations unsuitable for implicit agent or LSP hot-path execution.
- `docs/research/lanes/09-existing-tooling.md` recommends a shared analyzer that can power LSP, CLI, and later agent-facing adapters.

## Alternatives Considered.

- Tell agents to drive LSP directly. Rejected because LSP is cursor-oriented and awkward for repository-level tasks such as listing Compositions, resolving schemas, validating a workspace, or explaining render limits.
- Build MCP first. Rejected for first scope because `docs/research/lanes/03-agent-semantic-api.md` and `docs/research/lanes/11-security-reliability.md` both treat MCP as an outer adapter after analyzer, CLI, and security boundaries are stable.
- Make render authoritative in the first agent surface. Rejected because `docs/research/spikes/05-render-validate.md` proves render and validate cross Docker, cache, network, and kubeconfig boundaries.
- Return human logs for agents to parse. Rejected because the agent API research requires structured JSON with explicit diagnostics, errors, and security metadata.

## Risks.

- Fixture-backed render can mislead agents if `authoritative: false` and execution metadata are not prominent and stable.
- A CLI-first surface may need later adaptation for long-running clients that want persistent workspace state.
- Analyzer, LSP, CLI, and future MCP adapters can drift unless they share one semantic model and test fixtures.
- Trust gates must remain explicit; otherwise render, downloads, or cluster reads could be triggered indirectly by agent workflows.

## What Would Change This Decision.

- Protocol tests show LSP workspace methods are sufficient for the target agent workflows without scraping or fragile cursor orchestration.
- Users require MCP as the first integration surface before a CLI contract exists.
- Security review finds even fixture-backed render unsafe or too misleading for first scope.
- A shared analyzer cannot provide stable structured data for CLI, LSP, and future MCP without unacceptable complexity.
