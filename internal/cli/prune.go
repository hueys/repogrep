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
	gsource "github.com/hueys/repogrep/internal/source"
	"github.com/hueys/repogrep/internal/store"
)

func runPrune(ctx context.Context, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("prune", flag.ContinueOnError)
	cf := registerCommon(fs, cfg)
	months := fs.Int("months", cfg.PruneInactiveMonths, "flag repos with no push activity in N months")
	force := fs.Bool("force", false, "actually delete flagged repos (default: dry-run only)")
	yes := fs.Bool("yes", false, "skip the confirmation prompt (requires --force)")
	unstar := fs.Bool("unstar", false, "also unstar at the origin (e.g. GitHub) when deleting; requires --force")
	jsonOut := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateUnstarFlag(*unstar, *force); err != nil {
		return err
	}

	st, err := openStore(cf)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

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
		_, _ = fmt.Fprintln(tw, "REPO\tSTARS\tUPDATED\tREASON")
		for _, c := range candidates {
			_, _ = fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", sanitizeTerminal(c.Repo.FullName), c.Repo.Stars, dateOrDash(c.Repo.PushedAt), c.Reason)
		}
		_ = tw.Flush()
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

	if *unstar {
		unstarCandidates(ctx, candidates, cf.verbose)
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

func validateUnstarFlag(unstar, force bool) error {
	if unstar && !force {
		return fmt.Errorf("--unstar requires --force")
	}
	return nil
}

// unstarCandidates best-effort unstars each candidate at its origin source
// (skipping, with a warning, any source that doesn't implement
// source.Unstarer). Failures are reported to stderr but don't block the
// local delete that follows — pruning stays effective even if a given
// unstar call fails.
func unstarCandidates(ctx context.Context, candidates []store.PruneCandidate, verbose bool) {
	type resolved struct {
		unstarer gsource.Unstarer
		err      error
	}
	resolvedBySource := map[string]resolved{}

	for _, c := range candidates {
		r, ok := resolvedBySource[c.Repo.Source]
		if !ok {
			src, err := gsource.New(c.Repo.Source)
			switch {
			case err != nil:
				r = resolved{err: err}
			default:
				u, isUnstarer := src.(gsource.Unstarer)
				if !isUnstarer {
					r = resolved{err: fmt.Errorf("source %q does not support unstarring", c.Repo.Source)}
				} else {
					r = resolved{unstarer: u}
				}
			}
			resolvedBySource[c.Repo.Source] = r
		}

		if r.err != nil {
			fmt.Fprintf(os.Stderr, "repogrep: %s left starred: %v\n", sanitizeTerminal(c.Repo.FullName), r.err)
			continue
		}
		if err := r.unstarer.Unstar(ctx, c.Repo.Owner, c.Repo.Name); err != nil {
			fmt.Fprintf(os.Stderr, "repogrep: unstar %s: %v\n", sanitizeTerminal(c.Repo.FullName), err)
			continue
		}
		if verbose {
			fmt.Printf("unstarred %s\n", sanitizeTerminal(c.Repo.FullName))
		}
	}
}
