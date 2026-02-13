package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/sasanktumpati/ask/internal/config"
)

const version = "0.2.2"

func printHelp(w io.Writer, topic string, cfgPath string) {
	switch topic {
	case "", "root":
		printRootHelp(w, cfgPath)
	case "ask":
		printAskHelp(w)
	case "models", "model":
		printModelsHelp(w)
	case "provider", "providers":
		printProvidersHelp(w)
	case "key", "keys":
		printKeysHelp(w)
	case "config":
		printConfigHelp(w, cfgPath)
	case "markdown":
		printMarkdownHelp(w)
	default:
		fmt.Fprintf(w, "unknown help topic %q\n\n", topic)
		printRootHelp(w, cfgPath)
	}
}

func printRootHelp(w io.Writer, cfgPath string) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "ask v%s\n", version)
	fmt.Fprintln(tw, "Terminal LLM assistant with provider switching, model discovery, and command prefill.")
	fmt.Fprintln(tw)

	fmt.Fprintln(tw, "USAGE")
	fmt.Fprintln(tw, "  ask [global flags] \"question\" [ask flags]")
	fmt.Fprintln(tw, "  ask [global flags] <command> [flags]")
	fmt.Fprintln(tw)

	fmt.Fprintln(tw, "GLOBAL FLAGS")
	fmt.Fprintln(tw, "  -c, --config <path>\tconfig file path (or ASK_CONFIG)")
	fmt.Fprintln(tw, "  -h, --help\tshow help")
	fmt.Fprintln(tw, "  -v, --version\tshow version")
	fmt.Fprintln(tw)

	fmt.Fprintln(tw, "COMMANDS")
	fmt.Fprintln(tw, "  models\tlist/select/set provider models")
	fmt.Fprintln(tw, "  provider\tlist/show/set/add/remove providers")
	fmt.Fprintln(tw, "  key\tset/show/clear API keys")
	fmt.Fprintln(tw, "  config\tshow config and paths")
	fmt.Fprintln(tw, "  markdown\ttoggle markdown rendering")
	fmt.Fprintln(tw, "  help [topic]\tshow topic help")
	fmt.Fprintln(tw)

	fmt.Fprintln(tw, "EXAMPLES")
	fmt.Fprintln(tw, "  ask \"command to remove a commit from git\"")
	fmt.Fprintln(tw, "  ask -p ollama -m llama3.2 \"write a jq filter for this JSON\"")
	fmt.Fprintln(tw, "  ask models select --provider openai --search mini")
	fmt.Fprintln(tw, "  ask provider add myproxy --base-url https://llm.example.com/v1")
	fmt.Fprintln(tw)

	fmt.Fprintln(tw, "TOPICS")
	fmt.Fprintln(tw, "  ask help ask|models|provider|key|config|markdown")
	fmt.Fprintln(tw)

	fmt.Fprintln(tw, "CONFIG")
	fmt.Fprintf(tw, "  File:\t%s\n", cfgPath)
	fmt.Fprintf(tw, "  Template:\t%s\n", config.TemplatePathForConfig(cfgPath))
	fmt.Fprintln(tw, "  ASK_CONFIG_DIR:\tdefault config directory override")

	_ = tw.Flush()
}

func printAskHelp(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "USAGE")
	fmt.Fprintln(tw, "  ask \"question\" [options]")
	fmt.Fprintln(tw)
	fmt.Fprintln(tw, "OPTIONS")
	fmt.Fprintln(tw, "  -p, --provider <name>\tprovider to use")
	fmt.Fprintln(tw, "  -m, --model <id>\tmodel to use")
	fmt.Fprintln(tw, "  --timeout <dur|sec>\trequest timeout (default: 90s)")
	fmt.Fprintln(tw, "  --no-markdown\tdisable markdown rendering for this call")
	fmt.Fprintln(tw, "  --no-run\tprint returned command without run prompt")
	fmt.Fprintln(tw, "  --json\tprint structured JSON")
	fmt.Fprintln(tw)
	fmt.Fprintln(tw, "NOTES")
	fmt.Fprintln(tw, "  Response contract is JSON with keys: answer, command")
	fmt.Fprintln(tw, "  If command is present, ask prefills it so Enter runs it")
	_ = tw.Flush()
}

func printModelsHelp(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "USAGE")
	fmt.Fprintln(tw, "  ask models list [--provider <name>] [--search <text>]")
	fmt.Fprintln(tw, "  ask models select [--provider <name>] [--search <text>]")
	fmt.Fprintln(tw, "  ask models set <model> [--provider <name>]")
	fmt.Fprintln(tw, "  ask models current [--provider <name>]")
	fmt.Fprintln(tw)
	fmt.Fprintln(tw, "NOTES")
	fmt.Fprintln(tw, "  list/select always call provider model-list APIs (not hardcoded)")
	fmt.Fprintln(tw, "  select supports in-loop search using /text")
	_ = tw.Flush()
}

func printProvidersHelp(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "USAGE")
	fmt.Fprintln(tw, "  ask provider list")
	fmt.Fprintln(tw, "  ask provider current")
	fmt.Fprintln(tw, "  ask provider set <name>")
	fmt.Fprintln(tw, "  ask provider show [name]")
	fmt.Fprintln(tw, "  ask provider add <name> --base-url <url> [options]")
	fmt.Fprintln(tw, "  ask provider remove <name>")
	fmt.Fprintln(tw)
	fmt.Fprintln(tw, "ADD OPTIONS")
	fmt.Fprintln(tw, "  --model <id>\tdefault model for this provider")
	fmt.Fprintln(tw, "  --api-key <key>\tstore API key in config")
	fmt.Fprintln(tw, "  --api-key-env <ENV>\tenv var name for API key")
	fmt.Fprintln(tw, "  --models-path <path>\tdefault: /models")
	fmt.Fprintln(tw, "  --chat-path <path>\tdefault: /chat/completions")
	fmt.Fprintln(tw, "  --auth-header <name>\tdefault: Authorization")
	fmt.Fprintln(tw, "  --auth-prefix <text>\tdefault: Bearer ")
	fmt.Fprintln(tw, "  --header key=value\tadditional static headers (repeatable)")
	_ = tw.Flush()
}

func printKeysHelp(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "USAGE")
	fmt.Fprintln(tw, "  ask key set <provider> [--value <key>] [--env <ENV_VAR>]")
	fmt.Fprintln(tw, "  ask key show <provider>")
	fmt.Fprintln(tw, "  ask key clear <provider>")
	fmt.Fprintln(tw)
	fmt.Fprintln(tw, "NOTES")
	fmt.Fprintln(tw, "  key set without --value prompts for secret input")
	fmt.Fprintln(tw, "  or edit providers.<name>.api_key directly in config.json")
	fmt.Fprintln(tw, "  env var values take precedence over config api_key")
	_ = tw.Flush()
}

func printConfigHelp(w io.Writer, cfgPath string) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "USAGE")
	fmt.Fprintln(tw, "  ask config show")
	fmt.Fprintln(tw, "  ask config path")
	fmt.Fprintln(tw, "  ask config template")
	fmt.Fprintln(tw)
	fmt.Fprintln(tw, "PATHS")
	fmt.Fprintf(tw, "  Config:\t%s\n", cfgPath)
	fmt.Fprintf(tw, "  Template:\t%s\n", config.TemplatePathForConfig(cfgPath))
	_ = tw.Flush()
}

func printMarkdownHelp(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "USAGE")
	fmt.Fprintln(tw, "  ask markdown on")
	fmt.Fprintln(tw, "  ask markdown off")
	fmt.Fprintln(tw, "  ask markdown status")
	_ = tw.Flush()
}
