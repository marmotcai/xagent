package cmd

import (
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/trie"
	"github.com/minio/minio/pkg/words"
	"github.com/minio/cli"

	_ "github.com/spf13/viper"
	"os"
	"path/filepath"
	"sort"
)
// global flags for xagent.
var GlobalFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "config-dir, C",
		Value: defaultConfigDir.Get(),
		Usage: "[DEPRECATED] Path to legacy configuration directory.",
	},
	cli.StringFlag{
		Name:  "certs-dir, S",
		Value: defaultCertsDir.Get(),
		Usage: "Path to certs directory.",
	},
	cli.BoolFlag{
		Name:  "quiet",
		Usage: "Disable startup information.",
	},
	cli.BoolFlag{
		Name:  "anonymous",
		Usage: "Hide sensitive information from logging.",
	},
	cli.BoolFlag{
		Name:  "json",
		Usage: "Output server logs and startup information in json format.",
	},
}

// Help template for xagent.
var xagentHelpTemplate = `NAME:
  {{.Name}} - {{.Usage}}

DESCRIPTION:
  {{.Description}}

USAGE:
  {{.HelpName}} {{if .VisibleFlags}}[FLAGS] {{end}}COMMAND{{if .VisibleFlags}}{{end}} [ARGS...]

COMMANDS:
  {{range .VisibleCommands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
  {{end}}{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
VERSION:
  ` + Version +
	`{{ "\n"}}`

func newApp(name string) *cli.App {
	// Collection of xagent commands currently supported are.
	commands := []cli.Command{}

	// Collection of xagent commands currently supported in a trie tree.
	commandsTree := trie.NewTrie()

	// registerCommand registers a cli command.
	registerCommand := func(command cli.Command) {
		commands = append(commands, command)
		commandsTree.Insert(command.Name)
	}

	findClosestCommands := func(command string) []string {
		var closestCommands []string
		for _, value := range commandsTree.PrefixMatch(command) {
			closestCommands = append(closestCommands, value.(string))
		}

		sort.Strings(closestCommands)
		// Suggest other close commands - allow missed, wrongly added and
		// even transposed characters
		for _, value := range commandsTree.Walk(commandsTree.Root()) {
			if sort.SearchStrings(closestCommands, value.(string)) < len(closestCommands) {
				continue
			}
			// 2 is arbitrary and represents the max
			// allowed number of typed errors
			if words.DamerauLevenshteinDistance(command, value.(string)) < 2 {
				closestCommands = append(closestCommands, value.(string))
			}
		}

		return closestCommands
	}

	// Register all commands.
	registerCommand(serverCmd)
	registerCommand(proxyCmd)
/*	registerCommand(gatewayCmd)
	registerCommand(updateCmd)
	registerCommand(versionCmd)
*/
	// Set up app.
	cli.HelpFlag = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Show help.",
	}

	app := cli.NewApp()
	app.Name = name
	app.Author = "Atoml, Inc."
	app.Version = Version
	app.Usage = "XAgent Server."
	app.Description = `XAgent is an agent server.`
	app.Flags = GlobalFlags
	app.HideVersion = true     // Hide `--version` flag, we already have `xagent version`.
	// app.HideHelpCommand = true // Hide `help, h` command, we already have `xagent --help`.
	app.Commands = commands
	app.CustomAppHelpTemplate = xagentHelpTemplate
	app.CommandNotFound = func(ctx *cli.Context, command string) {
		console.Printf("‘%s’ is not a xagent sub-command. See ‘xagent --help’.\n", command)
		closestCommands := findClosestCommands(command)
		if len(closestCommands) > 0 {
			console.Println()
			console.Println("Did you mean one of these?")
			for _, cmd := range closestCommands {
				console.Printf("\t‘%s’\n", cmd)
			}
		}

		os.Exit(1)
	}

	return app
}


// Main main for xagent server.
func Main(args []string) {
	// Set the xagent app name.
	appName := filepath.Base(args[0])

	// Run the app - exit on error.
	if err := newApp(appName).Run(args); err != nil {
		os.Exit(1)
	}
}
