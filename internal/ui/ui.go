package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

var (
	Red, Green, Yellow, Blue, Cyan, Bold, Dim, NC string
)

const (
	Check     = "✓"
	Cross     = "✗"
	Arrow     = "→"
	SyncGlyph = "⟳"
)

func init() {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		Red = "\033[0;31m"
		Green = "\033[0;32m"
		Yellow = "\033[0;33m"
		Blue = "\033[0;34m"
		Cyan = "\033[0;36m"
		Bold = "\033[1m"
		Dim = "\033[2m"
		NC = "\033[0m"
	}
}

func Die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s%s %s%s\n", Red, Cross, fmt.Sprintf(format, args...), NC)
	os.Exit(1)
}
