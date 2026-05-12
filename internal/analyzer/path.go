package analyzer

import (
	"errors"
	"fmt"
	"os"
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
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root %q: %w", root, err)
	}
	joined := filepath.Clean(filepath.Join(absRoot, rel))
	realJoined, err := resolveRealPathThroughExistingPrefix(joined)
	if err != nil {
		return "", fmt.Errorf("resolve workspace path %q: %w", rel, err)
	}
	if !pathWithinRoot(realRoot, realJoined) {
		return "", fmt.Errorf("workspace path %q escapes root %q", rel, root)
	}
	return joined, nil
}

func resolveRealPathThroughExistingPrefix(path string) (string, error) {
	existing, suffix, err := longestExistingPrefix(path)
	if err != nil {
		return "", err
	}
	realExisting, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", err
	}
	parts := append([]string{realExisting}, suffix...)
	return filepath.Clean(filepath.Join(parts...)), nil
}

func longestExistingPrefix(path string) (string, []string, error) {
	current := filepath.Clean(path)
	var suffix []string
	for {
		if _, err := os.Lstat(current); err == nil {
			return current, suffix, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", nil, err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", nil, fmt.Errorf("no existing path prefix for %q", path)
		}
		suffix = append([]string{filepath.Base(current)}, suffix...)
		current = parent
	}
}

func pathWithinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}
