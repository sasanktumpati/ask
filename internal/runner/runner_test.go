package runner

import "testing"

func TestClipboardCommandsByOS(t *testing.T) {
	mac := clipboardCommands("darwin")
	if len(mac) != 1 || mac[0].name != "pbcopy" {
		t.Fatalf("darwin clipboard commands = %+v", mac)
	}

	win := clipboardCommands("windows")
	if len(win) != 1 || win[0].name != "cmd" {
		t.Fatalf("windows clipboard commands = %+v", win)
	}
	if len(win[0].args) != 2 || win[0].args[0] != "/c" || win[0].args[1] != "clip" {
		t.Fatalf("windows clipboard args = %+v", win[0].args)
	}

	linux := clipboardCommands("linux")
	if len(linux) < 3 {
		t.Fatalf("linux clipboard commands = %+v", linux)
	}
}

func TestCopyToClipboardRejectsEmpty(t *testing.T) {
	if err := copyToClipboard("   ", "darwin"); err == nil {
		t.Fatal("expected error for empty clipboard text")
	}
}
