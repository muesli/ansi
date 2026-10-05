package ansi

import (
	"bytes"
	"io"
	"unicode/utf8"
)

type Writer struct {
	Forward io.Writer

	ansi       bool
	ansiseq    bytes.Buffer
	lastseq    bytes.Buffer
	seqchanged bool
	runeBuf    []byte
	err        error
}

// Write is used to write content to the ANSI buffer.
// If buffered escape or style state exceeds MaxSequenceSize, Write returns
// ErrSequenceTooLong. Subsequent writes return the same error.
func (w *Writer) Write(b []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	sequenceStart := -w.ansiseq.Len()
	for i, c := range string(b) {
		if (c == Marker || w.ansi) && w.ansiseq.Len()+utf8.RuneLen(c) > MaxSequenceSize {
			w.err = ErrSequenceTooLong
			written := sequenceStart
			if written < 0 {
				written = 0
			}
			return written, w.err
		}
		if c == Marker {
			if w.ansiseq.Len() == 0 {
				sequenceStart = i
			}
			// ANSI escape sequence
			w.ansi = true
			w.seqchanged = true
			_, _ = w.ansiseq.WriteRune(c)
		} else if w.ansi {
			_, _ = w.ansiseq.WriteRune(c)
			if IsTerminator(c) {
				// ANSI sequence terminated
				w.ansi = false

				if bytes.HasSuffix(w.ansiseq.Bytes(), []byte("[0m")) {
					// reset sequence
					w.lastseq.Reset()
					w.seqchanged = false
				} else if c == 'm' {
					// color code
					if w.lastseq.Len()+w.ansiseq.Len() > MaxSequenceSize {
						w.err = ErrSequenceTooLong
						written := sequenceStart
						if written < 0 {
							written = 0
						}
						return written, w.err
					}
					_, _ = w.lastseq.Write(w.ansiseq.Bytes())
				}

				_, _ = w.ansiseq.WriteTo(w.Forward)
			}
		} else {
			_, err := w.writeRune(c)
			if err != nil {
				return 0, err
			}
		}
	}

	return len(b), nil
}

func (w *Writer) writeRune(r rune) (int, error) {
	if w.runeBuf == nil {
		w.runeBuf = make([]byte, utf8.UTFMax)
	}
	n := utf8.EncodeRune(w.runeBuf, r)
	return w.Forward.Write(w.runeBuf[:n])
}

func (w *Writer) LastSequence() string {
	return w.lastseq.String()
}

func (w *Writer) ResetAnsi() {
	if !w.seqchanged {
		return
	}
	_, _ = w.Forward.Write([]byte("\x1b[0m"))
}

func (w *Writer) RestoreAnsi() {
	_, _ = w.Forward.Write(w.lastseq.Bytes())
}
