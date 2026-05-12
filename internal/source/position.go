package source

import "unicode/utf16"

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
		r, size := rune(text[i]), 1
		if r >= 0x80 {
			r, size = decodeRune(text[i:])
		}
		if r == '\n' {
			line++
			character = 0
		} else {
			character += encodedLength(r, encoding)
		}
		i += size
	}
	return Position{Line: line, Character: character}
}

func ByteOffsetAtPosition(text string, target Position, encoding Encoding) int {
	line := 0
	character := 0
	for i := 0; i < len(text); {
		if line == target.Line && character >= target.Character {
			return i
		}
		r, size := rune(text[i]), 1
		if r >= 0x80 {
			r, size = decodeRune(text[i:])
		}
		if r == '\n' {
			if line == target.Line {
				return i
			}
			line++
			character = 0
		} else {
			character += encodedLength(r, encoding)
		}
		i += size
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

func decodeRune(text string) (rune, int) {
	for i := 1; i <= len(text) && i <= 4; i++ {
		r := []rune(text[:i])
		if len(r) == 1 && string(r) == text[:i] {
			return r[0], i
		}
	}
	return rune(text[0]), 1
}
