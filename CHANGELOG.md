# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

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

[v0.2.0]: https://github.com/hueys/repogrep/releases/tag/v0.2.0
[v0.1.0]: https://github.com/hueys/repogrep/releases/tag/v0.1.0
