package analyzer

import (
	"path/filepath"
	"sync"
)

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

func (s *DocumentStore) GetByFilePath(path string) (Document, bool) {
	clean, err := filepath.Abs(path)
	if err != nil {
		return Document{}, false
	}
	clean = filepath.Clean(clean)
	s.mu.Lock()
	defer s.mu.Unlock()
	var best Document
	bestOK := false
	for _, doc := range s.docs {
		docPath, ok := filePathFromURI(doc.URI)
		if !ok || filepath.Clean(docPath) != clean {
			continue
		}
		if !bestOK || doc.Generation > best.Generation {
			best = doc
			bestOK = true
		}
	}
	return best, bestOK
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
