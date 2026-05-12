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

func TestUTF8PositionsInsideMultibyteRune(t *testing.T) {
	text := "é"

	pos := PositionAtByteOffset(text, 1, EncodingUTF8)
	if pos.Line != 0 || pos.Character != 1 {
		t.Fatalf("position = %#v, want line 0 character 1", pos)
	}

	offset := ByteOffsetAtPosition(text, Position{Line: 0, Character: 1}, EncodingUTF8)
	if offset != 1 {
		t.Fatalf("offset = %d, want 1", offset)
	}
}

func TestCRLFLineEndings(t *testing.T) {
	text := "a\r\nb"

	pos := PositionAtByteOffset(text, len("a\r\n"), EncodingUTF8)
	if pos.Line != 1 || pos.Character != 0 {
		t.Fatalf("position at b = %#v, want line 1 character 0", pos)
	}

	pos = PositionAtByteOffset(text, len("a\r"), EncodingUTF8)
	if pos.Line != 0 || pos.Character != 1 {
		t.Fatalf("position inside CRLF = %#v, want line 0 character 1", pos)
	}

	offset := ByteOffsetAtPosition(text, Position{Line: 0, Character: 99}, EncodingUTF8)
	if offset != 1 {
		t.Fatalf("offset beyond CRLF line = %d, want 1", offset)
	}

	offset = ByteOffsetAtPosition(text, Position{Line: 1, Character: 0}, EncodingUTF8)
	if offset != 3 {
		t.Fatalf("offset at second line = %d, want 3", offset)
	}
}

func TestCRLineEndings(t *testing.T) {
	text := "a\rb"

	pos := PositionAtByteOffset(text, len("a\r"), EncodingUTF16)
	if pos.Line != 1 || pos.Character != 0 {
		t.Fatalf("position at b = %#v, want line 1 character 0", pos)
	}

	offset := ByteOffsetAtPosition(text, Position{Line: 0, Character: 99}, EncodingUTF16)
	if offset != 1 {
		t.Fatalf("offset beyond CR line = %d, want 1", offset)
	}

	offset = ByteOffsetAtPosition(text, Position{Line: 1, Character: 0}, EncodingUTF16)
	if offset != 2 {
		t.Fatalf("offset at second line = %d, want 2", offset)
	}
}

func TestInvalidUTF8BytesUseRawByteCountsForUTF8(t *testing.T) {
	text := string([]byte{'a', 0xff, 'b'})

	pos := PositionAtByteOffset(text, 2, EncodingUTF8)
	if pos.Line != 0 || pos.Character != 2 {
		t.Fatalf("position = %#v, want line 0 character 2", pos)
	}

	offset := ByteOffsetAtPosition(text, Position{Line: 0, Character: 2}, EncodingUTF8)
	if offset != 2 {
		t.Fatalf("offset = %d, want 2", offset)
	}
}

func TestByteOffsetAtPositionClampsNegativeValues(t *testing.T) {
	text := "abc\nxyz"

	offset := ByteOffsetAtPosition(text, Position{Line: -1, Character: 2}, EncodingUTF8)
	if offset != 0 {
		t.Fatalf("offset for negative line = %d, want 0", offset)
	}

	offset = ByteOffsetAtPosition(text, Position{Line: 1, Character: -1}, EncodingUTF8)
	if offset != len("abc\n") {
		t.Fatalf("offset for negative character = %d, want %d", offset, len("abc\n"))
	}
}
