// Command repogrep imports, searches, and prunes metadata about git repos
// (starting with GitHub stars) using a local SQLite store.
package main

import (
	"os"

	"github.com/hueys/repogrep/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
