package compressor

import (
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/muesli/ansi"
)

type Writer struct {
	Forward io.Writer

	ansi         bool
	ansiseq      bytes.Buffer
	lastseq      bytes.Buffer
	prevlastseq  bytes.Buffer
	resetreq     bool
	compressed   int
	uncompressed int
	err          error
}

// Bytes is shorthand for declaring a new default compressor instance,
// used to immediately compress a byte slice.
func Bytes(b []byte) []byte {
	var buf bytes.Buffer
	f := Writer{
		Forward: &buf,
	}
	_, _ = f.Write(b)
	_ = f.Close()

	return buf.Bytes()
}

// String is shorthand for declaring a new default compressor instance,
// used to immediately compress a string.
func String(s string) string {
	return string(Bytes([]byte(s)))
}

// Write is used to write content to the ANSI buffer.
// A forwarding error is retained; subsequent writes return the same error.
func (w *Writer) Write(b []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	w.uncompressed += len(b)
	sequenceStart := -w.ansiseq.Len()

	for i, c := range string(b) {
		_, size := utf8.DecodeRune(b[i:])
		chunk := b[i : i+size]
		if c == ansi.Marker {
			// ANSI escape sequence
			if w.ansiseq.Len() == 0 {
				sequenceStart = i
			}
			w.ansi = true
			_, _ = w.ansiseq.Write(chunk)
		} else if w.ansi {
			_, _ = w.ansiseq.Write(chunk)
			if ansi.IsTerminator(c) {
				// ANSI sequence terminated
				w.ansi = false

				terminated := false
				if bytes.HasSuffix(w.ansiseq.Bytes(), []byte("[0m")) {
					// reset sequence
					w.prevlastseq.Reset()
					w.prevlastseq.Write(w.lastseq.Bytes())

					w.lastseq.Reset()
					terminated = true
					w.resetreq = true
				} else if c == 'm' {
					// color code
					_, _ = w.lastseq.Write(w.ansiseq.Bytes())
				}

				if !terminated {
					// did we reset the sequence just to restore it again?
					if bytes.Equal(w.ansiseq.Bytes(), w.prevlastseq.Bytes()) {
						w.resetreq = false
						w.ansiseq.Reset()
					}

					w.prevlastseq.Reset()

					if w.resetreq {
						if err := w.resetAnsi(); err != nil {
							w.err = err
							written := sequenceStart
							if written < 0 {
								written = 0
							}
							return written, err
						}
					}

					n, err := w.Forward.Write(w.ansiseq.Bytes())
					w.compressed += n
					if err == nil && n < w.ansiseq.Len() {
						err = io.ErrShortWrite
					}
					if err != nil {
						w.err = err
						written := sequenceStart + n
						if written < 0 {
							written = 0
						}
						return written, err
					}
				}

				w.ansiseq.Reset()
			}
		} else {
			if w.resetreq {
				if err := w.resetAnsi(); err != nil {
					w.err = err
					return i, err
				}
			}

			n, err := w.writeBytes(chunk)
			if err != nil {
				w.err = err
				return i + n, err
			}
		}
	}

	return len(b), nil
}

func (w *Writer) writeBytes(b []byte) (int, error) {
	written, err := w.Forward.Write(b)
	w.compressed += written
	if err == nil && written < len(b) {
		err = io.ErrShortWrite
	}
	return written, err
}

// Close finishes the compression operation. Always call it before trying to
// retrieve the final result.
func (w *Writer) Close() error {
	if w.err != nil {
		return w.err
	}
	if w.resetreq {
		if err := w.resetAnsi(); err != nil {
			w.err = err
			return err
		}
	}

	// log.Println("Written uncompressed: ", w.uncompressed)
	// log.Println("Written compressed: ", w.compressed)

	return nil
}

func (w *Writer) ResetAnsi() {
	if w.err == nil {
		w.err = w.resetAnsi()
	}
}

func (w *Writer) resetAnsi() error {
	n, err := w.Forward.Write([]byte("\x1b[0m"))
	if err != nil {
		return err
	}
	if n < len("\x1b[0m") {
		return io.ErrShortWrite
	}
	w.prevlastseq.Reset()
	w.resetreq = false
	return nil
}
