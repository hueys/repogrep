package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hueys/repogrep/internal/config"
)

func runShow(ctx context.Context, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: repogrep show <owner/name> [flags]")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	if len(args) == 0 {
		fs.Usage()
		return fmt.Errorf("missing owner/name")
	}
	fullName := args[0]

	cf := registerCommon(fs, cfg)
	showReadme := fs.Bool("readme", false, "print the full README content")
	jsonOut := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	st, err := openStore(cf)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	repo, err := st.FindByFullName(fullName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("no repo named %q in the store", fullName)
		}
		return err
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(repo)
	}

	fmt.Printf("%s\n", repo.FullName)
	fmt.Printf("  source:      %s\n", repo.Source)
	fmt.Printf("  url:         %s\n", repo.URL)
	if repo.Description != "" {
		fmt.Printf("  description: %s\n", repo.Description)
	}
	fmt.Printf("  language:    %s\n", orDash(repo.PrimaryLanguage))
	fmt.Printf("  stars:       %d\n", repo.Stars)
	fmt.Printf("  topics:      %s\n", strings.Join(repo.Topics, ", "))
	fmt.Printf("  archived:    %t\n", repo.IsArchived)
	fmt.Printf("  pushed:      %s\n", dateOrDash(repo.PushedAt))
	fmt.Printf("  first seen:  %s\n", dateOrDash(repo.FirstSeen))
	fmt.Printf("  last synced: %s\n", dateOrDash(repo.LastSynced))

	if *showReadme {
		fmt.Println("\n--- README ---")
		if repo.README == "" {
			fmt.Println("(no README)")
		} else {
			fmt.Println(repo.README)
		}
	}
	return nil
}
