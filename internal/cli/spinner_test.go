package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestStartSpinnerDisabled(t *testing.T) {
	var out bytes.Buffer
	stop := startSpinner(false, &out, "Thinking")
	stop()
	if out.Len() != 0 {
		t.Fatalf("expected no output, got %q", out.String())
	}
}

func TestStartSpinnerRendersAndClears(t *testing.T) {
	prev := spinnerTickInterval
	spinnerTickInterval = 5 * time.Millisecond
	t.Cleanup(func() {
		spinnerTickInterval = prev
	})

	var out bytes.Buffer
	stop := startSpinner(true, &out, "Thinking")
	time.Sleep(25 * time.Millisecond)
	stop()

	got := out.String()
	if !strings.Contains(got, "Thinking") {
		t.Fatalf("spinner output missing label: %q", got)
	}
	if !strings.Contains(got, "\r") {
		t.Fatalf("spinner output missing carriage return: %q", got)
	}
	clearSeq := "\r" + strings.Repeat(" ", len("Thinking")+4) + "\r"
	if !strings.Contains(got, clearSeq) {
		t.Fatalf("spinner output missing clear sequence: %q", got)
	}
}
