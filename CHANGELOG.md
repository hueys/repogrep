# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [v0.2.1] - 2026-08-14

### Security
- Sanitize GitHub-sourced text (description, README, topics, full name)
  before printing it to the terminal in `show`, `list`, `search`,
  `prune`, verbose `import`/`update`, and `--unstar` output. Repo
  metadata comes straight from the GitHub API — anyone who can create a
  public repo controls this text — and was previously printed
  completely unmodified, so a repo with raw ANSI/OSC escape sequences
  in its description or README could spoof the terminal title, hide or
  overwrite prior output, or disguise a phishing hyperlink when
  viewed. `--json` output was unaffected. Found via `/security-review`.

### Added
- CI (GitHub Actions): build/vet/test and a lint job (`golangci-lint`
  + `zizmor`) on every push to `main` and every PR.
- `zizmor` added to `make lint`, auditing `.github/workflows/` for
  GitHub Actions supply-chain issues (unpinned action references,
  missing `persist-credentials: false`); both were found and fixed in
  the workflow itself.
- `CONTRIBUTING.md`.
- Tests for `internal/config` (`Default`/`ConfigDir`/`Load`, including
  `XDG_CONFIG_HOME` precedence and YAML/env override behavior) and for
  `search`'s query-vs-flags argument parsing.

### Changed
- README: clarified GitHub authentication requirements (`gh` CLI is
  convenient but not required; `$GITHUB_TOKEN` works standalone) and
  the token scope needed for private stars / `--unstar`.

## [v0.2.0] - 2026-08-14

### Added
- `prune --unstar`: optionally remove the star at the origin (currently
  GitHub) when deleting a repo, in addition to the local database row.
  Requires `--force`. Implemented via a new `source.Unstarer` capability
  interface, so sources without an equivalent concept (e.g. a future
  curated list or local-disk source) simply don't implement it and are
  skipped with a warning rather than failing.

## [v0.1.0] - 2026-08-14

Initial release.

### Added
- `import github`: import the authenticated user's starred GitHub repos
  (metadata + README, via the API — no cloning) into a local SQLite
  database.
- `update`: re-run import for every source already present in the
  database.
- `search`: full-text search (SQLite FTS5 + bm25 ranking) over repo
  name, description, README, and topics, with `--lang`/`--topic`
  filters and archived repos excluded by default.
- `list`: browse/filter stored repos by language, topic, source, or
  archived status.
- `show`: full detail (and optional README) for a single repo.
- `prune`: find archived or inactive repos (configurable inactivity
  threshold); dry-run by default, requires `--force` (and `--yes` to
  skip the confirmation prompt) to actually delete.
- Pluggable ingestion via a `source.Source` interface + registry, so
  non-GitHub sources can be added later without touching storage or
  CLI code.
- `repogrep -V` / `--version` / `version` to print the build version.
- Optional `~/.config/repogrep/config.yaml` for persistent defaults
  (db path, prune threshold, list/search limit, import concurrency).
- `Makefile` (`build`, `clean`, `lint`, `fmt`, `test`) and a
  repo-scoped `.golangci.yml`.
- MIT license.

[v0.2.1]: https://github.com/hueys/repogrep/releases/tag/v0.2.1
[v0.2.0]: https://github.com/hueys/repogrep/releases/tag/v0.2.0
[v0.1.0]: https://github.com/hueys/repogrep/releases/tag/v0.1.0
