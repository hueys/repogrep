# repogrep

A local CLI tool that imports metadata and README content from git repos
(starting with your GitHub stars) into a single SQLite file, then lets you
full-text search and prune them.

## Build

```sh
make build   # embeds a version from `git describe`; see `repogrep -V`
# or, without a version stamp:
go build -o repogrep ./cmd/repogrep
```

## Usage

```sh
# Import (or re-import) everything you've starred on GitHub. Uses `gh auth
# token` for credentials by default; falls back to $GITHUB_TOKEN.
repogrep import github

# Refresh every source already imported (currently just github).
repogrep update

# Browse / filter what's stored.
repogrep list --lang Go --topic cli --sort stars

# Full-text search over README + description + topics.
repogrep search "retrieval augmented generation" --lang Python

# Full detail (and README) for one repo.
repogrep show owner/name --readme

# Find archived/inactive repos (dry-run by default).
repogrep prune
repogrep prune --months 6 --force            # prompts before deleting
repogrep prune --months 6 --force --yes      # no prompt, for scripting
repogrep prune --months 6 --force --unstar   # also unstar on GitHub, not just delete locally
```

By default `prune` only removes rows from the local database — it never
touches GitHub. Add `--unstar` (alongside `--force`) to also remove the
star at the source, for sources that support it.

All commands accept `--db PATH` (default `~/.config/repogrep/repogrep.db`,
or `$XDG_CONFIG_HOME/repogrep/repogrep.db` if set) and
`-v` for verbose output. Run `repogrep <command> -h` for full flag lists.

## Data

Everything lives in one SQLite file: structured metadata in `repos`
(+`topics`), full-text search via an FTS5 index over
name/description/README/topics. No repos are cloned — only metadata and
README content are fetched via the GitHub API.

## Adding another source

Ingestion is pluggable via the `source.Source` interface
(`internal/source/source.go`). A new backend implements `Fetch` and
registers itself by name in an `init()`, e.g. see
`internal/source/github`. Nothing else needs to change — `import <name>`
and `update` pick it up automatically.

## License

[MIT](LICENSE)
