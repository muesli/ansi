# ansi

Raw ANSI sequence helpers

## ANSI Writer

## Compressor

```go
import "github.com/muesli/ansi/compressor"

w := compressor.Writer{Forward: os.Stdout}
w.Write([]byte("\x1b[31mHello, world!\x1b[0m"))
w.Close()
```
