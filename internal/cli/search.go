package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hueys/repogrep/internal/config"
	"github.com/hueys/repogrep/internal/store"
)

func runSearch(ctx context.Context, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: repogrep search <query> [flags]")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	cf := registerCommon(fs, cfg)
	lang := fs.String("lang", "", "filter by primary language")
	topic := fs.String("topic", "", "filter by topic")
	includeArchived := fs.Bool("include-archived", false, "include archived repos (excluded by default)")
	limit := fs.Int("limit", cfg.DefaultLimit, "max repos to show (0 = no limit)")
	jsonOut := fs.Bool("json", false, "output as JSON")

	if len(args) == 0 {
		fs.Usage()
		return fmt.Errorf("missing search query")
	}
	// Everything before the first flag (i.e. not starting with "-") is the
	// query; flag.Parse stops at the first non-flag arg, so collect query
	// words up front and parse flags from the remainder.
	var queryWords []string
	i := 0
	for ; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			break
		}
		queryWords = append(queryWords, args[i])
	}
	if err := fs.Parse(args[i:]); err != nil {
		return err
	}
	if len(queryWords) == 0 {
		fs.Usage()
		return fmt.Errorf("missing search query")
	}
	query := strings.Join(queryWords, " ")

	st, err := openStore(cf)
	if err != nil {
		return err
	}
	defer st.Close()

	repos, err := st.Search(query, store.SearchFilters{
		Language:        *lang,
		Topic:           *topic,
		IncludeArchived: *includeArchived,
		Limit:           *limit,
	})
	if err != nil {
		return err
	}
	printRepoTable(repos, *jsonOut)
	return nil
}
