package ansi

import "errors"

// MaxSequenceSize bounds the escape sequence and style state retained by a Writer.
const MaxSequenceSize = 4096

// ErrSequenceTooLong indicates that a Writer exceeded MaxSequenceSize.
var ErrSequenceTooLong = errors.New("ansi: sequence buffer exceeds 4096 bytes")

const Marker = '\x1B'

func IsTerminator(c rune) bool {
	return (c >= 0x40 && c <= 0x5a) || (c >= 0x61 && c <= 0x7a)
}
