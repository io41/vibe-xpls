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
