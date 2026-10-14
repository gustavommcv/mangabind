package main

import (
	"fmt"
	"io"
)

// Human-readable output is best-effort. Machine-readable output uses
// writeMachineJSON, which reports write failures to callers.
func printTo(writer io.Writer, values ...any) {
	_, _ = fmt.Fprint(writer, values...)
}

func printfTo(writer io.Writer, format string, values ...any) {
	_, _ = fmt.Fprintf(writer, format, values...)
}

func printlnTo(writer io.Writer, values ...any) {
	_, _ = fmt.Fprintln(writer, values...)
}
