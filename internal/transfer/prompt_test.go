package transfer

import (
	"strings"
	"sync"
	"testing"
)

func TestConfirmConcurrentPromptsEachGetOneAnswer(t *testing.T) {
	stdin, reader = strings.NewReader("y\nN\n\n"), nil
	t.Cleanup(func() { reader = nil })

	var mu sync.Mutex
	var yes, no int
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok := confirm("", "? ")
			mu.Lock()
			defer mu.Unlock()
			if ok {
				yes++
			} else {
				no++
			}
		}()
	}
	wg.Wait()
	if yes != 1 || no != 2 {
		t.Errorf("got %d yes / %d no, want 1 / 2", yes, no)
	}
}

func TestConfirmEOF(t *testing.T) {
	stdin, reader = strings.NewReader(""), nil
	t.Cleanup(func() { reader = nil })
	if confirm("", "? ") {
		t.Error("confirm on EOF = true, want false")
	}
}
