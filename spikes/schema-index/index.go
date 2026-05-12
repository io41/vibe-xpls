package schemaindex

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type GVK struct {
	APIVersion string
	Kind       string
}

type SourceType string

const (
	SourceXRD             SourceType = "xrd"
	SourceComposition     SourceType = "composition"
	SourceProviderCRD     SourceType = "provider-crd"
	SourcePackageMetadata SourceType = "package-metadata"
)

type Source struct {
	Type SourceType
	Path string
	Name string
}

type FieldDocumentation struct {
	Path        string
	Description string
	Source      Source
}

type Resource struct {
	GVK          GVK
	Name         string
	Source       Source
	Fields       map[string]FieldDocumentation
	CompositeRef *GVK
}

type PackageMetadata struct {
	APIVersion   string
	Kind         string
	Name         string
	Dependencies []PackageDependency
	Source       Source
}

type PackageDependency struct {
	Provider      string
	Configuration string
	Version       string
}

type Index struct {
	resources map[GVK]Resource
	packages  map[string]PackageMetadata
}

func NewIndex() *Index {
	return &Index{
		resources: map[GVK]Resource{},
		packages:  map[string]PackageMetadata{},
	}
}

func LoadDir(dir string) (*Index, error) {
	idx := NewIndex()
	var files []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	for _, path := range files {
		if err := idx.IndexFile(path); err != nil {
			return nil, err
		}
	}

	return idx, nil
}

func (idx *Index) IndexFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	root, err := parseYAMLSubset(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	return idx.indexDocument(path, root, parseDocDirectives(data))
}

func (idx *Index) LookupKind(apiVersion, kind string) (Resource, bool) {
	resource, ok := idx.resources[GVK{APIVersion: apiVersion, Kind: kind}]
	if !ok {
		return Resource{}, false
	}
	return cloneResource(resource), true
}

func (idx *Index) FieldDocumentation(apiVersion, kind, fieldPath string) (FieldDocumentation, bool) {
	resource, ok := idx.resources[GVK{APIVersion: apiVersion, Kind: kind}]
	if !ok {
		return FieldDocumentation{}, false
	}
	doc, ok := resource.Fields[fieldPath]
	return doc, ok
}

func (idx *Index) PackageMetadata(name string) (PackageMetadata, bool) {
	meta, ok := idx.packages[name]
	if !ok {
		return PackageMetadata{}, false
	}
	return clonePackageMetadata(meta), true
}

func (idx *Index) Resources() []Resource {
	resources := make([]Resource, 0, len(idx.resources))
	for _, resource := range idx.resources {
		resources = append(resources, cloneResource(resource))
	}
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].GVK.APIVersion == resources[j].GVK.APIVersion {
			return resources[i].GVK.Kind < resources[j].GVK.Kind
		}
		return resources[i].GVK.APIVersion < resources[j].GVK.APIVersion
	})
	return resources
}

func (idx *Index) indexDocument(path string, root *yamlNode, directiveDocs map[string]string) error {
	apiVersion := scalarAt(root, "apiVersion")
	kind := scalarAt(root, "kind")

	switch {
	case apiVersion == "apiextensions.crossplane.io/v1" && kind == "CompositeResourceDefinition":
		return idx.indexXRD(path, root)
	case apiVersion == "apiextensions.k8s.io/v1" && kind == "CustomResourceDefinition":
		return idx.indexCRD(path, root)
	case apiVersion == "apiextensions.crossplane.io/v1" && kind == "Composition":
		return idx.indexComposition(path, root, directiveDocs)
	case strings.HasPrefix(apiVersion, "meta.pkg.crossplane.io/"):
		return idx.indexPackageMetadata(path, root, directiveDocs)
	default:
		return nil
	}
}

func (idx *Index) indexXRD(path string, root *yamlNode) error {
	source := Source{
		Type: SourceXRD,
		Path: path,
		Name: scalarAt(root, "metadata", "name"),
	}
	group := scalarAt(root, "spec", "group")
	kind := scalarAt(root, "spec", "names", "kind")
	if group == "" || kind == "" {
		return fmt.Errorf("xrd %s is missing spec.group or spec.names.kind", path)
	}

	versions := nodeAt(root, "spec", "versions")
	if versions == nil || len(versions.List) == 0 {
		return fmt.Errorf("xrd %s has no spec.versions entries", path)
	}

	for _, versionNode := range versions.List {
		if scalarAt(versionNode, "served") == "false" {
			continue
		}
		version := scalarAt(versionNode, "name")
		if version == "" {
			continue
		}
		schema := nodeAt(versionNode, "schema", "openAPIV3Schema")
		fields := collectOpenAPIDocs(schema, source)
		resource := Resource{
			GVK: GVK{
				APIVersion: group + "/" + version,
				Kind:       kind,
			},
			Name:   source.Name,
			Source: source,
			Fields: fields,
		}
		if err := idx.addResource(resource); err != nil {
			return err
		}
	}

	return nil
}

func (idx *Index) indexCRD(path string, root *yamlNode) error {
	source := Source{
		Type: SourceProviderCRD,
		Path: path,
		Name: scalarAt(root, "metadata", "name"),
	}
	group := scalarAt(root, "spec", "group")
	kind := scalarAt(root, "spec", "names", "kind")
	if group == "" || kind == "" {
		return fmt.Errorf("crd %s is missing spec.group or spec.names.kind", path)
	}

	versions := nodeAt(root, "spec", "versions")
	if versions == nil || len(versions.List) == 0 {
		return fmt.Errorf("crd %s has no spec.versions entries", path)
	}

	for _, versionNode := range versions.List {
		if scalarAt(versionNode, "served") == "false" {
			continue
		}
		version := scalarAt(versionNode, "name")
		if version == "" {
			continue
		}
		schema := nodeAt(versionNode, "schema", "openAPIV3Schema")
		fields := collectOpenAPIDocs(schema, source)
		resource := Resource{
			GVK: GVK{
				APIVersion: group + "/" + version,
				Kind:       kind,
			},
			Name:   source.Name,
			Source: source,
			Fields: fields,
		}
		if err := idx.addResource(resource); err != nil {
			return err
		}
	}

	return nil
}

func (idx *Index) indexComposition(path string, root *yamlNode, directiveDocs map[string]string) error {
	source := Source{
		Type: SourceComposition,
		Path: path,
		Name: scalarAt(root, "metadata", "name"),
	}
	resource := Resource{
		GVK: GVK{
			APIVersion: scalarAt(root, "apiVersion"),
			Kind:       scalarAt(root, "kind"),
		},
		Name:   source.Name,
		Source: source,
		Fields: docsFromDirectives(directiveDocs, source),
	}

	ref := GVK{
		APIVersion: scalarAt(root, "spec", "compositeTypeRef", "apiVersion"),
		Kind:       scalarAt(root, "spec", "compositeTypeRef", "kind"),
	}
	if ref.APIVersion != "" || ref.Kind != "" {
		resource.CompositeRef = &ref
	}

	return idx.addResource(resource)
}

func (idx *Index) indexPackageMetadata(path string, root *yamlNode, directiveDocs map[string]string) error {
	source := Source{
		Type: SourcePackageMetadata,
		Path: path,
		Name: scalarAt(root, "metadata", "name"),
	}
	resource := Resource{
		GVK: GVK{
			APIVersion: scalarAt(root, "apiVersion"),
			Kind:       scalarAt(root, "kind"),
		},
		Name:   source.Name,
		Source: source,
		Fields: docsFromDirectives(directiveDocs, source),
	}
	if err := idx.addResource(resource); err != nil {
		return err
	}

	meta := PackageMetadata{
		APIVersion: resource.GVK.APIVersion,
		Kind:       resource.GVK.Kind,
		Name:       source.Name,
		Source:     source,
	}
	dependsOn := nodeAt(root, "spec", "dependsOn")
	if dependsOn != nil {
		for _, item := range dependsOn.List {
			meta.Dependencies = append(meta.Dependencies, PackageDependency{
				Provider:      scalarAt(item, "provider"),
				Configuration: scalarAt(item, "configuration"),
				Version:       scalarAt(item, "version"),
			})
		}
	}
	idx.packages[meta.Name] = meta

	return nil
}

func (idx *Index) addResource(resource Resource) error {
	if resource.GVK.APIVersion == "" || resource.GVK.Kind == "" {
		return fmt.Errorf("cannot index resource with empty apiVersion or kind from %s", resource.Source.Path)
	}
	if _, exists := idx.resources[resource.GVK]; exists {
		return fmt.Errorf("duplicate apiVersion/kind %s/%s", resource.GVK.APIVersion, resource.GVK.Kind)
	}
	idx.resources[resource.GVK] = cloneResource(resource)
	return nil
}

func collectOpenAPIDocs(schema *yamlNode, source Source) map[string]FieldDocumentation {
	docs := map[string]FieldDocumentation{}
	collectProperties(nodeAt(schema, "properties"), nil, source, docs)
	return docs
}

func collectProperties(properties *yamlNode, prefix []string, source Source, docs map[string]FieldDocumentation) {
	if properties == nil || len(properties.Map) == 0 {
		return
	}

	keys := make([]string, 0, len(properties.Map))
	for key := range properties.Map {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		property := properties.Map[key]
		pathParts := append(append([]string{}, prefix...), key)
		fieldPath := strings.Join(pathParts, ".")
		if description := scalarAt(property, "description"); description != "" {
			docs[fieldPath] = FieldDocumentation{
				Path:        fieldPath,
				Description: description,
				Source:      source,
			}
		}
		collectProperties(nodeAt(property, "properties"), pathParts, source, docs)
		collectProperties(nodeAt(property, "items", "properties"), pathParts, source, docs)
	}
}

func parseDocDirectives(data []byte) map[string]string {
	docs := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		rest, ok := strings.CutPrefix(line, "# xpls:doc ")
		if !ok {
			continue
		}
		fieldPart, textPart, ok := strings.Cut(rest, " text=")
		if !ok {
			continue
		}
		field, ok := strings.CutPrefix(strings.TrimSpace(fieldPart), "field=")
		if !ok || field == "" {
			continue
		}
		docs[field] = strings.TrimSpace(textPart)
	}
	return docs
}

func docsFromDirectives(raw map[string]string, source Source) map[string]FieldDocumentation {
	docs := map[string]FieldDocumentation{}
	for fieldPath, description := range raw {
		docs[fieldPath] = FieldDocumentation{
			Path:        fieldPath,
			Description: description,
			Source:      source,
		}
	}
	return docs
}

type yamlNode struct {
	Scalar string
	Map    map[string]*yamlNode
	List   []*yamlNode
}

type yamlFrame struct {
	indent int
	node   *yamlNode
}

func parseYAMLSubset(data []byte) (*yamlNode, error) {
	root := &yamlNode{Map: map[string]*yamlNode{}}
	stack := []yamlFrame{{indent: -1, node: root}}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		raw := strings.TrimRight(scanner.Text(), " \t\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := leadingSpaces(raw)
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			return nil, fmt.Errorf("line %d has no parent", lineNumber)
		}
		parent := stack[len(stack)-1].node

		if strings.HasPrefix(trimmed, "- ") {
			item := &yamlNode{Map: map[string]*yamlNode{}}
			parent.List = append(parent.List, item)
			remainder := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			stack = append(stack, yamlFrame{indent: indent, node: item})
			if remainder == "" {
				continue
			}
			child, nested, err := setKeyValue(item, remainder, lineNumber)
			if err != nil {
				return nil, err
			}
			if nested {
				stack = append(stack, yamlFrame{indent: indent + 2, node: child})
			}
			continue
		}

		child, nested, err := setKeyValue(parent, trimmed, lineNumber)
		if err != nil {
			return nil, err
		}
		if nested {
			stack = append(stack, yamlFrame{indent: indent, node: child})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return root, nil
}

func setKeyValue(parent *yamlNode, content string, lineNumber int) (*yamlNode, bool, error) {
	key, value, ok := strings.Cut(content, ":")
	if !ok {
		return nil, false, fmt.Errorf("line %d is not a key/value entry", lineNumber)
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, false, fmt.Errorf("line %d has an empty key", lineNumber)
	}
	if parent.Map == nil {
		parent.Map = map[string]*yamlNode{}
	}

	value = strings.TrimSpace(value)
	child := &yamlNode{}
	if value == "" {
		child.Map = map[string]*yamlNode{}
		parent.Map[key] = child
		return child, true, nil
	}

	child.Scalar = unquoteScalar(value)
	parent.Map[key] = child
	return child, false, nil
}

func unquoteScalar(value string) string {
	if unquoted, err := strconv.Unquote(value); err == nil {
		return unquoted
	}
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}

func leadingSpaces(value string) int {
	spaces := 0
	for _, ch := range value {
		if ch != ' ' {
			return spaces
		}
		spaces++
	}
	return spaces
}

func nodeAt(root *yamlNode, path ...string) *yamlNode {
	node := root
	for _, part := range path {
		if node == nil || node.Map == nil {
			return nil
		}
		node = node.Map[part]
	}
	return node
}

func scalarAt(root *yamlNode, path ...string) string {
	node := nodeAt(root, path...)
	if node == nil {
		return ""
	}
	return node.Scalar
}

func cloneResource(resource Resource) Resource {
	clone := resource
	if resource.Fields != nil {
		clone.Fields = make(map[string]FieldDocumentation, len(resource.Fields))
		for key, value := range resource.Fields {
			clone.Fields[key] = value
		}
	}
	if resource.CompositeRef != nil {
		ref := *resource.CompositeRef
		clone.CompositeRef = &ref
	}
	return clone
}

func clonePackageMetadata(meta PackageMetadata) PackageMetadata {
	clone := meta
	if meta.Dependencies != nil {
		clone.Dependencies = append([]PackageDependency{}, meta.Dependencies...)
	}
	return clone
}
