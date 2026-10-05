package ansi

import (
	"bytes"
	"io"
	"testing"
)

type errorWriter func([]byte) (int, error)

func (f errorWriter) Write(p []byte) (int, error) { return f(p) }

func TestWritePropagatesEscapeErrors(t *testing.T) {
	for _, split := range []bool{false, true} {
		w := Writer{Forward: errorWriter(func(p []byte) (int, error) { return 0, io.ErrClosedPipe })}
		input := "\x1b[31m"
		if split {
			if _, err := w.Write([]byte(input[:2])); err != nil {
				t.Fatal(err)
			}
			input = input[2:]
		}
		if n, err := w.Write([]byte(input)); n != 0 || err != io.ErrClosedPipe {
			t.Fatalf("Write = (%d, %v); want (0, ErrClosedPipe)", n, err)
		}
	}
}

func TestWriteDetectsShortEscapeWrite(t *testing.T) {
	w := Writer{Forward: errorWriter(func(p []byte) (int, error) { return len(p) - 1, nil })}
	if n, err := w.Write([]byte("\x1b[31m")); n != 4 || err != io.ErrShortWrite {
		t.Fatalf("Write = (%d, %v); want (4, ErrShortWrite)", n, err)
	}
}

func TestWriteCountsSuccessfulPrefix(t *testing.T) {
	remaining := 2
	w := Writer{Forward: errorWriter(func(p []byte) (int, error) {
		if remaining == 0 {
			return 0, io.ErrClosedPipe
		}
		remaining -= len(p)
		return len(p), nil
	})}
	if n, err := w.Write([]byte("abc")); n != 2 || err != io.ErrClosedPipe {
		t.Fatalf("Write = (%d, %v); want (2, ErrClosedPipe)", n, err)
	}
}

func TestWriteErrorCountStaysWithinInput(t *testing.T) {
	w := Writer{Forward: errorWriter(func(p []byte) (int, error) { return len(p) - 1, io.ErrClosedPipe })}
	input := []byte{'\x1b', 0xff, 'm'}
	if n, err := w.Write(input); err == nil || n < 0 || n > len(input) {
		t.Fatalf("Write = (%d, %v) for %d input bytes", n, err, len(input))
	}
}

func TestWriteAfterEscapeFailure(t *testing.T) {
	var sink bytes.Buffer
	w := Writer{Forward: errorWriter(func(p []byte) (int, error) { return 0, io.ErrClosedPipe })}
	_, _ = w.Write([]byte("\x1b[31m"))
	w.Forward = &sink
	if n, err := w.Write([]byte("next")); err == nil || n != 0 || sink.Len() != 0 {
		t.Fatalf("Write after failed escape = (%d, %v), output %q; want persistent error", n, err, sink.String())
	}
}

func TestSplitEscapeErrorCount(t *testing.T) {
	w := Writer{Forward: errorWriter(func(p []byte) (int, error) { return 3, io.ErrClosedPipe })}
	_, _ = w.Write([]byte("\x1b["))
	if n, err := w.Write([]byte("31m")); n != 1 || err != io.ErrClosedPipe {
		t.Fatalf("Write = (%d, %v); want (1, ErrClosedPipe)", n, err)
	}
}

func TestWriteReportsResetAndRestoreErrors(t *testing.T) {
	for _, restore := range []bool{false, true} {
		var sink bytes.Buffer
		w := Writer{Forward: &sink}
		_, _ = w.Write([]byte("\x1b[31m"))
		w.Forward = errorWriter(func(p []byte) (int, error) { return 0, io.ErrClosedPipe })
		if restore {
			w.RestoreAnsi()
		} else {
			w.ResetAnsi()
		}
		if n, err := w.Write([]byte("next")); n != 0 || err != io.ErrClosedPipe {
			t.Fatalf("Write after reset/restore = (%d, %v); want (0, ErrClosedPipe)", n, err)
		}
	}
}

func TestWritePreservesInputBytesAcrossChunks(t *testing.T) {
	for _, input := range [][]byte{
		[]byte("你好"),
		{'\x1b', 0xff, 'm'},
	} {
		var sink bytes.Buffer
		w := Writer{Forward: &sink}
		for _, b := range input {
			if n, err := w.Write([]byte{b}); n != 1 || err != nil {
				t.Fatalf("Write = (%d, %v)", n, err)
			}
		}
		if !bytes.Equal(sink.Bytes(), input) {
			t.Fatalf("output %q; want original bytes %q", sink.Bytes(), input)
		}
	}
}
