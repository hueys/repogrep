package cli

import (
	"context"
	"flag"

	"github.com/hueys/repogrep/internal/config"
	"github.com/hueys/repogrep/internal/store"
)

func runList(ctx context.Context, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	cf := registerCommon(fs, cfg)
	lang := fs.String("lang", "", "filter by primary language")
	topic := fs.String("topic", "", "filter by topic")
	src := fs.String("source", "", "filter by source (e.g. github)")
	archived := fs.Bool("archived", false, "show only archived repos")
	sortBy := fs.String("sort", "", "sort by: stars, updated (default: name)")
	limit := fs.Int("limit", cfg.DefaultLimit, "max repos to show (0 = no limit)")
	jsonOut := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, err := openStore(cf)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	repos, err := st.List(store.ListFilters{
		Source:       *src,
		Language:     *lang,
		Topic:        *topic,
		ArchivedOnly: *archived,
		Sort:         *sortBy,
		Limit:        *limit,
	})
	if err != nil {
		return err
	}
	printRepoTable(repos, *jsonOut)
	return nil
}
