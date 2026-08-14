// Package migrations embeds repogrep's numbered schema migration files.
package migrations

import "embed"

// FS holds the embedded *.sql migration files, applied in filename order.
//
//go:embed *.sql
var FS embed.FS
