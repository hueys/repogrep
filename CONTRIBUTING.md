# Contributing

Thanks for taking a look at repogrep. This is a small personal tool, so
the process is intentionally lightweight.

## Getting set up

Requires Go 1.26+ (see `go.mod`). No other tools are required to build
and test; `make lint` locally needs
[`golangci-lint`](https://golangci-lint.run) and
[`zizmor`](https://docs.zizmor.sh) (the latter audits
`.github/workflows/` for GitHub Actions security issues, e.g. unpinned
action references). Neither is required to build/test — CI runs both
either way.

```sh
git clone https://github.com/hueys/repogrep.git
cd repogrep
make build
make test
```

## Making changes

- Branch off `main` (`git checkout -b my-change`); don't commit
  directly to `main`.
- Keep changes focused — one logical change per PR is easier to review
  than a mix of unrelated fixes.
- Run before opening a PR:
  ```sh
  make fmt
  go vet ./...
  make lint
  make test
  ```
  CI runs the same checks (`.github/workflows/ci.yml`) on every push
  and PR.
- Add or update tests alongside behavior changes. See the existing
  `*_test.go` files for the patterns used here: real temp-file SQLite
  for `internal/store`, an `httptest.Server`-backed client for
  `internal/source/github`, table-driven tests for pure functions.
- Match the existing code style — this codebase favors small,
  single-purpose files and doc comments on exported identifiers over
  broader Go conventions debates.

## Adding a new ingestion source

Ingestion is pluggable via the `source.Source` interface
(`internal/source/source.go`) — see the README's "Adding another
source" section and `internal/source/github` for a concrete example.
A new source needs its own package, an implementation of `Fetch`
(and optionally `Unstar`, if the source has an equivalent concept),
and a `source.Register(...)` call in `init()`. Nothing else in the
CLI or storage layer needs to change.

## Reporting issues

Open a [GitHub issue](https://github.com/hueys/repogrep/issues) with
what you ran, what you expected, and what happened instead. Include
`repogrep -V` output if it's version-specific.
