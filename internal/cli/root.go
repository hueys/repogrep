// Package cli implements repogrep's command-line dispatch using only the
// standard library's flag package: repogrep <command> [flags]. Each
// subcommand owns its own flag.FlagSet (via registerCommon for the shared
// --db/-v flags) so global options are passed after the subcommand name,
// e.g. `repogrep list --db path.db`.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/hueys/repogrep/internal/config"
	_ "github.com/hueys/repogrep/internal/source/github" // registers the "github" source
)

type command struct {
	name  string
	usage string
	run   func(ctx context.Context, cfg config.Config, args []string) error
}

func commands() []command {
	return []command{
		{"import", "Import repos from a source, e.g. `import github`", runImport},
		{"update", "Refresh repos from every source already imported", runUpdate},
		{"search", "Full-text search stored repos", runSearch},
		{"list", "List/filter stored repos", runList},
		{"show", "Show full detail for one repo", runShow},
		{"prune", "Find (and optionally remove) inactive/archived repos", runPrune},
	}
}

// Execute runs the CLI and returns a process exit code.
func Execute() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "repogrep:", err)
		return 1
	}

	cmds := commands()
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printUsage(cmds)
		return 0
	}

	name := args[0]
	for _, c := range cmds {
		if c.name != name {
			continue
		}
		if err := c.run(context.Background(), cfg, args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			fmt.Fprintln(os.Stderr, "repogrep:", err)
			return 1
		}
		return 0
	}

	fmt.Fprintf(os.Stderr, "repogrep: unknown command %q\n\n", name)
	printUsage(cmds)
	return 1
}

func printUsage(cmds []command) {
	fmt.Fprintln(os.Stderr, "Usage: repogrep <command> [flags]")
	fmt.Fprintln(os.Stderr, "\nCommands:")
	sorted := append([]command(nil), cmds...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].name < sorted[j].name })
	for _, c := range sorted {
		fmt.Fprintf(os.Stderr, "  %-8s %s\n", c.name, c.usage)
	}
	fmt.Fprintln(os.Stderr, "\nRun `repogrep <command> -h` for command-specific flags.")
}
