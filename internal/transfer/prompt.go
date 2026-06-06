package transfer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

var (
	// promptMu serializes interactive prompts: discovery may start several
	// transfers concurrently, and without it their questions interleave and
	// race for the same stdin input.
	promptMu sync.Mutex
	stdin    io.Reader // nil means os.Stdin, looked up on first use
	reader   *bufio.Reader
)

// confirm prints details followed by question and reports whether the user
// answered "y" or "Y". The whole exchange holds a lock so concurrent prompts
// are shown and answered one at a time.
func confirm(details, question string) bool {
	promptMu.Lock()
	defer promptMu.Unlock()

	if reader == nil {
		in := stdin
		if in == nil {
			in = os.Stdin
		}
		reader = bufio.NewReader(in)
	}
	fmt.Print(details)
	fmt.Print(question)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(line), "y")
}
