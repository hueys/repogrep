package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hueys/repogrep/internal/model"
)

func printRepoTable(repos []model.Repo, jsonOut bool) {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(repos)
		return
	}
	if len(repos) == 0 {
		fmt.Println("no repos found")
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "REPO\tLANGUAGE\tSTARS\tUPDATED\tTOPICS")
	for _, r := range repos {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\n",
			sanitizeTerminal(r.FullName), sanitizeTerminal(orDash(r.PrimaryLanguage)), r.Stars,
			dateOrDash(r.PushedAt), sanitizeTerminal(strings.Join(r.Topics, ",")))
	}
	_ = tw.Flush()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func dateOrDash(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02")
}
