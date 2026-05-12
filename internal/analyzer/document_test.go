package analyzer

import "testing"

func TestDocumentStoreGenerations(t *testing.T) {
	store := NewDocumentStore()

	first := store.Open("file:///composition.yaml", "kind: Composition\n")
	second := store.Change("file:///composition.yaml", "kind: Composition\nmetadata:\n  name: demo\n")

	if first.Generation != 1 {
		t.Fatalf("first generation = %d, want 1", first.Generation)
	}
	if second.Generation != 2 {
		t.Fatalf("second generation = %d, want 2", second.Generation)
	}
	if got, ok := store.Get("file:///composition.yaml"); !ok || got.Text != second.Text {
		t.Fatalf("latest document not stored: %#v ok=%v", got, ok)
	}
}

func TestDocumentStoreCloseClearsDocument(t *testing.T) {
	store := NewDocumentStore()
	store.Open("file:///composition.yaml", "kind: Composition\n")

	closed := store.Close("file:///composition.yaml")

	if closed.Generation != 2 {
		t.Fatalf("close generation = %d, want 2", closed.Generation)
	}
	if _, ok := store.Get("file:///composition.yaml"); ok {
		t.Fatal("closed document should be removed")
	}
}
