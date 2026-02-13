package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/chzyer/readline"
	"golang.org/x/term"
)

// RunOptions controls command prefill behavior and IO streams.
type RunOptions struct {
	Command string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// PromptAndRun presents an editable shell prompt prefilled with Command.
// Enter executes the command, Ctrl+C copies it to clipboard and exits,
// and Ctrl+D exits without execution.
func PromptAndRun(opts RunOptions) error {
	cmd := strings.TrimSpace(opts.Command)
	if cmd == "" {
		return nil
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	fmt.Fprintln(opts.Stdout)

	cfg := &readline.Config{
		Prompt:          "$ ",
		InterruptPrompt: "\n",
		EOFPrompt:       "\n",
		Stdout:          opts.Stdout,
		Stderr:          opts.Stderr,
	}
	if in, ok := opts.Stdin.(io.ReadCloser); ok {
		cfg.Stdin = in
	} else if opts.Stdin != nil {
		cfg.Stdin = io.NopCloser(opts.Stdin)
	}

	rl, err := readline.NewEx(cfg)
	if err != nil {
		return fmt.Errorf("init command prompt: %w", err)
	}
	defer rl.Close()

	input, err := rl.ReadlineWithDefault(cmd)
	if err == readline.ErrInterrupt {
		clearPromptLine(opts.Stdout)
		fmt.Fprintln(opts.Stdout)
		if copyErr := copyToClipboard(cmd, runtime.GOOS); copyErr == nil {
			fmt.Fprintln(opts.Stdout, "Command copied to clipboard.")
		} else {
			fmt.Fprintln(opts.Stdout, "Command cancelled.")
		}
		return nil
	}
	if err == io.EOF {
		clearPromptLine(opts.Stdout)
		return nil
	}
	if err != nil {
		return fmt.Errorf("read command input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		input = cmd
	}

	shell := strings.TrimSpace(os.Getenv("SHELL"))
	if shell == "" {
		shell = "sh"
	}

	execCmd := exec.Command(shell, "-lc", input)
	execCmd.Stdout = opts.Stdout
	execCmd.Stderr = opts.Stderr
	if stdin, ok := opts.Stdin.(*os.File); ok {
		execCmd.Stdin = stdin
	} else {
		execCmd.Stdin = os.Stdin
	}

	return execCmd.Run()
}

func clearPromptLine(w io.Writer) {
	if !isTerminalWriter(w) {
		return
	}
	fmt.Fprint(w, "\r\033[2K\r")
}

func isTerminalWriter(w io.Writer) bool {
	fdw, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	return term.IsTerminal(int(fdw.Fd()))
}

type clipboardCmd struct {
	name string
	args []string
}

func copyToClipboard(text string, goos string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("clipboard text is empty")
	}

	for _, c := range clipboardCommands(goos) {
		cmd := exec.Command(c.name, c.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return errors.New("no working clipboard command found")
}

func clipboardCommands(goos string) []clipboardCmd {
	switch goos {
	case "darwin":
		return []clipboardCmd{{name: "pbcopy"}}
	case "windows":
		return []clipboardCmd{{name: "cmd", args: []string{"/c", "clip"}}}
	default:
		return []clipboardCmd{
			{name: "wl-copy"},
			{name: "xclip", args: []string{"-selection", "clipboard"}},
			{name: "xsel", args: []string{"--clipboard", "--input"}},
		}
	}
}
