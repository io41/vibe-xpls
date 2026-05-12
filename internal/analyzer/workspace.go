package analyzer

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type WorkspaceShape string

const (
	WorkspaceRootPackage   WorkspaceShape = "root-package"
	WorkspaceNestedPackage WorkspaceShape = "nested-package"
	WorkspaceMultiPackage  WorkspaceShape = "multi-package"
	WorkspaceNoPackageRoot WorkspaceShape = "no-package-root"
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

var markerPriority = map[string]int{
	"crossplane.yaml": 0,
	"crossplane.yml":  1,
	"upbound.yaml":    2,
	"upbound.yml":     3,
}

func DetectWorkspace(root string) (Workspace, error) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return Workspace{}, err
	}
	rootsByDir := map[string]PackageRoot{}
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
		if _, ok := markerPriority[name]; ok {
			dir := filepath.Dir(path)
			root := PackageRoot{Root: dir, Marker: name}
			if existing, ok := rootsByDir[dir]; !ok || markerLess(root.Marker, existing.Marker) {
				rootsByDir[dir] = root
			}
		}
		return nil
	})
	if err != nil {
		return Workspace{}, err
	}
	roots := make([]PackageRoot, 0, len(rootsByDir))
	for _, root := range rootsByDir {
		roots = append(roots, root)
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].Root != roots[j].Root {
			return roots[i].Root < roots[j].Root
		}
		return markerLess(roots[i].Marker, roots[j].Marker)
	})
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

func markerLess(left string, right string) bool {
	leftPriority, leftOK := markerPriority[left]
	rightPriority, rightOK := markerPriority[right]
	if leftOK && rightOK {
		return leftPriority < rightPriority
	}
	if leftOK != rightOK {
		return leftOK
	}
	return left < right
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
