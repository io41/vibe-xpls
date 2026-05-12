package analyzer

import (
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
