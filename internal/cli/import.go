package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hueys/repogrep/internal/config"
	gsource "github.com/hueys/repogrep/internal/source"
)

func runImport(ctx context.Context, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	cf := registerCommon(fs, cfg)
	user := fs.String("user", "", "username to import stars for (default: authenticated user)")
	limit := fs.Int("limit", 0, "limit number of repos imported (0 = no limit)")
	concurrency := fs.Int("concurrency", cfg.ImportConcurrency, "concurrent per-repo fetches (e.g. READMEs)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: repogrep import <source> [flags]")
		fmt.Fprintln(os.Stderr, "\nAvailable sources:", gsource.Names())
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	if len(args) == 0 {
		fs.Usage()
		return fmt.Errorf("missing source name")
	}
	sourceName := args[0]

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	src, err := gsource.New(sourceName)
	if err != nil {
		return err
	}

	st, err := openStore(cf)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	results, err := src.Fetch(ctx, gsource.FetchOptions{
		Username:    *user,
		Limit:       *limit,
		Concurrency: *concurrency,
	})
	if err != nil {
		return fmt.Errorf("fetch from %s: %w", sourceName, err)
	}

	added, updated, errored := ingest(st, results, cf.verbose)
	fmt.Printf("import %s: %d added, %d updated, %d errored\n", sourceName, added, updated, errored)
	if errored > 0 {
		return fmt.Errorf("%d repo(s) failed to import", errored)
	}
	return nil
}

// ingest drains results into st, printing per-repo errors to stderr and
// (if verbose) a line per successfully stored repo. Returns summary counts.
func ingest(st storeUpserter, results <-chan gsource.FetchResult, verbose bool) (added, updated, errored int) {
	for res := range results {
		if res.Err != nil {
			errored++
			fmt.Fprintf(os.Stderr, "repogrep: %s: %v\n", sanitizeTerminal(res.Record.FullName), res.Err)
			continue
		}
		now := time.Now()
		_, inserted, err := st.UpsertRepo(recordToRepo(res.Record, now))
		if err != nil {
			errored++
			fmt.Fprintf(os.Stderr, "repogrep: store %s: %v\n", sanitizeTerminal(res.Record.FullName), err)
			continue
		}
		if inserted {
			added++
		} else {
			updated++
		}
		if verbose {
			tag := "~"
			if inserted {
				tag = "+"
			}
			fmt.Printf("%s %s\n", tag, sanitizeTerminal(res.Record.FullName))
		}
	}
	return added, updated, errored
}
