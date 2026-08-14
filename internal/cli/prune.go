package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/hueys/repogrep/internal/config"
)

func runPrune(ctx context.Context, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("prune", flag.ContinueOnError)
	cf := registerCommon(fs, cfg)
	months := fs.Int("months", cfg.PruneInactiveMonths, "flag repos with no push activity in N months")
	force := fs.Bool("force", false, "actually delete flagged repos (default: dry-run only)")
	yes := fs.Bool("yes", false, "skip the confirmation prompt (requires --force)")
	jsonOut := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, err := openStore(cf)
	if err != nil {
		return err
	}
	defer st.Close()

	candidates, err := st.PruneCandidates(*months)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		fmt.Println("no prune candidates")
		return nil
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(candidates); err != nil {
			return err
		}
	} else {
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "REPO\tSTARS\tUPDATED\tREASON")
		for _, c := range candidates {
			fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", c.Repo.FullName, c.Repo.Stars, dateOrDash(c.Repo.PushedAt), c.Reason)
		}
		tw.Flush()
	}

	if !*force {
		fmt.Printf("\n%d repo(s) flagged (dry-run; pass --force to delete)\n", len(candidates))
		return nil
	}

	if !*yes {
		fmt.Printf("\nDelete %d repo(s)? [y/N] ", len(candidates))
		reader := bufio.NewReader(os.Stdin)
		resp, _ := reader.ReadString('\n')
		resp = strings.ToLower(strings.TrimSpace(resp))
		if resp != "y" && resp != "yes" {
			fmt.Println("aborted, nothing deleted")
			return nil
		}
	}

	ids := make([]int64, len(candidates))
	for i, c := range candidates {
		ids[i] = c.Repo.ID
	}
	n, err := st.Delete(ids)
	if err != nil {
		return err
	}
	fmt.Printf("deleted %d repo(s)\n", n)
	return nil
}
