package cli

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

var spinnerTickInterval = 120 * time.Millisecond

var spinnerFrames = []rune{'|', '/', '-', '\\'}

func startSpinner(enabled bool, w io.Writer, label string) func() {
	if !enabled || w == nil {
		return func() {}
	}

	label = strings.TrimSpace(label)
	if label == "" {
		label = "Loading"
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		frame := 0
		ticker := time.NewTicker(spinnerTickInterval)
		defer ticker.Stop()

		render := func() {
			fmt.Fprintf(w, "\r%c %s", spinnerFrames[frame%len(spinnerFrames)], label)
			frame++
		}

		render()
		for {
			select {
			case <-done:
				clearLen := len(label) + 4
				fmt.Fprintf(w, "\r%s\r", strings.Repeat(" ", clearLen))
				return
			case <-ticker.C:
				render()
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			wg.Wait()
		})
	}
}

func isTerminalWriter(w io.Writer) bool {
	fdw, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	return term.IsTerminal(int(fdw.Fd()))
}
