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

func TestResolveWorkspacePathAcceptsSymlinkedWorkspaceRoot(t *testing.T) {
	realRoot := t.TempDir()
	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	if err := os.Symlink(realRoot, root); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
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

func TestResolveWorkspacePathAcceptsMissingLeafInsideWorkspace(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "api", "composition.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
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

func TestResolveWorkspacePathRejectsSymlinkEscapeWithMissingLeaf(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := ResolveWorkspacePath(root, "linked/missing.yaml")
	if err == nil {
		t.Fatal("expected symlink escape to fail")
	}
}
