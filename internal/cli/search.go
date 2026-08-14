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
	sortBy := fs.String("sort", "", "sort by: relevance (default), stars, updated")
	limit := fs.Int("limit", cfg.DefaultLimit, "max repos to show (0 = no limit)")
	jsonOut := fs.Bool("json", false, "output as JSON")

	if len(args) == 0 {
		fs.Usage()
		return fmt.Errorf("missing search query")
	}
	queryWords, flagArgs := splitSearchArgs(args)
	if err := fs.Parse(flagArgs); err != nil {
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
	defer func() { _ = st.Close() }()

	repos, err := st.Search(query, store.SearchFilters{
		Language:        *lang,
		Topic:           *topic,
		IncludeArchived: *includeArchived,
		Sort:            *sortBy,
		Limit:           *limit,
	})
	if err != nil {
		return err
	}
	printRepoTable(repos, *jsonOut)
	return nil
}

// splitSearchArgs splits args into the free-form query words (everything up
// to the first arg starting with "-") and the remaining flag arguments.
// flag.Parse stops at the first non-flag arg, so the query has to be
// collected up front rather than left to fs.Parse/fs.Args.
func splitSearchArgs(args []string) (queryWords, flagArgs []string) {
	i := 0
	for ; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			break
		}
		queryWords = append(queryWords, args[i])
	}
	return queryWords, args[i:]
}
