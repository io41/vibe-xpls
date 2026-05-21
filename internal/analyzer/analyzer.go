package analyzer

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

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
	workspace, err := DetectWorkspace(options.WorkspaceRoot)
	if err != nil {
		return nil, err
	}
	schemas := NewSchemaIndex()
	schemas.LoadBuiltIns()
	if !schemas.bundleStatus.OK {
		return nil, fmt.Errorf("load built-in schema bundle: %s", schemas.bundleStatus.Message)
	}
	return &Analyzer{
		workspace: workspace,
		limits:    defaultLimits(options.Limits),
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
	_, parsed, ok := a.currentYAMLDocument(uri)
	if !ok {
		return "", false
	}
	return parsed.PathAtOffset(offset)
}

func (a *Analyzer) currentYAMLDocument(uri string) (Document, YAMLDocument, bool) {
	doc, ok := a.docs.Get(uri)
	if !ok || a.documentExceedsLimit(doc) {
		return Document{}, YAMLDocument{}, false
	}
	return doc, ParseYAMLDocument(doc.Text), true
}

func (a *Analyzer) documentExceedsLimit(doc Document) bool {
	return a.limits.MaxDocumentBytes > 0 && int64(len(doc.Text)) > a.limits.MaxDocumentBytes
}

func defaultLimits(limits Limits) Limits {
	defaults := DefaultLimits()
	if limits.MaxDocumentBytes == 0 {
		limits.MaxDocumentBytes = defaults.MaxDocumentBytes
	}
	if limits.MaxDiagnosticsPerDoc == 0 {
		limits.MaxDiagnosticsPerDoc = defaults.MaxDiagnosticsPerDoc
	}
	if limits.MaxYAMLFiles == 0 {
		limits.MaxYAMLFiles = defaults.MaxYAMLFiles
	}
	if limits.MaxYAMLBytes == 0 {
		limits.MaxYAMLBytes = defaults.MaxYAMLBytes
	}
	if limits.DocumentSoftDeadline == 0 {
		limits.DocumentSoftDeadline = defaults.DocumentSoftDeadline
	}
	return limits
}

func filePathFromURI(uri string) (string, bool) {
	parsed, err := url.Parse(uri)
	if err == nil && parsed.Scheme == "file" {
		if parsed.Host != "" && parsed.Host != "localhost" {
			return "", false
		}
		if parsed.Path == "" {
			return "", false
		}
		return filepath.Clean(filepath.FromSlash(parsed.Path)), true
	}
	if filepath.IsAbs(uri) {
		return filepath.Clean(uri), true
	}
	return "", false
}

func baseNameFromURI(uri string) string {
	if path, ok := filePathFromURI(uri); ok {
		return filepath.Base(path)
	}
	return filepath.Base(strings.TrimPrefix(uri, "file://"))
}
