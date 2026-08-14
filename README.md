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

## Authentication

repogrep needs a GitHub token to list your stars and fetch READMEs. It
looks for one in this order and uses the first it finds:

1. **The `gh` CLI** ([cli.github.com](https://cli.github.com)), if
   installed and logged in — repogrep runs `gh auth token` for you.
   This is the easiest path if you already use `gh`; nothing else to
   configure.
   ```sh
   gh auth login
   ```
2. **`$GITHUB_TOKEN`** — a [personal access
   token](https://github.com/settings/tokens), if `gh` isn't installed
   or isn't logged in.
   ```sh
   export GITHUB_TOKEN=ghp_...
   ```

`gh` itself is **not required** — it's just the more convenient option
when it's already set up. Either way, the token needs read access to
whatever repos you've starred; add the `repo` scope if any of your
stars are private, or you use `prune --unstar` (unstarring needs the
same scope as reading private repos). `gh auth login`'s default scopes
already cover this.

If neither is found, commands that talk to GitHub fail with:
```
repogrep: no GitHub token found: run `gh auth login` or set GITHUB_TOKEN
```

## Usage

```sh
# Import (or re-import) everything you've starred on GitHub.
repogrep import github

# Refresh every source already imported (currently just github).
repogrep update

# Browse / filter what's stored.
repogrep list --lang Go --topic cli --sort stars

# Full-text search over README + description + topics.
repogrep search "retrieval augmented generation" --lang Python
repogrep search "vector database" --sort stars   # relevance is the default

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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md) for how to report a vulnerability.

## Changelog

See [CHANGELOG.md](CHANGELOG.md).

## License

[MIT](LICENSE)
