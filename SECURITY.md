# Security Policy

## Supported Versions

repogrep is pre-1.0 and released on an ad-hoc basis; only the latest
tagged release is supported. There isn't a backport policy yet.

## Reporting a Vulnerability

Please **do not** open a public issue for security vulnerabilities.

Instead, use GitHub's [private vulnerability
reporting](https://github.com/hueys/repogrep/security/advisories/new)
(Security tab → "Report a vulnerability"). This opens a private
discussion with the maintainer and, if applicable, lets us coordinate
a fix and a GitHub Security Advisory before any public disclosure.

If that's not workable for some reason, opening a normal issue with as
few exploit details as possible (and a note that it's security-related)
is a reasonable fallback.

## Scope

repogrep is a local CLI tool: it reads your GitHub token (via `gh auth
token` or `$GITHUB_TOKEN`), talks to the GitHub API, and stores data in
a local SQLite file. Reports about things like: token handling,
injection via GitHub-sourced content (repo names, descriptions,
READMEs, topics) reaching a sink unsafely, or CI/workflow supply-chain
issues (see `.github/workflows/`) are all in scope. Reports about the
inherent behavior of a local single-user tool run with the user's own
already-trusted credentials/flags/environment are generally not
security issues — see the threat model implied throughout this repo's
`/security-review` history if you want prior art on what's been
considered.
