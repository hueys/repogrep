// Command repogrep imports, searches, and prunes metadata about git repos
// (starting with GitHub stars) using a local SQLite store.
package main

import (
	"os"

	"github.com/hueys/repogrep/internal/cli"
)

// version is overridden at build time via:
//
//	go build -ldflags "-X main.version=$(git describe --tags --always --dirty)"
//
// See the Makefile's `build` target.
var version = "dev"

func main() {
	os.Exit(cli.Execute(version))
}
