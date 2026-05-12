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
