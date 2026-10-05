package ansi

import (
	"bytes"
	"io"
	"testing"
)

func TestWriterBoundsSequenceState(t *testing.T) {
	for _, input := range [][]byte{
		append([]byte("\x1b"), bytes.Repeat([]byte("1"), 4096)...),
		bytes.Repeat([]byte("\x1b[31m"), 4096),
	} {
		w := Writer{Forward: io.Discard}
		for _, b := range input {
			_, err := w.Write([]byte{b})
			if w.ansiseq.Len() > 4096 || w.lastseq.Len() > 4096 {
				t.Fatal("retained sequence state exceeded the limit")
			}
			if err != nil {
				break
			}
		}
		if _, err := w.Write([]byte("more")); err == nil {
			t.Fatalf("error after reaching limit = %v; want ErrSequenceTooLong", err)
		}
	}
}

func TestSequenceLimitBoundaryAndReset(t *testing.T) {
	var sink bytes.Buffer
	w := Writer{Forward: &sink}
	sequence := append([]byte("\x1b"), bytes.Repeat([]byte("1"), 4094)...)
	sequence = append(sequence, 'm')
	if n, err := w.Write(sequence); n != len(sequence) || err != nil {
		t.Fatalf("sequence at limit = (%d, %v)", n, err)
	}
	if _, err := w.Write([]byte("\x1b[0m\x1b[31m")); err != nil {
		t.Fatalf("reset should release style history: %v", err)
	}
}

func TestSequenceLimitCountsForwardedPrefix(t *testing.T) {
	var sink bytes.Buffer
	w := Writer{Forward: &sink}
	input := append([]byte("abc\x1b"), bytes.Repeat([]byte("1"), 4096)...)
	if n, err := w.Write(input); n != 3 || err == nil || sink.String() != "abc" {
		t.Fatalf("Write = (%d, %v), output %q; want prefix and limit error", n, err, sink.String())
	}
}
