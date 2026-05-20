# Crossplane LSP First Runnable Milestone Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first runnable `vibe-xpls` binary that Zed can launch through `crossplane-yaml` as `<vibe-xpls-binary> serve` and use for Crossplane diagnostics, hover, completion, and stale diagnostic clearing.

**Architecture:** Create a Go product module with a shared analyzer core, a thin LSP adapter, and an internal debug CLI. The analyzer owns workspace/package detection, path safety, document generations, mixed YAML/template parsing, schema indexing, diagnostics, hover, and completion. The LSP adapter owns JSON-RPC framing, LSP lifecycle, document sync, position encoding conversion, and formatting analyzer results.

**Tech Stack:** Go 1.24+ module, stdlib JSON-RPC framing over stdio, `github.com/goccy/go-yaml` behind an internal parser facade, `go.yaml.in/yaml/v4` as parser fallback/reference, local `crossplane-yaml` dev-extension launch path, `go test ./...` for verification.

---

## Scope Check

This is one vertical product slice, not a collection of independent products. It deliberately excludes the public agent JSON CLI, MCP, render/validate execution, Docker, downloads, cluster reads, writes, code actions, go-to-definition, and rendered virtual documents.

The plan starts with parser selection because the design requires that decision before product code. After that, each task leaves runnable, testable code and a small commit.

## External Review Corrections

Claude Code reviewed this plan with `claude-opus-4-7` and max effort on 2026-05-13. The review found several implementation gaps that block the approved acceptance criteria. These corrections are part of the plan and override any narrower code snippets in later tasks.

Required corrections before execution is considered complete:

- **Workspace schema discovery:** add a dedicated task after the schema model that scans package roots for XRDs and provider CRDs, extracts `spec.versions[].schema.openAPIV3Schema`, records provenance, and reports conflicts on the offending schema files. Tests must cover a workspace XRD, a provider CRD, and a duplicate Crossplane core schema.
- **Real YAML path traversal:** replace the line-oriented `forEachMappingLine` strategy with a `goccy/go-yaml` parser/AST walk. It must handle mapping nodes and sequence nodes, including Composition paths such as `spec.pipeline[0].step`, `spec.pipeline[0].functionRef.name`, and `spec.pipeline[0].input`. The parser facade must expose stable paths, value spans, and key spans from the raw document.
- **Diagnostic range mapping:** LSP diagnostics must convert analyzer `Span` byte offsets into protocol ranges using the negotiated position encoding. A malformed YAML fixture must prove diagnostics on a later line are not reported at `(0,0)-(0,1)`.
- **Position encoding negotiation:** initialize must inspect `capabilities.general.positionEncodings`, choose `utf-8` when offered and otherwise `utf-16`, return `positionEncoding` in server capabilities, and use that encoding for diagnostics, hover, and completion.
- **Generation fencing:** document generations must be meaningful in analyzer and LSP tests. Hover/completion requests should capture the document generation used for offset/path resolution and return an empty response when the document generation changes before response construction. Diagnostics must include the generation that produced them and stale diagnostic publications must be dropped.
- **Workspace scan caps:** package and schema scanning must enforce `MaxYAMLFiles` and `MaxYAMLBytes`, return bounded diagnostics or debug status when caps are hit, and include a test that forces the cap.
- **Package-scoped schema indexes:** multi-package workspaces must use a schema index per package root. Hover and completion must resolve through the package containing the document so schemas from package A do not appear in package B.
- **No-root activation cascade:** implement the activation signals from the spec: ancestor package marker, documented Zed/Crossplane filename classification, Crossplane core `apiVersion`, and Crossplane document kind/shape. Tests must cover activation and deactivation clearing diagnostics.
- **Completion acceptance edits:** LSP completion items for YAML keys must include explicit text edits so accepting a completion preserves valid YAML indentation. Labels remain concise (`spec`, `kind`), but the edit inserts YAML key syntax such as `spec:` or `    kind:`. Snippet placeholders, automatic child lines, and extra newline insertion are out of scope for this milestone.
- **YAML error spans:** parser errors must produce source spans from parser token positions when available. They must not default to `(0,0)` once a parser position exists.
- **Limits defaulting:** zero fields in `Limits` must be defaulted field-by-field rather than replacing the entire caller-provided struct.
- **Symlinked workspace paths:** path safety must resolve the longest existing prefix of a path before comparing against the workspace realpath so logical paths under symlinked parents are not falsely rejected.
- **Zed extension verification:** the Zed validation task must record the `crossplane-yaml` commit or local diff and verify that `<vibe-xpls-binary> serve` is still wired into the launch path before manual validation starts.
- **Pinned dependencies:** use explicit parser dependency versions recorded in the parser decision and commit them through `go.mod`/`go.sum`.

## File Structure

- Create: `docs/research/decisions/parser-milestone-01.md` - records the parser decision for this milestone.
- Create: `go.mod` - root product Go module.
- Create: `cmd/vibe-xpls/main.go` - binary entrypoint.
- Create: `internal/app/app.go` - routes `serve`, `debug`, and `--version`.
- Create: `internal/source/position.go` - byte offset and LSP position conversion.
- Create: `internal/analyzer/limits.go` - resource limits and diagnostic caps.
- Create: `internal/analyzer/path.go` - workspace path normalization and symlink escape checks.
- Create: `internal/analyzer/workspace.go` - package root detection and workspace shape classification.
- Create: `internal/analyzer/document.go` - document store and generation tracking.
- Create: `internal/analyzer/template.go` - template span detection and same-length masking.
- Create: `internal/analyzer/yaml.go` - parser facade and stable YAML path extraction.
- Create: `internal/analyzer/schema.go` - schema model, built-ins, precedence, and workspace indexing.
- Create: `internal/analyzer/analyzer.go` - public analyzer API.
- Create: `internal/analyzer/diagnostics.go` - diagnostics and no-root activation behavior.
- Create: `internal/analyzer/hover.go` - hover facts from indexed schemas.
- Create: `internal/analyzer/completion.go` - completion candidates from indexed schemas.
- Create: `internal/debugcli/cli.go` - internal, non-contractual debug CLI.
- Create: `internal/lsp/protocol.go` - minimal LSP types.
- Create: `internal/lsp/rpc.go` - Content-Length JSON-RPC framing.
- Create: `internal/lsp/server.go` - LSP adapter over analyzer.
- Create: `internal/testkit/fixtures.go` - shared fixture helper functions.
- Create: `internal/analyzer/testdata/workspaces/**` - root, nested, multi-package, no-root, malformed, mixed-template, conflict, huge-file, and path-safety fixtures.
- Update: `docs/research/decisions/gate-04-zed-readiness.md` - manual Zed validation criteria and final evidence.

## Task 0: Baseline

**Files:**
- Read: `docs/superpowers/specs/2026-05-12-crossplane-lsp-first-runnable-milestone-design.md`
- Verify: repository state only.

- [ ] **Step 1: Confirm branch and clean worktree**

Run:

```bash
git status --short --branch
```

Expected: output shows `## research/crossplane-lsp-research-program` and no modified or untracked files.

- [ ] **Step 2: Confirm Go toolchain**

Run:

```bash
go version
```

Expected: output starts with `go version go1.` and the version is at least Go 1.24.

- [ ] **Step 3: Re-read the approved design**

Run:

```bash
sed -n '1,320p' docs/superpowers/specs/2026-05-12-crossplane-lsp-first-runnable-milestone-design.md
```

Expected: output includes product boundary, analyzer-first architecture, no external execution, and the three acceptance layers.

## Task 1: Parser Decision

**Files:**
- Create: `docs/research/decisions/parser-milestone-01.md`

- [ ] **Step 1: Write the parser decision record**

Create `docs/research/decisions/parser-milestone-01.md` with:

```markdown
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
```

- [ ] **Step 2: Commit the decision**

Run:

```bash
git add docs/research/decisions/parser-milestone-01.md
git commit -m "docs: decide first milestone parser"
```

Expected: commit succeeds.

## Task 2: Root Go Module And Binary Skeleton

**Files:**
- Create: `go.mod`
- Create: `cmd/vibe-xpls/main.go`
- Create: `internal/app/app.go`
- Create: `internal/app/app_test.go`

- [ ] **Step 1: Create root module**

Run:

```bash
go mod init github.com/io41/vibe-xpls
go get github.com/goccy/go-yaml@v1.19.2
go get go.yaml.in/yaml/v4@v4.0.0-rc.4
```

Expected: `go.mod` and `go.sum` exist.

- [ ] **Step 2: Add the application entrypoint test**

Create `internal/app/app_test.go`:

```go
package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--version"}, &stdout, &stderr, Runners{})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "vibe-xpls v0.X.X" {
		t.Fatalf("version output = %q", got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"render"}, &stdout, &stderr, Runners{})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr should explain unknown command, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout should be empty, got %q", stdout.String())
	}
}
```

- [ ] **Step 3: Run the failing test**

Run:

```bash
go test ./internal/app
```

Expected: FAIL because package `internal/app` does not exist yet.

- [ ] **Step 4: Add the app package**

Create `internal/app/app.go`:

```go
package app

import (
	"fmt"
	"io"
)

const Version = "v0.X.X"

type ServerRunner func(stdin io.Reader, stdout io.Writer, stderr io.Writer) int
type DebugRunner func(args []string, stdout io.Writer) int

type Runners struct {
	Serve ServerRunner
	Debug DebugRunner
}

func Run(args []string, stdout io.Writer, stderr io.Writer, runners Runners) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "missing command: use serve, debug, or --version")
		return 2
	}

	switch args[0] {
	case "--version", "version":
		fmt.Fprintf(stdout, "vibe-xpls %s\n", Version)
		return 0
	case "serve":
		if runners.Serve == nil {
			fmt.Fprintln(stderr, "serve command is unavailable")
			return 2
		}
		return runners.Serve(nil, stdout, stderr)
	case "debug":
		if runners.Debug == nil {
			fmt.Fprintln(stderr, "debug command is unavailable")
			return 2
		}
		return runners.Debug(args[1:], stdout)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}
```

Create `cmd/vibe-xpls/main.go`:

```go
package main

import (
	"os"

	"github.com/io41/vibe-xpls/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr, app.Runners{}))
}
```

- [ ] **Step 5: Verify the binary skeleton**

Run:

```bash
go test ./...
go run ./cmd/vibe-xpls --version
```

Expected: tests PASS and version output is `vibe-xpls v0.X.X`.

- [ ] **Step 6: Commit**

Run:

```bash
git add go.mod go.sum cmd/vibe-xpls internal/app
git commit -m "feat: scaffold vibe xpls binary"
```

Expected: commit succeeds.

## Task 3: Source Position Conversion

**Files:**
- Create: `internal/source/position.go`
- Create: `internal/source/position_test.go`

- [ ] **Step 1: Write source position tests**

Create `internal/source/position_test.go`:

```go
package source

import "testing"

func TestPositionAtByteOffsetUTF8(t *testing.T) {
	text := "apiVersion: example/v1\nmetadata:\n  name: café\n"

	pos := PositionAtByteOffset(text, len("apiVersion: example/v1\nmetadata:\n  name: café"), EncodingUTF8)

	if pos.Line != 2 || pos.Character != 13 {
		t.Fatalf("position = %#v, want line 2 character 13", pos)
	}
}

func TestPositionAtByteOffsetUTF16(t *testing.T) {
	text := "emoji: 😀\nkind: Example\n"
	offset := len("emoji: 😀")

	pos := PositionAtByteOffset(text, offset, EncodingUTF16)

	if pos.Line != 0 || pos.Character != 9 {
		t.Fatalf("position = %#v, want line 0 character 9", pos)
	}
}

func TestByteOffsetAtPositionUTF16(t *testing.T) {
	text := "emoji: 😀\nkind: Example\n"

	offset := ByteOffsetAtPosition(text, Position{Line: 1, Character: 4}, EncodingUTF16)

	if got, want := text[offset:offset+2], ": "; got != want {
		t.Fatalf("offset points to %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run failing tests**

Run:

```bash
go test ./internal/source
```

Expected: FAIL because package `internal/source` does not exist yet.

- [ ] **Step 3: Implement source positions**

Create `internal/source/position.go`:

```go
package source

import "unicode/utf16"

type Encoding string

const (
	EncodingUTF8  Encoding = "utf-8"
	EncodingUTF16 Encoding = "utf-16"
)

type Position struct {
	Line      int
	Character int
}

type Range struct {
	Start Position
	End   Position
}

func PositionAtByteOffset(text string, offset int, encoding Encoding) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}

	line := 0
	character := 0
	for i := 0; i < offset; {
		r, size := rune(text[i]), 1
		if r >= 0x80 {
			r, size = decodeRune(text[i:])
		}
		if r == '\n' {
			line++
			character = 0
		} else {
			character += encodedLength(r, encoding)
		}
		i += size
	}
	return Position{Line: line, Character: character}
}

func ByteOffsetAtPosition(text string, target Position, encoding Encoding) int {
	line := 0
	character := 0
	for i := 0; i < len(text); {
		if line == target.Line && character >= target.Character {
			return i
		}
		r, size := rune(text[i]), 1
		if r >= 0x80 {
			r, size = decodeRune(text[i:])
		}
		if r == '\n' {
			if line == target.Line {
				return i
			}
			line++
			character = 0
		} else {
			character += encodedLength(r, encoding)
		}
		i += size
	}
	return len(text)
}

func encodedLength(r rune, encoding Encoding) int {
	switch encoding {
	case EncodingUTF16:
		return len(utf16.Encode([]rune{r}))
	default:
		if r < 0x80 {
			return 1
		}
		return len(string(r))
	}
}

func decodeRune(text string) (rune, int) {
	for i := 1; i <= len(text) && i <= 4; i++ {
		r := []rune(text[:i])
		if len(r) == 1 && string(r) == text[:i] {
			return r[0], i
		}
	}
	return rune(text[0]), 1
}
```

- [ ] **Step 4: Verify source package**

Run:

```bash
go test ./internal/source
go test ./...
```

Expected: tests PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/source
git commit -m "feat: add source position conversion"
```

Expected: commit succeeds.

## Task 4: Workspace Fixtures And Package Detection

**Files:**
- Create: `internal/testkit/fixtures.go`
- Create: `internal/analyzer/testdata/workspaces/root/crossplane.yaml`
- Create: `internal/analyzer/testdata/workspaces/root/api/composition.yaml`
- Create: `internal/analyzer/testdata/workspaces/nested/packages/network/crossplane.yaml`
- Create: `internal/analyzer/testdata/workspaces/multi/packages/a/crossplane.yaml`
- Create: `internal/analyzer/testdata/workspaces/multi/packages/b/upbound.yaml`
- Create: `internal/analyzer/testdata/workspaces/no-root/plain.yaml`
- Create: `internal/analyzer/workspace.go`
- Create: `internal/analyzer/workspace_test.go`

- [ ] **Step 1: Add workspace fixture helper**

Create `internal/testkit/fixtures.go`:

```go
package testkit

import (
	"path/filepath"
	"runtime"
	"testing"
)

func RepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate testkit caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func FixturePath(t *testing.T, parts ...string) string {
	t.Helper()
	items := append([]string{RepoRoot(t)}, parts...)
	return filepath.Join(items...)
}
```

- [ ] **Step 2: Add minimal workspace fixtures**

Create `internal/analyzer/testdata/workspaces/root/crossplane.yaml`:

```yaml
apiVersion: meta.pkg.crossplane.io/v1
kind: Configuration
metadata:
  name: root-package
```

Create `internal/analyzer/testdata/workspaces/root/api/composition.yaml`:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: root-composition
```

Create `internal/analyzer/testdata/workspaces/nested/packages/network/crossplane.yaml`:

```yaml
apiVersion: meta.pkg.crossplane.io/v1
kind: Configuration
metadata:
  name: nested-network
```

Create `internal/analyzer/testdata/workspaces/multi/packages/a/crossplane.yaml`:

```yaml
apiVersion: meta.pkg.crossplane.io/v1
kind: Configuration
metadata:
  name: package-a
```

Create `internal/analyzer/testdata/workspaces/multi/packages/b/upbound.yaml`:

```yaml
apiVersion: meta.dev.upbound.io/v1alpha1
kind: Project
metadata:
  name: package-b
```

Create `internal/analyzer/testdata/workspaces/no-root/plain.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ordinary
```

- [ ] **Step 3: Write package detection tests**

Create `internal/analyzer/workspace_test.go`:

```go
package analyzer

import (
	"path/filepath"
	"testing"

	"github.com/io41/vibe-xpls/internal/testkit"
)

func TestDetectWorkspaceShapes(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want WorkspaceShape
		roots int
	}{
		{"root", []string{"internal", "analyzer", "testdata", "workspaces", "root"}, WorkspaceRootPackage, 1},
		{"nested", []string{"internal", "analyzer", "testdata", "workspaces", "nested"}, WorkspaceNestedPackage, 1},
		{"multi", []string{"internal", "analyzer", "testdata", "workspaces", "multi"}, WorkspaceMultiPackage, 2},
		{"no-root", []string{"internal", "analyzer", "testdata", "workspaces", "no-root"}, WorkspaceNoPackageRoot, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, err := DetectWorkspace(testkit.FixturePath(t, tt.path...))
			if err != nil {
				t.Fatalf("detect workspace: %v", err)
			}
			if ws.Shape != tt.want {
				t.Fatalf("shape = %s, want %s", ws.Shape, tt.want)
			}
			if len(ws.PackageRoots) != tt.roots {
				t.Fatalf("package root count = %d, want %d", len(ws.PackageRoots), tt.roots)
			}
		})
	}
}

func TestNearestPackageRoot(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	ws, err := DetectWorkspace(root)
	if err != nil {
		t.Fatalf("detect workspace: %v", err)
	}
	doc := filepath.Join(root, "api", "composition.yaml")
	pkg, ok := ws.PackageForFile(doc)
	if !ok {
		t.Fatal("expected nearest package root")
	}
	if pkg.Root != root {
		t.Fatalf("package root = %s, want %s", pkg.Root, root)
	}
}
```

- [ ] **Step 4: Run failing tests**

Run:

```bash
go test ./internal/analyzer -run 'TestDetectWorkspaceShapes|TestNearestPackageRoot'
```

Expected: FAIL because workspace types are missing.

- [ ] **Step 5: Implement package detection**

Create `internal/analyzer/workspace.go`:

```go
package analyzer

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type WorkspaceShape string

const (
	WorkspaceRootPackage    WorkspaceShape = "root-package"
	WorkspaceNestedPackage  WorkspaceShape = "nested-package"
	WorkspaceMultiPackage   WorkspaceShape = "multi-package"
	WorkspaceNoPackageRoot  WorkspaceShape = "no-package-root"
)

type Workspace struct {
	Root         string
	Shape        WorkspaceShape
	PackageRoots []PackageRoot
}

type PackageRoot struct {
	Root   string
	Marker string
}

func DetectWorkspace(root string) (Workspace, error) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return Workspace{}, err
	}
	var roots []PackageRoot
	err = filepath.WalkDir(cleanRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == ".worktrees" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if name == "crossplane.yaml" || name == "crossplane.yml" || name == "upbound.yaml" || name == "upbound.yml" {
			roots = append(roots, PackageRoot{Root: filepath.Dir(path), Marker: name})
		}
		return nil
	})
	if err != nil {
		return Workspace{}, err
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Root < roots[j].Root })
	return Workspace{Root: cleanRoot, Shape: classifyWorkspace(cleanRoot, roots), PackageRoots: roots}, nil
}

func classifyWorkspace(root string, roots []PackageRoot) WorkspaceShape {
	if len(roots) == 0 {
		return WorkspaceNoPackageRoot
	}
	if len(roots) > 1 {
		return WorkspaceMultiPackage
	}
	if roots[0].Root == root {
		return WorkspaceRootPackage
	}
	return WorkspaceNestedPackage
}

func (w Workspace) PackageForFile(path string) (PackageRoot, bool) {
	clean, err := filepath.Abs(path)
	if err != nil {
		return PackageRoot{}, false
	}
	var best PackageRoot
	for _, root := range w.PackageRoots {
		if clean == root.Root || strings.HasPrefix(clean, root.Root+string(filepath.Separator)) {
			if len(root.Root) > len(best.Root) {
				best = root
			}
		}
	}
	return best, best.Root != ""
}
```

- [ ] **Step 6: Verify package detection**

Run:

```bash
go test ./internal/analyzer -run 'TestDetectWorkspaceShapes|TestNearestPackageRoot'
go test ./...
```

Expected: tests PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add internal/testkit internal/analyzer/testdata/workspaces internal/analyzer/workspace.go internal/analyzer/workspace_test.go
git commit -m "feat: detect crossplane workspace packages"
```

Expected: commit succeeds.

## Task 5: Limits And Path Safety

**Files:**
- Create: `internal/analyzer/limits.go`
- Create: `internal/analyzer/path.go`
- Create: `internal/analyzer/path_test.go`

- [ ] **Step 1: Write path safety tests**

Create `internal/analyzer/path_test.go`:

```go
package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWorkspacePathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	_, err := ResolveWorkspacePath(root, "../outside.yaml")
	if err == nil {
		t.Fatal("expected traversal outside workspace to fail")
	}
}

func TestResolveWorkspacePathAcceptsWorkspaceFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "api", "composition.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("kind: Composition\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveWorkspacePath(root, "api/composition.yaml")
	if err != nil {
		t.Fatalf("resolve path: %v", err)
	}
	if resolved != path {
		t.Fatalf("resolved path = %s, want %s", resolved, path)
	}
}

func TestResolveWorkspacePathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.yaml"), []byte("kind: Secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := ResolveWorkspacePath(root, "linked/secret.yaml")
	if err == nil {
		t.Fatal("expected symlink escape to fail")
	}
}
```

- [ ] **Step 2: Run failing tests**

Run:

```bash
go test ./internal/analyzer -run 'TestResolveWorkspacePath'
```

Expected: FAIL because path safety functions are missing.

- [ ] **Step 3: Implement limits and path safety**

Create `internal/analyzer/limits.go`:

```go
package analyzer

import "time"

type Limits struct {
	MaxDocumentBytes      int64
	MaxDiagnosticsPerDoc int
	MaxYAMLFiles         int
	MaxYAMLBytes         int64
	DocumentSoftDeadline time.Duration
}

func DefaultLimits() Limits {
	return Limits{
		MaxDocumentBytes:      2 * 1024 * 1024,
		MaxDiagnosticsPerDoc: 100,
		MaxYAMLFiles:         10000,
		MaxYAMLBytes:         100 * 1024 * 1024,
		DocumentSoftDeadline: 500 * time.Millisecond,
	}
}
```

Create `internal/analyzer/path.go`:

```go
package analyzer

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ResolveWorkspacePath(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("workspace path %q must be relative", rel)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	joined := filepath.Clean(filepath.Join(absRoot, rel))
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		realRoot = absRoot
	}
	realJoined, err := filepath.EvalSymlinks(joined)
	if err != nil {
		realJoined = joined
	}
	if realJoined != realRoot && !strings.HasPrefix(realJoined, realRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("workspace path %q escapes root %q", rel, root)
	}
	return joined, nil
}
```

- [ ] **Step 4: Verify path safety**

Run:

```bash
go test ./internal/analyzer -run 'TestResolveWorkspacePath'
go test ./...
```

Expected: tests PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/analyzer/limits.go internal/analyzer/path.go internal/analyzer/path_test.go
git commit -m "feat: add analyzer path safety limits"
```

Expected: commit succeeds.

## Task 6: Document Store And Generations

**Files:**
- Create: `internal/analyzer/document.go`
- Create: `internal/analyzer/document_test.go`

- [ ] **Step 1: Write document generation tests**

Create `internal/analyzer/document_test.go`:

```go
package analyzer

import "testing"

func TestDocumentStoreGenerations(t *testing.T) {
	store := NewDocumentStore()

	first := store.Open("file:///composition.yaml", "kind: Composition\n")
	second := store.Change("file:///composition.yaml", "kind: Composition\nmetadata:\n  name: demo\n")

	if first.Generation != 1 {
		t.Fatalf("first generation = %d, want 1", first.Generation)
	}
	if second.Generation != 2 {
		t.Fatalf("second generation = %d, want 2", second.Generation)
	}
	if got, ok := store.Get("file:///composition.yaml"); !ok || got.Text != second.Text {
		t.Fatalf("latest document not stored: %#v ok=%v", got, ok)
	}
}

func TestDocumentStoreCloseClearsDocument(t *testing.T) {
	store := NewDocumentStore()
	store.Open("file:///composition.yaml", "kind: Composition\n")

	closed := store.Close("file:///composition.yaml")

	if closed.Generation != 2 {
		t.Fatalf("close generation = %d, want 2", closed.Generation)
	}
	if _, ok := store.Get("file:///composition.yaml"); ok {
		t.Fatal("closed document should be removed")
	}
}
```

- [ ] **Step 2: Run failing tests**

Run:

```bash
go test ./internal/analyzer -run 'TestDocumentStore'
```

Expected: FAIL because document store is missing.

- [ ] **Step 3: Implement document store**

Create `internal/analyzer/document.go`:

```go
package analyzer

import "sync"

type Generation uint64

type Document struct {
	URI        string
	Text       string
	Generation Generation
	Closed     bool
}

type DocumentStore struct {
	mu   sync.Mutex
	next map[string]Generation
	docs map[string]Document
}

func NewDocumentStore() *DocumentStore {
	return &DocumentStore{
		next: map[string]Generation{},
		docs: map[string]Document{},
	}
}

func (s *DocumentStore) Open(uri, text string) Document {
	return s.set(uri, text, false)
}

func (s *DocumentStore) Change(uri, text string) Document {
	return s.set(uri, text, false)
}

func (s *DocumentStore) Close(uri string) Document {
	s.mu.Lock()
	defer s.mu.Unlock()
	gen := s.nextGenerationLocked(uri)
	delete(s.docs, uri)
	return Document{URI: uri, Generation: gen, Closed: true}
}

func (s *DocumentStore) Get(uri string) (Document, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.docs[uri]
	return doc, ok
}

func (s *DocumentStore) set(uri, text string, closed bool) Document {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc := Document{URI: uri, Text: text, Generation: s.nextGenerationLocked(uri), Closed: closed}
	s.docs[uri] = doc
	return doc
}

func (s *DocumentStore) nextGenerationLocked(uri string) Generation {
	s.next[uri]++
	return s.next[uri]
}
```

- [ ] **Step 4: Verify document store**

Run:

```bash
go test ./internal/analyzer -run 'TestDocumentStore'
go test ./...
```

Expected: tests PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/analyzer/document.go internal/analyzer/document_test.go
git commit -m "feat: track analyzer document generations"
```

Expected: commit succeeds.

## Task 7: Template Masking And YAML Path Facade

**Files:**
- Create: `internal/analyzer/template.go`
- Create: `internal/analyzer/yaml.go`
- Create: `internal/analyzer/parse_test.go`
- Create: `internal/analyzer/testdata/workspaces/root/api/mixed-template.yaml`

- [ ] **Step 1: Add mixed template fixture**

Create `internal/analyzer/testdata/workspaces/root/api/mixed-template.yaml`:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: {{ .Name }}
spec:
  compositeTypeRef:
    apiVersion: platform.example.org/v1alpha1
    kind: CompositeBucket
  {{ .TemplatedKey }}: ignored
```

- [ ] **Step 2: Write parser facade tests**

Create `internal/analyzer/parse_test.go`:

```go
package analyzer

import (
	"os"
	"strings"
	"testing"

	"github.com/io41/vibe-xpls/internal/testkit"
)

func TestMaskTemplateActionsPreservesLength(t *testing.T) {
	text := "metadata:\n  name: {{ .Name }}\n"

	mixed := ParseMixedDocument(text)

	if len(mixed.MaskedText) != len(text) {
		t.Fatalf("masked length = %d, want %d", len(mixed.MaskedText), len(text))
	}
	if strings.Contains(mixed.MaskedText, "{{") {
		t.Fatalf("masked text still contains template delimiter: %q", mixed.MaskedText)
	}
	if len(mixed.TemplateDiagnostics) != 0 {
		t.Fatalf("unexpected template diagnostics: %#v", mixed.TemplateDiagnostics)
	}
}

func TestUnterminatedTemplateDiagnostic(t *testing.T) {
	mixed := ParseMixedDocument("metadata:\n  name: {{ .Name\n")

	if len(mixed.TemplateDiagnostics) != 1 {
		t.Fatalf("template diagnostics = %d, want 1", len(mixed.TemplateDiagnostics))
	}
	if !strings.Contains(mixed.TemplateDiagnostics[0].Message, "missing closing delimiter") {
		t.Fatalf("diagnostic = %#v", mixed.TemplateDiagnostics[0])
	}
}

func TestStablePathEligibility(t *testing.T) {
	data, err := os.ReadFile(testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root", "api", "mixed-template.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	doc := ParseYAMLDocument(string(data))

	if !doc.IsStablePath("spec.compositeTypeRef.kind") {
		t.Fatal("expected plain key path to be stable")
	}
	if doc.IsStablePath("spec.xxxxxxxxxxxxxxxxxx") {
		t.Fatal("expected templated key path to be ineligible")
	}
	offset := strings.Index(string(data), "kind: CompositeBucket")
	path, ok := doc.PathAtOffset(offset)
	if !ok || path != "spec.compositeTypeRef.kind" {
		t.Fatalf("path at offset = %q ok=%v, want spec.compositeTypeRef.kind", path, ok)
	}
}
```

- [ ] **Step 3: Run failing parser tests**

Run:

```bash
go test ./internal/analyzer -run 'TestMaskTemplateActions|TestUnterminatedTemplateDiagnostic|TestStablePathEligibility'
```

Expected: FAIL because parser facade functions are missing.

- [ ] **Step 4: Implement template masking**

Create `internal/analyzer/template.go` with a productionized version of the spike behavior:

```go
package analyzer

import "strings"

type Span struct {
	Start int
	End   int
}

type TemplateAction struct {
	Text string
	Span Span
}

type Diagnostic struct {
	URI      string
	Source   string
	Message  string
	Severity string
	Span     Span
}

type MixedDocument struct {
	RawText             string
	MaskedText          string
	Actions             []TemplateAction
	TemplateDiagnostics []Diagnostic
}

func ParseMixedDocument(text string) MixedDocument {
	actions, diagnostics := findTemplateActions(text)
	return MixedDocument{
		RawText:             text,
		MaskedText:          maskTemplateActions(text, actions),
		Actions:             actions,
		TemplateDiagnostics: diagnostics,
	}
}

func findTemplateActions(text string) ([]TemplateAction, []Diagnostic) {
	var actions []TemplateAction
	var diagnostics []Diagnostic
	for scan := 0; scan < len(text); {
		openRel := strings.Index(text[scan:], "{{")
		if openRel < 0 {
			break
		}
		start := scan + openRel
		closeRel := strings.Index(text[start+2:], "}}")
		if closeRel < 0 {
			return actions, append(diagnostics, Diagnostic{
				Source:   "template",
				Severity: "error",
				Message:  "template action is missing closing delimiter",
				Span:     Span{Start: start, End: len(text)},
			})
		}
		end := start + 2 + closeRel + 2
		actions = append(actions, TemplateAction{Text: text[start:end], Span: Span{Start: start, End: end}})
		scan = end
	}
	return actions, diagnostics
}

func maskTemplateActions(text string, actions []TemplateAction) string {
	masked := []byte(text)
	for _, action := range actions {
		for i := action.Span.Start; i < action.Span.End; i++ {
			if masked[i] != '\n' && masked[i] != '\r' {
				masked[i] = 'x'
			}
		}
	}
	return string(masked)
}
```

- [ ] **Step 5: Implement YAML path facade**

Create `internal/analyzer/yaml.go`:

```go
package analyzer

import (
	"strings"

	"github.com/goccy/go-yaml"
)

type YAMLDocument struct {
	Mixed       MixedDocument
	Values      map[string]string
	StablePaths map[string]bool
	PathSpans   map[string]Span
	Diagnostics []Diagnostic
}

func ParseYAMLDocument(text string) YAMLDocument {
	mixed := ParseMixedDocument(text)
	doc := YAMLDocument{
		Mixed:       mixed,
		Values:      map[string]string{},
		StablePaths: map[string]bool{},
		PathSpans:   map[string]Span{},
		Diagnostics: append([]Diagnostic{}, mixed.TemplateDiagnostics...),
	}
	var decoded any
	if err := yaml.Unmarshal([]byte(mixed.MaskedText), &decoded); err != nil {
		doc.Diagnostics = append(doc.Diagnostics, Diagnostic{
			Source:   "yaml",
			Severity: "error",
			Message:  err.Error(),
			Span:     Span{Start: 0, End: 0},
		})
	}
	forEachMappingLine(mixed.MaskedText, func(path, value string, keyStart, keyEnd int) {
		doc.Values[path] = value
		doc.StablePaths[path] = !spanTouchesTemplate(Span{Start: keyStart, End: keyEnd}, mixed.Actions)
		doc.PathSpans[path] = Span{Start: keyStart, End: keyEnd + len(value) + 1}
	})
	return doc
}

func (d YAMLDocument) IsStablePath(path string) bool {
	return d.StablePaths[path]
}

func (d YAMLDocument) PathAtOffset(offset int) (string, bool) {
	bestPath := ""
	bestLen := 0
	for path, span := range d.PathSpans {
		if offset >= span.Start && offset <= span.End && len(path) > bestLen {
			bestPath = path
			bestLen = len(path)
		}
	}
	return bestPath, bestPath != ""
}

func forEachMappingLine(text string, fn func(path, value string, keyStart, keyEnd int)) {
	var stack []string
	offset := 0
	for _, line := range strings.SplitAfter(text, "\n") {
		raw := strings.TrimRight(line, "\r\n")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			offset += len(line)
			continue
		}
		colon := strings.Index(raw, ":")
		if colon < 0 {
			offset += len(line)
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		depth := indent / 2
		if depth < len(stack) {
			stack = stack[:depth]
		}
		key := strings.TrimSpace(raw[:colon])
		if key == "" {
			offset += len(line)
			continue
		}
		if depth == len(stack) {
			stack = append(stack, key)
		} else {
			stack[depth] = key
		}
		path := strings.Join(stack[:depth+1], ".")
		value := strings.TrimSpace(raw[colon+1:])
		keyStart := offset + strings.Index(raw, key)
		fn(path, value, keyStart, keyStart+len(key))
		offset += len(line)
	}
}

func spanTouchesTemplate(span Span, actions []TemplateAction) bool {
	for _, action := range actions {
		if span.Start < action.Span.End && span.End > action.Span.Start {
			return true
		}
	}
	return false
}
```

- [ ] **Step 6: Verify parser facade**

Run:

```bash
go test ./internal/analyzer -run 'TestMaskTemplateActions|TestUnterminatedTemplateDiagnostic|TestStablePathEligibility'
go test ./...
```

Expected: tests PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add internal/analyzer/template.go internal/analyzer/yaml.go internal/analyzer/parse_test.go internal/analyzer/testdata/workspaces/root/api/mixed-template.yaml
git commit -m "feat: parse mixed yaml template documents"
```

Expected: commit succeeds.

## Task 8: Schema Model And Built-Ins

**Files:**
- Create: `internal/analyzer/schema.go`
- Create: `internal/analyzer/schema_test.go`

- [ ] **Step 1: Write schema model tests**

Create `internal/analyzer/schema_test.go`:

```go
package analyzer

import "testing"

func TestBuiltInCrossplaneSchemas(t *testing.T) {
	idx := NewSchemaIndex()
	idx.LoadBuiltIns()

	doc, ok := idx.FieldDocumentation("apiextensions.crossplane.io/v1", "Composition", "spec.compositeTypeRef.kind")
	if !ok {
		t.Fatal("expected built-in Composition field documentation")
	}
	if doc.Description == "" {
		t.Fatal("expected non-empty field documentation")
	}
}

func TestProviderSchemaCanBeAdded(t *testing.T) {
	idx := NewSchemaIndex()
	idx.AddWorkspaceSchema(Schema{
		GVK: SourceGVK{APIVersion: "s3.aws.upbound.io/v1beta1", Kind: "Bucket"},
		Fields: map[string]FieldDoc{
			"spec.forProvider.bucketName": {Path: "spec.forProvider.bucketName", Description: "Name of the S3 bucket to create."},
		},
		Provenance: SchemaProvenance{Path: "provider-crd.yaml", Owner: SchemaOwnerProvider},
	})

	doc, ok := idx.FieldDocumentation("s3.aws.upbound.io/v1beta1", "Bucket", "spec.forProvider.bucketName")
	if !ok {
		t.Fatal("expected provider field documentation")
	}
	if doc.Description != "Name of the S3 bucket to create." {
		t.Fatalf("description = %q", doc.Description)
	}
}

func TestCoreDuplicateDoesNotOverrideBuiltIn(t *testing.T) {
	idx := NewSchemaIndex()
	idx.LoadBuiltIns()
	idx.AddWorkspaceSchema(Schema{
		GVK: SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"},
		Fields: map[string]FieldDoc{
			"spec.compositeTypeRef.kind": {Path: "spec.compositeTypeRef.kind", Description: "wrong workspace duplicate"},
		},
		Provenance: SchemaProvenance{Path: "duplicate-composition-crd.yaml", Owner: SchemaOwnerCore},
	})

	doc, ok := idx.FieldDocumentation("apiextensions.crossplane.io/v1", "Composition", "spec.compositeTypeRef.kind")
	if !ok {
		t.Fatal("expected built-in field documentation")
	}
	if doc.Description == "wrong workspace duplicate" {
		t.Fatal("workspace duplicate replaced built-in schema")
	}
	if len(idx.Diagnostics()) != 1 {
		t.Fatalf("diagnostics = %d, want 1", len(idx.Diagnostics()))
	}
}
```

- [ ] **Step 2: Run failing schema tests**

Run:

```bash
go test ./internal/analyzer -run 'TestBuiltInCrossplaneSchemas|TestProviderSchemaCanBeAdded|TestCoreDuplicateDoesNotOverrideBuiltIn'
```

Expected: FAIL because schema model is missing.

- [ ] **Step 3: Implement schema model**

Create `internal/analyzer/schema.go`:

```go
package analyzer

type SourceGVK struct {
	APIVersion string
	Kind       string
}

type SchemaOwner string

const (
	SchemaOwnerCore     SchemaOwner = "core"
	SchemaOwnerProvider SchemaOwner = "provider"
	SchemaOwnerUser     SchemaOwner = "user"
)

type FieldDoc struct {
	Path        string
	Description string
}

type SchemaProvenance struct {
	Path  string
	Owner SchemaOwner
}

type Schema struct {
	GVK        SourceGVK
	Fields     map[string]FieldDoc
	Provenance SchemaProvenance
}

type SchemaIndex struct {
	schemas     map[SourceGVK]Schema
	diagnostics []Diagnostic
}

func NewSchemaIndex() *SchemaIndex {
	return &SchemaIndex{schemas: map[SourceGVK]Schema{}}
}

func (idx *SchemaIndex) LoadBuiltIns() {
	idx.schemas[SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"}] = Schema{
		GVK: SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"},
		Fields: map[string]FieldDoc{
			"apiVersion":                       {Path: "apiVersion", Description: "Crossplane API version for a Composition."},
			"kind":                             {Path: "kind", Description: "Crossplane resource kind."},
			"metadata.name":                    {Path: "metadata.name", Description: "Composition name."},
			"spec.compositeTypeRef.apiVersion": {Path: "spec.compositeTypeRef.apiVersion", Description: "Composite API version selected by this Composition."},
			"spec.compositeTypeRef.kind":       {Path: "spec.compositeTypeRef.kind", Description: "Composite kind selected by this Composition."},
		},
		Provenance: SchemaProvenance{Path: "builtin://crossplane/composition", Owner: SchemaOwnerCore},
	}
	idx.schemas[SourceGVK{APIVersion: "meta.pkg.crossplane.io/v1", Kind: "Configuration"}] = Schema{
		GVK: SourceGVK{APIVersion: "meta.pkg.crossplane.io/v1", Kind: "Configuration"},
		Fields: map[string]FieldDoc{
			"apiVersion":              {Path: "apiVersion", Description: "Crossplane package metadata API version."},
			"kind":                    {Path: "kind", Description: "Crossplane package metadata kind."},
			"metadata.name":           {Path: "metadata.name", Description: "Package metadata name."},
			"spec.dependsOn.provider": {Path: "spec.dependsOn.provider", Description: "Provider package dependency declared by this configuration."},
		},
		Provenance: SchemaProvenance{Path: "builtin://crossplane/configuration", Owner: SchemaOwnerCore},
	}
}

func (idx *SchemaIndex) AddWorkspaceSchema(schema Schema) {
	existing, ok := idx.schemas[schema.GVK]
	if ok && existing.Provenance.Owner == SchemaOwnerCore {
		idx.diagnostics = append(idx.diagnostics, Diagnostic{
			Source:   "schema",
			Severity: "warning",
			Message:  "workspace schema duplicates built-in Crossplane core schema",
			URI:      schema.Provenance.Path,
		})
		return
	}
	if ok {
		idx.diagnostics = append(idx.diagnostics, Diagnostic{
			Source:   "schema",
			Severity: "warning",
			Message:  "workspace schema conflicts with another workspace schema",
			URI:      schema.Provenance.Path,
		})
	}
	idx.schemas[schema.GVK] = schema
}

func (idx *SchemaIndex) FieldDocumentation(apiVersion, kind, fieldPath string) (FieldDoc, bool) {
	schema, ok := idx.schemas[SourceGVK{APIVersion: apiVersion, Kind: kind}]
	if !ok {
		return FieldDoc{}, false
	}
	doc, ok := schema.Fields[fieldPath]
	return doc, ok
}

func (idx *SchemaIndex) Fields(apiVersion, kind string) []FieldDoc {
	schema, ok := idx.schemas[SourceGVK{APIVersion: apiVersion, Kind: kind}]
	if !ok {
		return nil
	}
	fields := make([]FieldDoc, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		fields = append(fields, field)
	}
	return fields
}

func (idx *SchemaIndex) Diagnostics() []Diagnostic {
	out := make([]Diagnostic, len(idx.diagnostics))
	copy(out, idx.diagnostics)
	return out
}
```

- [ ] **Step 4: Verify schema model**

Run:

```bash
go test ./internal/analyzer -run 'TestBuiltInCrossplaneSchemas|TestProviderSchemaCanBeAdded|TestCoreDuplicateDoesNotOverrideBuiltIn'
go test ./...
```

Expected: tests PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/analyzer/schema.go internal/analyzer/schema_test.go
git commit -m "feat: add schema index with builtins"
```

Expected: commit succeeds.

## Task 9: Analyzer API, Diagnostics, Hover, And Completion

**Files:**
- Create: `internal/analyzer/analyzer.go`
- Create: `internal/analyzer/diagnostics.go`
- Create: `internal/analyzer/hover.go`
- Create: `internal/analyzer/completion.go`
- Create: `internal/analyzer/analyzer_test.go`

- [ ] **Step 1: Write analyzer behavior tests**

Create `internal/analyzer/analyzer_test.go`:

```go
package analyzer

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/io41/vibe-xpls/internal/testkit"
)

func TestAnalyzerDiagnosticsHoverAndCompletion(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\n"
	a.OpenDocument(uri, text)

	diagnostics := a.Diagnostics(uri)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	hover, ok := a.Hover(uri, "spec.compositeTypeRef.kind")
	if !ok || !strings.Contains(hover.Markdown, "Composite kind") {
		t.Fatalf("hover = %#v ok=%v", hover, ok)
	}
	completion := a.Completion(uri, "spec.compositeTypeRef")
	if !containsCompletion(completion.Items, "kind") {
		t.Fatalf("completion missing kind: %#v", completion.Items)
	}
}

func TestAnalyzerUnknownProviderDoesNotInventFields(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "bucket.yaml")
	text := "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n"
	a.OpenDocument(uri, text)

	completion := a.Completion(uri, "spec.forProvider")
	if len(completion.Items) != 0 {
		t.Fatalf("unknown provider schema should not invent completions: %#v", completion.Items)
	}
}

func TestNoRootActivationTogglesDiagnostics(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	a.OpenDocument(uri, "apiVersion: v1\nkind: ConfigMap\nbroken\n")
	if got := len(a.Diagnostics(uri)); got != 0 {
		t.Fatalf("ordinary no-root yaml should stay quiet, got %d diagnostics", got)
	}
	a.ChangeDocument(uri, "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nbroken\n")
	if got := len(a.Diagnostics(uri)); got == 0 {
		t.Fatal("Crossplane no-root document should activate diagnostics")
	}
	a.ChangeDocument(uri, "apiVersion: v1\nkind: ConfigMap\nbroken\n")
	if got := len(a.Diagnostics(uri)); got != 0 {
		t.Fatalf("removing activation should clear diagnostics, got %d", got)
	}
}

func TestHugeDocumentDowngradesAnalysis(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: Limits{
		MaxDocumentBytes:      16,
		MaxDiagnosticsPerDoc: 100,
		MaxYAMLFiles:         10000,
		MaxYAMLBytes:         100 * 1024 * 1024,
		DocumentSoftDeadline: DefaultLimits().DocumentSoftDeadline,
	}})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "large.yaml")
	a.OpenDocument(uri, strings.Repeat("a", 32))
	diagnostics := a.Diagnostics(uri)
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "size limit") {
		t.Fatalf("expected size limit diagnostic, got %#v", diagnostics)
	}
}

func containsCompletion(items []CompletionItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run failing analyzer tests**

Run:

```bash
go test ./internal/analyzer -run 'TestAnalyzerDiagnosticsHoverAndCompletion|TestAnalyzerUnknownProviderDoesNotInventFields|TestNoRootActivationTogglesDiagnostics|TestHugeDocumentDowngradesAnalysis'
```

Expected: FAIL because analyzer API is missing.

- [ ] **Step 3: Implement analyzer API**

Create `internal/analyzer/analyzer.go`:

```go
package analyzer

type Options struct {
	WorkspaceRoot string
	Limits        Limits
}

type Analyzer struct {
	workspace Workspace
	limits    Limits
	docs      *DocumentStore
	schemas   *SchemaIndex
}

func New(options Options) (*Analyzer, error) {
	limits := options.Limits
	if limits.MaxDiagnosticsPerDoc == 0 {
		limits = DefaultLimits()
	}
	workspace, err := DetectWorkspace(options.WorkspaceRoot)
	if err != nil {
		return nil, err
	}
	schemas := NewSchemaIndex()
	schemas.LoadBuiltIns()
	return &Analyzer{
		workspace: workspace,
		limits:    limits,
		docs:      NewDocumentStore(),
		schemas:   schemas,
	}, nil
}

func (a *Analyzer) OpenDocument(uri, text string) Document {
	return a.docs.Open(uri, text)
}

func (a *Analyzer) ChangeDocument(uri, text string) Document {
	return a.docs.Change(uri, text)
}

func (a *Analyzer) CloseDocument(uri string) Document {
	return a.docs.Close(uri)
}

func (a *Analyzer) Document(uri string) (Document, bool) {
	return a.docs.Get(uri)
}

func (a *Analyzer) PathAtOffset(uri string, offset int) (string, bool) {
	doc, ok := a.docs.Get(uri)
	if !ok {
		return "", false
	}
	return ParseYAMLDocument(doc.Text).PathAtOffset(offset)
}
```

Create `internal/analyzer/diagnostics.go`:

```go
package analyzer

import "strings"

func (a *Analyzer) Diagnostics(uri string) []Diagnostic {
	doc, ok := a.docs.Get(uri)
	if !ok {
		return nil
	}
	if int64(len(doc.Text)) > a.limits.MaxDocumentBytes {
		return []Diagnostic{{
			URI:      uri,
			Source:   "analyzer",
			Severity: "warning",
			Message:  "document exceeds analyzer size limit; full analysis skipped",
			Span:     Span{Start: 0, End: 0},
		}}
	}
	parsed := ParseYAMLDocument(doc.Text)
	if a.workspace.Shape == WorkspaceNoPackageRoot && !isActiveCrossplaneDocument(parsed) {
		return nil
	}
	out := append([]Diagnostic{}, parsed.Diagnostics...)
	if len(out) > a.limits.MaxDiagnosticsPerDoc {
		out = out[:a.limits.MaxDiagnosticsPerDoc]
	}
	return out
}

func isActiveCrossplaneDocument(doc YAMLDocument) bool {
	apiVersion := doc.Values["apiVersion"]
	kind := doc.Values["kind"]
	if strings.Contains(apiVersion, "crossplane.io/") {
		return true
	}
	switch kind {
	case "Composition", "CompositeResourceDefinition", "CustomResourceDefinition", "Configuration":
		return true
	default:
		return false
	}
}
```

Create `internal/analyzer/hover.go`:

```go
package analyzer

type Hover struct {
	Markdown string
}

func (a *Analyzer) Hover(uri, fieldPath string) (Hover, bool) {
	doc, ok := a.docs.Get(uri)
	if !ok {
		return Hover{}, false
	}
	parsed := ParseYAMLDocument(doc.Text)
	if !parsed.IsStablePath(fieldPath) {
		return Hover{}, false
	}
	apiVersion := parsed.Values["apiVersion"]
	kind := parsed.Values["kind"]
	field, ok := a.schemas.FieldDocumentation(apiVersion, kind, fieldPath)
	if !ok {
		return Hover{}, false
	}
	return Hover{Markdown: field.Description}, true
}

func (a *Analyzer) HoverAtOffset(uri string, offset int) (Hover, bool) {
	path, ok := a.PathAtOffset(uri, offset)
	if !ok {
		return Hover{}, false
	}
	return a.Hover(uri, path)
}
```

Create `internal/analyzer/completion.go`:

```go
package analyzer

import "strings"

type Completion struct {
	Items []CompletionItem
}

type CompletionItem struct {
	Label         string
	Documentation string
}

func (a *Analyzer) Completion(uri, parentPath string) Completion {
	doc, ok := a.docs.Get(uri)
	if !ok {
		return Completion{}
	}
	parsed := ParseYAMLDocument(doc.Text)
	apiVersion := parsed.Values["apiVersion"]
	kind := parsed.Values["kind"]
	var items []CompletionItem
	prefix := parentPath
	if prefix != "" {
		prefix += "."
	}
	for _, field := range a.schemas.Fields(apiVersion, kind) {
		if !strings.HasPrefix(field.Path, prefix) {
			continue
		}
		rest := strings.TrimPrefix(field.Path, prefix)
		if rest == "" || strings.Contains(rest, ".") {
			continue
		}
		items = append(items, CompletionItem{Label: rest, Documentation: field.Description})
	}
	return Completion{Items: items}
}

func (a *Analyzer) CompletionAtOffset(uri string, offset int) Completion {
	path, ok := a.PathAtOffset(uri, offset)
	if !ok {
		return Completion{}
	}
	parent := path
	if dot := strings.LastIndex(parent, "."); dot >= 0 {
		parent = parent[:dot]
	}
	return a.Completion(uri, parent)
}
```

- [ ] **Step 4: Verify analyzer behavior**

Run:

```bash
go test ./internal/analyzer -run 'TestAnalyzerDiagnosticsHoverAndCompletion|TestAnalyzerUnknownProviderDoesNotInventFields|TestNoRootActivationTogglesDiagnostics|TestHugeDocumentDowngradesAnalysis'
go test ./...
```

Expected: tests PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/analyzer/analyzer.go internal/analyzer/diagnostics.go internal/analyzer/hover.go internal/analyzer/completion.go internal/analyzer/analyzer_test.go
git commit -m "feat: expose analyzer diagnostics hover completion"
```

Expected: commit succeeds.

## Task 10: Internal Debug CLI

**Files:**
- Create: `internal/debugcli/cli.go`
- Create: `internal/debugcli/cli_test.go`
- Modify: `internal/app/app.go`
- Modify: `cmd/vibe-xpls/main.go`

- [ ] **Step 1: Write debug CLI tests**

Create `internal/debugcli/cli_test.go`:

```go
package debugcli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDiagnosticsCommandIsInternalJSON(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"diagnostics", "--workspace", "../analyzer/testdata/workspaces/root", "--uri", "file:///composition.yaml", "--text", "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, output=%s", code, out.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("decode json: %v; output=%s", err, out.String())
	}
	if envelope["contract"] != "internal-debug" {
		t.Fatalf("contract = %#v, want internal-debug", envelope["contract"])
	}
}

func TestUnknownCommandFails(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"render"}, &out)

	if code == 0 {
		t.Fatal("unknown debug command should fail")
	}
	if !strings.Contains(out.String(), "unknown debug command") {
		t.Fatalf("output = %q", out.String())
	}
}
```

- [ ] **Step 2: Run failing debug CLI tests**

Run:

```bash
go test ./internal/debugcli
```

Expected: FAIL because debug CLI package is missing.

- [ ] **Step 3: Implement debug CLI**

Create `internal/debugcli/cli.go`:

```go
package debugcli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/io41/vibe-xpls/internal/analyzer"
)

type Envelope struct {
	OK       bool   `json:"ok"`
	Contract string `json:"contract"`
	Command  string `json:"command"`
	Data     any    `json:"data,omitempty"`
	Error    string `json:"error,omitempty"`
}

func Run(args []string, out io.Writer) int {
	if len(args) == 0 {
		return write(out, Envelope{OK: false, Contract: "internal-debug", Error: "missing debug command"})
	}
	switch args[0] {
	case "diagnostics":
		return diagnostics(args[1:], out)
	default:
		return write(out, Envelope{OK: false, Contract: "internal-debug", Command: args[0], Error: fmt.Sprintf("unknown debug command %q", args[0])})
	}
}

func diagnostics(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("diagnostics", flag.ContinueOnError)
	workspace := fs.String("workspace", ".", "workspace root")
	uri := fs.String("uri", "file:///debug.yaml", "document URI")
	text := fs.String("text", "", "document text")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return write(out, Envelope{OK: false, Contract: "internal-debug", Command: "diagnostics", Error: err.Error()})
	}
	a, err := analyzer.New(analyzer.Options{WorkspaceRoot: *workspace, Limits: analyzer.DefaultLimits()})
	if err != nil {
		return write(out, Envelope{OK: false, Contract: "internal-debug", Command: "diagnostics", Error: err.Error()})
	}
	a.OpenDocument(*uri, *text)
	return write(out, Envelope{OK: true, Contract: "internal-debug", Command: "diagnostics", Data: a.Diagnostics(*uri)})
}

func write(out io.Writer, envelope Envelope) int {
	if err := json.NewEncoder(out).Encode(envelope); err != nil {
		return 1
	}
	if envelope.OK {
		return 0
	}
	return 2
}
```

- [ ] **Step 4: Wire debug command through app**

The `internal/app` package already has a `DebugRunner` in `Runners`. Modify only `cmd/vibe-xpls/main.go` to pass `debugcli.Run`:

Modify `cmd/vibe-xpls/main.go`:

```go
package main

import (
	"os"

	"github.com/io41/vibe-xpls/internal/app"
	"github.com/io41/vibe-xpls/internal/debugcli"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr, app.Runners{
		Debug: debugcli.Run,
	}))
}
```

- [ ] **Step 5: Verify debug CLI**

Run:

```bash
go test ./internal/debugcli ./internal/app
go run ./cmd/vibe-xpls debug diagnostics --workspace internal/analyzer/testdata/workspaces/root --uri file:///composition.yaml --text 'apiVersion: apiextensions.crossplane.io/v1
kind: Composition
'
go test ./...
```

Expected: tests PASS and debug command prints JSON with `"contract":"internal-debug"`.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/debugcli internal/app cmd/vibe-xpls
git commit -m "feat: add internal analyzer debug cli"
```

Expected: commit succeeds.

## Task 11: LSP Protocol And Framing

**Files:**
- Create: `internal/lsp/protocol.go`
- Create: `internal/lsp/rpc.go`
- Create: `internal/lsp/rpc_test.go`

- [ ] **Step 1: Write JSON-RPC framing tests**

Create `internal/lsp/rpc_test.go`:

```go
package lsp

import (
	"bufio"
	"bytes"
	"testing"
)

func TestReadWriteMessage(t *testing.T) {
	var out bytes.Buffer
	err := WriteMessage(&out, Message{JSONRPC: "2.0", ID: 1, Method: "initialize"})
	if err != nil {
		t.Fatalf("write message: %v", err)
	}

	msg, err := ReadMessage(bufio.NewReader(&out))
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if msg.Method != "initialize" || msg.ID.(float64) != 1 {
		t.Fatalf("message = %#v", msg)
	}
}

func TestReadMessageRequiresContentLength(t *testing.T) {
	_, err := ReadMessage(bufio.NewReader(bytes.NewBufferString("\r\n{}")))
	if err == nil {
		t.Fatal("expected missing Content-Length error")
	}
}
```

- [ ] **Step 2: Run failing framing tests**

Run:

```bash
go test ./internal/lsp -run 'TestRead'
```

Expected: FAIL because LSP package is missing.

- [ ] **Step 3: Implement protocol and framing**

Create `internal/lsp/protocol.go`:

```go
package lsp

import "encoding/json"

type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}
```

Create `internal/lsp/rpc.go`:

```go
package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func ReadMessage(r *bufio.Reader) (Message, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return Message{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return Message{}, fmt.Errorf("malformed header %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return Message{}, err
			}
			length = parsed
		}
	}
	if length < 0 {
		return Message{}, errors.New("missing Content-Length header")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return Message{}, err
	}
	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return Message{}, err
	}
	return msg, nil
}

func WriteMessage(w io.Writer, msg Message) error {
	if msg.JSONRPC == "" {
		msg.JSONRPC = "2.0"
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}
```

- [ ] **Step 4: Verify framing**

Run:

```bash
go test ./internal/lsp -run 'TestRead'
go test ./...
```

Expected: tests PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/lsp/protocol.go internal/lsp/rpc.go internal/lsp/rpc_test.go
git commit -m "feat: add lsp jsonrpc framing"
```

Expected: commit succeeds.

## Task 12: LSP Server Adapter

**Files:**
- Create: `internal/lsp/server.go`
- Create: `internal/lsp/server_test.go`
- Modify: `cmd/vibe-xpls/main.go`

- [ ] **Step 1: Write LSP server tests**

Create `internal/lsp/server_test.go`:

```go
package lsp

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/io41/vibe-xpls/internal/testkit"
)

func TestInitializeAdvertisesHoverCompletionAndSync(t *testing.T) {
	output := runServerFrames(t, frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{"general":{"positionEncodings":["utf-8","utf-16"]}},"rootUri":"file://`+escapeTestRoot(t)+`"}}`))

	if !strings.Contains(output, `"hoverProvider":true`) {
		t.Fatalf("missing hoverProvider in %s", output)
	}
	if !strings.Contains(output, `"completionProvider"`) {
		t.Fatalf("missing completionProvider in %s", output)
	}
}

func TestDidClosePublishesEmptyDiagnostics(t *testing.T) {
	root := escapeTestRoot(t)
	output := runServerFrames(t,
		frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"rootUri":"file://`+root+`","capabilities":{}}}`),
		frame(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file://`+root+`/api/composition.yaml","text":"apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"}}}`),
		frame(`{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":"file://`+root+`/api/composition.yaml"}}}`),
	)

	if !strings.Contains(output, `"method":"textDocument/publishDiagnostics"`) {
		t.Fatalf("missing diagnostics notification: %s", output)
	}
	if !strings.Contains(output, `"diagnostics":[]`) {
		t.Fatalf("missing empty diagnostics clear: %s", output)
	}
}

func TestHoverAndCompletionUseAnalyzer(t *testing.T) {
	root := escapeTestRoot(t)
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\n"
	output := runServerFrames(t,
		frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"rootUri":"file://`+root+`","capabilities":{}}}`),
		frame(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file://`+root+`/api/composition.yaml","text":`+strconv.Quote(text)+`}}}`),
		frame(`{"jsonrpc":"2.0","id":2,"method":"textDocument/hover","params":{"textDocument":{"uri":"file://`+root+`/api/composition.yaml"},"position":{"line":4,"character":4}}}`),
		frame(`{"jsonrpc":"2.0","id":3,"method":"textDocument/completion","params":{"textDocument":{"uri":"file://`+root+`/api/composition.yaml"},"position":{"line":3,"character":2}}}`),
	)

	if !strings.Contains(output, "Composite kind") {
		t.Fatalf("hover did not include analyzer docs: %s", output)
	}
	if !strings.Contains(output, `"label":"kind"`) {
		t.Fatalf("completion did not include analyzer item: %s", output)
	}
}

func runServerFrames(t *testing.T, frames ...string) string {
	t.Helper()
	var in bytes.Buffer
	for _, frame := range frames {
		in.WriteString(frame)
	}
	in.WriteString(frame(`{"jsonrpc":"2.0","method":"exit"}`))
	var out bytes.Buffer
	code := NewServer(&in, &out, &out).Run()
	if code != 0 {
		t.Fatalf("server exit code = %d output=%s", code, out.String())
	}
	return out.String()
}

func frame(body string) string {
	return "Content-Length: " + strconv.Itoa(len([]byte(body))) + "\r\n\r\n" + body
}

func escapeTestRoot(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root"), "\\", "/")
}
```

- [ ] **Step 2: Run failing LSP server tests**

Run:

```bash
go test ./internal/lsp -run 'TestInitialize|TestDidClose|TestHoverAndCompletion'
```

Expected: FAIL because `NewServer` is missing.

- [ ] **Step 3: Implement LSP server adapter**

Create `internal/lsp/server.go`:

```go
package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/io41/vibe-xpls/internal/analyzer"
	"github.com/io41/vibe-xpls/internal/source"
)

type Server struct {
	in       *bufio.Reader
	out      io.Writer
	errOut   io.Writer
	analyzer *analyzer.Analyzer
}

func NewServer(in io.Reader, out io.Writer, errOut io.Writer) *Server {
	return &Server{in: bufio.NewReader(in), out: out, errOut: errOut}
}

func (s *Server) Run() int {
	for {
		msg, err := ReadMessage(s.in)
		if errors.Is(err, io.EOF) {
			return 0
		}
		if err != nil {
			fmt.Fprintln(s.errOut, err)
			return 1
		}
		if msg.Method == "exit" {
			return 0
		}
		if err := s.handle(msg); err != nil {
			fmt.Fprintln(s.errOut, err)
			return 1
		}
	}
}

func (s *Server) handle(msg Message) error {
	switch msg.Method {
	case "initialize":
		root := rootFromInitialize(msg.Params)
		a, err := analyzer.New(analyzer.Options{WorkspaceRoot: root, Limits: analyzer.DefaultLimits()})
		if err != nil {
			return s.respond(msg.ID, nil, &ResponseError{Code: -32602, Message: err.Error()})
		}
		s.analyzer = a
		return s.respond(msg.ID, map[string]any{
			"capabilities": map[string]any{
				"textDocumentSync": 1,
				"hoverProvider":    true,
				"completionProvider": map[string]any{
					"triggerCharacters": []string{".", ":", "\n"},
				},
			},
			"serverInfo": map[string]any{"name": "vibe-xpls", "version": "v0.X.X"},
		}, nil)
	case "shutdown":
		return s.respond(msg.ID, nil, nil)
	case "textDocument/didOpen":
		uri, text := openParams(msg.Params)
		s.analyzer.OpenDocument(uri, text)
		return s.publishDiagnostics(uri)
	case "textDocument/didChange":
		uri, text := changeParams(msg.Params)
		s.analyzer.ChangeDocument(uri, text)
		return s.publishDiagnostics(uri)
	case "textDocument/didClose":
		uri := closeParams(msg.Params)
		s.analyzer.CloseDocument(uri)
		return s.notify("textDocument/publishDiagnostics", map[string]any{"uri": uri, "diagnostics": []any{}})
	case "textDocument/hover":
		uri, position := positionedParams(msg.Params)
		if doc, ok := s.analyzer.Document(uri); ok {
			offset := source.ByteOffsetAtPosition(doc.Text, source.Position{Line: position.Line, Character: position.Character}, source.EncodingUTF16)
			if hover, ok := s.analyzer.HoverAtOffset(uri, offset); ok {
				return s.respond(msg.ID, map[string]any{"contents": map[string]any{"kind": "markdown", "value": hover.Markdown}}, nil)
			}
		}
		return s.respond(msg.ID, nil, nil)
	case "textDocument/completion":
		uri, position := positionedParams(msg.Params)
		items := []map[string]any{}
		if doc, ok := s.analyzer.Document(uri); ok {
			offset := source.ByteOffsetAtPosition(doc.Text, source.Position{Line: position.Line, Character: position.Character}, source.EncodingUTF16)
			for _, item := range s.analyzer.CompletionAtOffset(uri, offset).Items {
				items = append(items, map[string]any{"label": item.Label, "documentation": item.Documentation})
			}
		}
		return s.respond(msg.ID, map[string]any{"isIncomplete": false, "items": items}, nil)
	default:
		if msg.ID != nil {
			return s.respond(msg.ID, nil, &ResponseError{Code: -32601, Message: "method not found"})
		}
		return nil
	}
}

func (s *Server) publishDiagnostics(uri string) error {
	diagnostics := s.analyzer.Diagnostics(uri)
	items := make([]map[string]any, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		items = append(items, map[string]any{
			"range":    map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 0, "character": 1}},
			"severity": 1,
			"source":   diagnostic.Source,
			"message":  diagnostic.Message,
		})
	}
	return s.notify("textDocument/publishDiagnostics", map[string]any{"uri": uri, "diagnostics": items})
}

func (s *Server) respond(id any, result any, responseError *ResponseError) error {
	return WriteMessage(s.out, Message{JSONRPC: "2.0", ID: id, Result: result, Error: responseError})
}

func (s *Server) notify(method string, params any) error {
	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return WriteMessage(s.out, Message{JSONRPC: "2.0", Method: method, Params: payload})
}

func rootFromInitialize(raw json.RawMessage) string {
	var params struct{ RootURI string `json:"rootUri"` }
	_ = json.Unmarshal(raw, &params)
	if params.RootURI == "" {
		return "."
	}
	parsed, err := url.Parse(params.RootURI)
	if err != nil || parsed.Path == "" {
		return "."
	}
	return parsed.Path
}
```

Add simple parameter helpers in the same file:

```go
func openParams(raw json.RawMessage) (string, string) {
	var params struct {
		TextDocument struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"textDocument"`
	}
	_ = json.Unmarshal(raw, &params)
	return params.TextDocument.URI, params.TextDocument.Text
}

func changeParams(raw json.RawMessage) (string, string) {
	var params struct {
		TextDocument struct{ URI string `json:"uri"` } `json:"textDocument"`
		ContentChanges []struct{ Text string `json:"text"` } `json:"contentChanges"`
	}
	_ = json.Unmarshal(raw, &params)
	text := ""
	if len(params.ContentChanges) > 0 {
		text = params.ContentChanges[len(params.ContentChanges)-1].Text
	}
	return params.TextDocument.URI, text
}

func closeParams(raw json.RawMessage) string {
	var params struct {
		TextDocument struct{ URI string `json:"uri"` } `json:"textDocument"`
	}
	_ = json.Unmarshal(raw, &params)
	return params.TextDocument.URI
}

func positionedParams(raw json.RawMessage) (string, Position) {
	var params struct {
		TextDocument struct{ URI string `json:"uri"` } `json:"textDocument"`
		Position     Position `json:"position"`
	}
	_ = json.Unmarshal(raw, &params)
	return params.TextDocument.URI, params.Position
}
```

- [ ] **Step 4: Wire serve command**

Modify `cmd/vibe-xpls/main.go`:

```go
package main

import (
	"io"
	"os"

	"github.com/io41/vibe-xpls/internal/app"
	"github.com/io41/vibe-xpls/internal/debugcli"
	"github.com/io41/vibe-xpls/internal/lsp"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr, app.Runners{
		Debug: debugcli.Run,
		Serve: func(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
			if stdin == nil {
				stdin = os.Stdin
			}
			return lsp.NewServer(stdin, stdout, stderr).Run()
		},
	}))
}
```

- [ ] **Step 5: Verify LSP adapter**

Run:

```bash
go test ./internal/lsp
go test ./...
go run ./cmd/vibe-xpls --version
```

Expected: tests PASS and binary still prints `vibe-xpls v0.X.X`.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/lsp cmd/vibe-xpls
git commit -m "feat: serve analyzer over lsp"
```

Expected: commit succeeds.

## Task 12A: Completion Text Edit Acceptance

**Files:**
- Modify: `internal/analyzer/completion.go`
- Modify: `internal/analyzer/analyzer_test.go`
- Modify: `internal/lsp/server.go`
- Modify: `internal/lsp/server_test.go`
- Modify: `docs/research/decisions/gate-04-zed-readiness.md`

- [ ] **Step 1: Add analyzer completion edit tests**

Append tests to `internal/analyzer/analyzer_test.go` that prove cursor-based completions carry an edit range and YAML key text.

```go
func TestAnalyzerCompletionAtOffsetIncludesRootKeyEdit(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-root-edit.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nmetadata:\n  name: root-composition\ns"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	item, ok := completionItemByLabel(completion.Items, "spec")
	if !ok {
		t.Fatalf("completion missing spec: %#v", completion.Items)
	}
	if item.TextEdit == nil {
		t.Fatalf("spec completion missing text edit: %#v", item)
	}
	if item.TextEdit.NewText != "spec:" {
		t.Fatalf("new text = %q, want spec:", item.TextEdit.NewText)
	}
	if got, want := item.TextEdit.Replace, (Span{Start: strings.LastIndex(text, "\n") + 1, End: len(text)}); got != want {
		t.Fatalf("replace span = %#v, want %#v", got, want)
	}
}

func TestAnalyzerCompletionAtOffsetIncludesNestedKeyEdit(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-nested-edit.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    k"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	item, ok := completionItemByLabel(completion.Items, "kind")
	if !ok {
		t.Fatalf("completion missing kind: %#v", completion.Items)
	}
	if item.TextEdit == nil {
		t.Fatalf("kind completion missing text edit: %#v", item)
	}
	if item.TextEdit.NewText != "    kind:" {
		t.Fatalf("new text = %q, want indented kind key", item.TextEdit.NewText)
	}
	if got, want := item.TextEdit.Replace, (Span{Start: strings.LastIndex(text, "\n") + 1, End: len(text)}); got != want {
		t.Fatalf("replace span = %#v, want %#v", got, want)
	}
}

func completionItemByLabel(items []CompletionItem, label string) (CompletionItem, bool) {
	for _, item := range items {
		if item.Label == label {
			return item, true
		}
	}
	return CompletionItem{}, false
}
```

- [ ] **Step 2: Run analyzer tests and confirm failure**

Run:

```bash
go test ./internal/analyzer -run 'TestAnalyzerCompletionAtOffsetIncludes.*KeyEdit'
```

Expected: FAIL because `CompletionItem` does not yet carry a text edit.

- [ ] **Step 3: Add analyzer text edit metadata**

Modify `internal/analyzer/completion.go` so cursor-based completion items carry the replacement span and plain YAML key text. Keep path-based `Completion(uri, parentPath)` label-only because it has no cursor range.

```go
type CompletionItem struct {
	Label         string
	Documentation string
	TextEdit      *CompletionTextEdit
}

type CompletionTextEdit struct {
	Replace Span
	NewText string
}
```

Update `completionContext` to retain the line replacement span and indentation:

```go
type completionContext struct {
	parentPath     string
	prefix         string
	rootOccurrence PathOccurrence
	replace        Span
	indent         string
}
```

In `completionContextAtOffset`, set the replacement span to the current key prefix, including leading indentation:

```go
return completionContext{
	parentPath:     parentPath,
	prefix:         prefix,
	rootOccurrence: rootOccurrence,
	replace:        Span{Start: lineStart, End: offset},
	indent:         text[lineStart:indentEnd],
}, true
```

After building and prefix-filtering schema completion items in `CompletionAtOffset`, attach the edit:

```go
completion := filterCompletion(completionFromSchema(a.schemas, apiVersion, kind, context.parentPath), context.prefix)
for i := range completion.Items {
	completion.Items[i].TextEdit = &CompletionTextEdit{
		Replace: context.replace,
		NewText: context.indent + completion.Items[i].Label + ":",
	}
}
return completion
```

- [ ] **Step 4: Verify analyzer edit tests**

Run:

```bash
go test ./internal/analyzer -run 'TestAnalyzerCompletionAtOffsetIncludes.*KeyEdit|TestAnalyzerCompletionAtOffsetUsesMappingKeyContext|TestAnalyzerCompletionAtOffsetFiltersPartialMappingKey'
```

Expected: tests PASS.

- [ ] **Step 5: Add LSP completion text edit tests**

Add an LSP test to `internal/lsp/server_test.go` that verifies the protocol payload includes an explicit `textEdit` and no snippet fields.

```go
func TestCompletionItemsIncludePlainTextEdits(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "api", "completion-edit.yaml"))
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nmetadata:\n  name: root-composition\ns"

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{"rootUri": fileURI(root), "capabilities": map[string]any{}}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
		requestFrame(t, 2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtOffset(t, text, len(text), source.EncodingUTF16),
		}),
	)

	completion := resultMap(t, responseForID(t, messages, 2))
	item := completionItemByLabelForTest(t, asSlice(t, completion["items"]), "spec")
	edit := asMap(t, item["textEdit"])
	if edit["newText"] != "spec:" {
		t.Fatalf("newText = %#v, want spec:", edit["newText"])
	}
	if _, ok := item["insertTextFormat"]; ok {
		t.Fatalf("completion should not use snippets: %#v", item)
	}
	rng := asMap(t, edit["range"])
	start := asMap(t, rng["start"])
	end := asMap(t, rng["end"])
	if start["line"] != float64(4) || start["character"] != float64(0) || end["line"] != float64(4) || end["character"] != float64(1) {
		t.Fatalf("textEdit range = %#v, want line 4 char 0..1", rng)
	}
}

func completionItemByLabelForTest(t *testing.T, items []any, label string) map[string]any {
	t.Helper()
	for _, raw := range items {
		item := asMap(t, raw)
		if item["label"] == label {
			return item
		}
	}
	t.Fatalf("completion item %q not found in %#v", label, items)
	return nil
}
```

- [ ] **Step 6: Run LSP test and confirm failure**

Run:

```bash
go test ./internal/lsp -run TestCompletionItemsIncludePlainTextEdits
```

Expected: FAIL because the LSP adapter does not yet serialize `textEdit`.

- [ ] **Step 7: Serialize LSP text edits**

Modify `internal/lsp/server.go` to add protocol structs:

```go
type completionItem struct {
	Label         string    `json:"label"`
	Documentation string    `json:"documentation,omitempty"`
	TextEdit      *textEdit `json:"textEdit,omitempty"`
}

type textEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}
```

When converting analyzer completion items, map analyzer byte spans through the existing `rangeFromSpan` helper and the negotiated position encoding:

```go
items := make([]completionItem, 0, len(completion.Items))
for _, item := range completion.Items {
	out := completionItem{Label: item.Label, Documentation: item.Documentation}
	if item.TextEdit != nil {
		out.TextEdit = &textEdit{
			Range:   s.rangeFromSpan(snapshot.Text, item.TextEdit.Replace),
			NewText: item.TextEdit.NewText,
		}
	}
	items = append(items, out)
}
```

Do not add `insertTextFormat`, snippet placeholders, or extra newline insertion in this milestone.

- [ ] **Step 8: Verify completion text edits**

Run:

```bash
go test ./internal/analyzer -run 'TestAnalyzerCompletionAtOffsetIncludes.*KeyEdit|TestAnalyzerCompletionAtOffsetUsesMappingKeyContext|TestAnalyzerCompletionAtOffsetFiltersPartialMappingKey'
go test ./internal/lsp -run 'TestCompletionItemsIncludePlainTextEdits|TestHoverAndCompletionUseAnalyzer|TestHoverAndCompletionUseNegotiatedUTF8Positions'
go test ./...
```

Expected: tests PASS.

- [ ] **Step 9: Update Zed validation artifact**

Modify `docs/research/decisions/gate-04-zed-readiness.md` so the completion validation notes include:

```markdown
- Completion acceptance must be tested, not only suggestion visibility.
- Root-level key completion: with a partial root key such as `s` after `metadata.name`, accepting `spec` inserts `spec:` at column 0.
- Nested key completion: with a partial nested key such as `k` under `spec.compositeTypeRef`, accepting `kind` inserts `kind:` at the child-key indentation.
- Completion remains plain text; snippet placeholders and automatic child-line insertion are not part of this milestone.
```

- [ ] **Step 10: Commit**

Run:

```bash
git add internal/analyzer/completion.go internal/analyzer/analyzer_test.go internal/lsp/server.go internal/lsp/server_test.go docs/research/decisions/gate-04-zed-readiness.md
git commit -m "fix: add completion text edits"
```

Expected: commit succeeds.

## Task 13: Zed Manual Validation Artifact

**Files:**
- Update: `docs/research/decisions/gate-04-zed-readiness.md`

- [ ] **Step 1: Build the local binary for Zed**

Run:

```bash
go build -o <vibe-xpls-binary> ./cmd/vibe-xpls
<vibe-xpls-binary> --version
```

Expected: version output is `vibe-xpls v0.X.X`.

- [ ] **Step 2: Update validation record**

Update `docs/research/decisions/gate-04-zed-readiness.md`:

```markdown
# Zed First Runnable Milestone Validation

## Binary

- Binary path: `<vibe-xpls-binary>`
- Version output: `vibe-xpls v0.X.X`
- Canonical path checked with: `realpath <vibe-xpls-binary>`
- Built from worktree: `<vibe-xpls-worktree>`

## Zed Extension

- Extension repository: `<crossplane-yaml-repo>`
- Launch command: `<vibe-xpls-binary> serve`

## Required Checks

- [ ] Zed launches `<vibe-xpls-binary>`.
- [ ] Missing-binary behavior is understandable when `<vibe-xpls-binary>` is missing.
- [ ] Root package attaches.
- [ ] Nested package attaches.
- [ ] Multi-package workspace attaches without schema cross-contamination.
- [ ] No-root workspace stays quiet.
- [ ] `.yaml` attach behavior was checked without user `file_types` mapping.
- [ ] `.yaml` attach behavior was checked with the documented Crossplane `file_types` mapping.
- [ ] Diagnostics appear.
- [ ] Diagnostics clear after valid edits.
- [ ] Diagnostics clear after document close.
- [ ] Hover works visibly.
- [ ] Completion works visibly.
- [ ] Accepting completion inserts the completed YAML key at the correct indentation.

## Evidence Notes

Record Zed log excerpts, fixture paths, and screenshots or manual observations here during validation. Do not record environment variables, kubeconfig content, registry credentials, or secret-bearing file contents.
```

- [ ] **Step 3: Commit validation template**

Run:

```bash
git add docs/research/decisions/gate-04-zed-readiness.md
git commit -m "docs: add zed validation checklist"
```

Expected: commit succeeds.

## Task 14: Final Verification And Milestone Evidence

**Files:**
- Modify: `docs/research/decisions/gate-04-zed-readiness.md`

- [ ] **Step 1: Run full tests**

Run:

```bash
go test ./...
```

Expected: all tests PASS.

- [ ] **Step 2: Run internal debug CLI smoke check**

Run:

```bash
go run ./cmd/vibe-xpls debug diagnostics --workspace internal/analyzer/testdata/workspaces/root --uri file:///composition.yaml --text 'apiVersion: apiextensions.crossplane.io/v1
kind: Composition
'
```

Expected: JSON output includes `"contract":"internal-debug"` and `"ok":true`.

- [ ] **Step 3: Run LSP smoke check**

Run:

```bash
printf 'Content-Length: 120\r\n\r\n{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"rootUri":"file://%s","capabilities":{}}}' "$(pwd)/internal/analyzer/testdata/workspaces/root" | go run ./cmd/vibe-xpls serve
```

Expected: output starts with `Content-Length:` and includes `"serverInfo":{"name":"vibe-xpls","version":"v0.X.X"}`.

- [ ] **Step 4: Complete manual Zed validation**

Follow `docs/research/decisions/gate-04-zed-readiness.md` and mark each checkbox with the evidence gathered.

Expected: every required manual check is marked complete or the implementation returns to the failing task for a fix.

- [ ] **Step 5: Confirm no external execution occurred**

Run:

```bash
rg -n "crossplane render|crossplane beta validate|docker|kubectl|kubeconfig|http.Get|net/http|os/exec" cmd internal
```

Expected: no matches that invoke external Crossplane execution, Docker, cluster reads, network reads, or workspace writes during normal editor behavior. Matches in tests must be reviewed and documented before committing.

- [ ] **Step 6: Commit final validation evidence**

Run:

```bash
git add docs/research/decisions/gate-04-zed-readiness.md
git commit -m "docs: record first runnable zed validation"
```

Expected: commit succeeds.

## Completion Checklist

- [ ] Parser decision is committed before product code.
- [ ] Root Go module builds.
- [ ] `go test ./...` passes.
- [ ] `go run ./cmd/vibe-xpls --version` prints `vibe-xpls v0.X.X`.
- [ ] Analyzer fixture tests cover package detection, schema lookup, schema precedence, diagnostics, hover, completion, mixed YAML/template basics, no-root activation, bounded-resource behavior, path safety, and stale generation behavior.
- [ ] LSP tests cover document sync, diagnostics, hover, completion, completion text edits, negotiated position conversion, stale diagnostic clearing, and stale pull-request behavior.
- [ ] Manual Zed validation evidence is recorded.
- [ ] No normal editor path invokes external Crossplane execution, Docker, downloads, cluster reads, kubeconfig reads, or workspace writes.
- [ ] Debug CLI output remains marked `internal-debug`.
