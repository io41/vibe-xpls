package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/io41/vibe-xpls/internal/testkit"
)

func TestDetectWorkspaceShapes(t *testing.T) {
	tests := []struct {
		name  string
		path  []string
		want  WorkspaceShape
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

func TestDuplicateMarkersSharePackageRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\n")
	writeFile(t, filepath.Join(root, "upbound.yaml"), "apiVersion: meta.dev.upbound.io/v1alpha1\nkind: Project\n")

	ws, err := DetectWorkspace(root)
	if err != nil {
		t.Fatalf("detect workspace: %v", err)
	}
	if ws.Shape != WorkspaceRootPackage {
		t.Fatalf("shape = %s, want %s", ws.Shape, WorkspaceRootPackage)
	}
	if len(ws.PackageRoots) != 1 {
		t.Fatalf("package root count = %d, want 1", len(ws.PackageRoots))
	}
	if ws.PackageRoots[0].Root != root {
		t.Fatalf("package root = %s, want %s", ws.PackageRoots[0].Root, root)
	}
	if ws.PackageRoots[0].Marker != "crossplane.yaml" {
		t.Fatalf("marker = %s, want crossplane.yaml", ws.PackageRoots[0].Marker)
	}
}

func TestPackageForFileNestedPackage(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "nested")
	ws, err := DetectWorkspace(root)
	if err != nil {
		t.Fatalf("detect workspace: %v", err)
	}
	want := filepath.Join(root, "packages", "network")
	doc := filepath.Join(want, "templates", "composition.yaml")

	pkg, ok := ws.PackageForFile(doc)
	if !ok {
		t.Fatal("expected nested package root")
	}
	if pkg.Root != want {
		t.Fatalf("package root = %s, want %s", pkg.Root, want)
	}
}

func TestPackageForFileMultiPackage(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "multi")
	ws, err := DetectWorkspace(root)
	if err != nil {
		t.Fatalf("detect workspace: %v", err)
	}
	want := filepath.Join(root, "packages", "b")
	doc := filepath.Join(want, "apis", "project.yaml")

	pkg, ok := ws.PackageForFile(doc)
	if !ok {
		t.Fatal("expected multi-package root")
	}
	if pkg.Root != want {
		t.Fatalf("package root = %s, want %s", pkg.Root, want)
	}
}

func TestPackageForFileNoMatch(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	ws, err := DetectWorkspace(root)
	if err != nil {
		t.Fatalf("detect workspace: %v", err)
	}
	outside := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root", "plain.yaml")

	if pkg, ok := ws.PackageForFile(outside); ok {
		t.Fatalf("package root = %#v, want no match", pkg)
	}
}

func TestPackageForFilePrefixBoundary(t *testing.T) {
	root := t.TempDir()
	pkgRoot := filepath.Join(root, "pkg")
	otherRoot := filepath.Join(root, "pkg2")
	ws := Workspace{PackageRoots: []PackageRoot{{Root: pkgRoot, Marker: "crossplane.yaml"}}}

	if _, ok := ws.PackageForFile(filepath.Join(otherRoot, "composition.yaml")); ok {
		t.Fatal("pkg root should not match pkg2 path")
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
