package source

import "testing"

func TestPositionAtByteOffsetUTF8(t *testing.T) {
	text := "apiVersion: example/v1\nmetadata:\n  name: café\n"

	pos := PositionAtByteOffset(text, len("apiVersion: example/v1\nmetadata:\n  name: café"), EncodingUTF8)

	if pos.Line != 2 || pos.Character != 13 {
		t.Fatalf("position = %#v, want line 2 character 13", pos)
	}
}

func TestPositionAtByteOffsetUTF16(t *testing.T) {
	text := "emoji: 😀\nkind: Example\n"
	offset := len("emoji: 😀")

	pos := PositionAtByteOffset(text, offset, EncodingUTF16)

	if pos.Line != 0 || pos.Character != 9 {
		t.Fatalf("position = %#v, want line 0 character 9", pos)
	}
}

func TestByteOffsetAtPositionUTF16(t *testing.T) {
	text := "emoji: 😀\nkind: Example\n"

	offset := ByteOffsetAtPosition(text, Position{Line: 1, Character: 4}, EncodingUTF16)

	if got, want := text[offset:offset+2], ": "; got != want {
		t.Fatalf("offset points to %q, want %q", got, want)
	}
}
