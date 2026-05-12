package source

import (
	"unicode/utf16"
	"unicode/utf8"
)

type Encoding string

const (
	EncodingUTF8  Encoding = "utf-8"
	EncodingUTF16 Encoding = "utf-16"
)

type Position struct {
	Line      int
	Character int
}

type Range struct {
	Start Position
	End   Position
}

func PositionAtByteOffset(text string, offset int, encoding Encoding) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}

	line := 0
	character := 0
	for i := 0; i < offset; {
		if eolLen := lineEndingLength(text, i); eolLen > 0 {
			if offset < i+eolLen {
				return Position{Line: line, Character: character}
			}
			line++
			character = 0
			i += eolLen
			continue
		}

		if encoding == EncodingUTF16 {
			r, size := utf8.DecodeRuneInString(text[i:])
			character += encodedLength(r, encoding)
			i += size
		} else {
			character++
			i++
		}
	}
	return Position{Line: line, Character: character}
}

func ByteOffsetAtPosition(text string, target Position, encoding Encoding) int {
	if target.Line < 0 {
		return 0
	}
	if target.Character < 0 {
		target.Character = 0
	}

	line := 0
	character := 0
	for i := 0; i < len(text); {
		if line == target.Line && character >= target.Character {
			return i
		}
		if eolLen := lineEndingLength(text, i); eolLen > 0 {
			if line == target.Line {
				return i
			}
			line++
			character = 0
			i += eolLen
			continue
		}

		if encoding == EncodingUTF16 {
			r, size := utf8.DecodeRuneInString(text[i:])
			character += encodedLength(r, encoding)
			i += size
		} else {
			character++
			i++
		}
	}
	return len(text)
}

func encodedLength(r rune, encoding Encoding) int {
	switch encoding {
	case EncodingUTF16:
		return len(utf16.Encode([]rune{r}))
	default:
		if r < 0x80 {
			return 1
		}
		return len(string(r))
	}
}

func lineEndingLength(text string, offset int) int {
	switch text[offset] {
	case '\n':
		return 1
	case '\r':
		if offset+1 < len(text) && text[offset+1] == '\n' {
			return 2
		}
		return 1
	default:
		return 0
	}
}
