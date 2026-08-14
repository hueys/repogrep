package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/hueys/repogrep/internal/config"
	gsource "github.com/hueys/repogrep/internal/source"
)

// runUpdate re-runs Fetch for every source already present in the store
// (no persisted per-source state is needed: FetchOptions.Username == ""
// means "the source's default identity").
func runUpdate(ctx context.Context, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	cf := registerCommon(fs, cfg)
	concurrency := fs.Int("concurrency", cfg.ImportConcurrency, "concurrent per-repo fetches (e.g. READMEs)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, err := openStore(cf)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	sources, err := st.ListSources()
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		fmt.Println("nothing to update: no repos imported yet (try `repogrep import github`)")
		return nil
	}

	var totalAdded, totalUpdated, totalErrored int
	for _, name := range sources {
		src, err := gsource.New(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repogrep: skip source %q: %v\n", name, err)
			continue
		}
		results, err := src.Fetch(ctx, gsource.FetchOptions{Concurrency: *concurrency})
		if err != nil {
			fmt.Fprintf(os.Stderr, "repogrep: fetch %q: %v\n", name, err)
			continue
		}
		added, updated, errored := ingest(st, results, cf.verbose)
		totalAdded += added
		totalUpdated += updated
		totalErrored += errored
	}

	fmt.Printf("update: %d added, %d updated, %d errored\n", totalAdded, totalUpdated, totalErrored)
	if totalErrored > 0 {
		return fmt.Errorf("%d repo(s) failed to update", totalErrored)
	}
	return nil
}
