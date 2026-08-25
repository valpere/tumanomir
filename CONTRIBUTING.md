# Contributing

tumanomir is a solo-maintained project, but issues, bug reports, and pull
requests are welcome.

## Before opening an issue

Check the [Troubleshooting](docs/user-guide.en.md#8-troubleshooting) section
of the user guide first — most "why doesn't X work" questions are answered
there (missing instrument, prompt/context-window sizing, corpus format
errors, etc).

For bug reports, include:
- `tumanomir` version (`tumanomir version`)
- The command you ran and its full output
- Go version (`go version`) and OS

## Development setup

```bash
git clone https://github.com/valpere/tumanomir
cd tumanomir
make build     # -> bin/tumanomir
make test      # go test ./...
make vet       # go vet ./...
make lint      # golangci-lint run (requires golangci-lint installed)
make dogfood   # bin/tumanomir check docs/requirements.md — dogfood smoke test
make ci        # build + vet + test + lint + dogfood, all together
```

Requires Go >= 1.26. For `measure`/`gate`/`calibrate --instrument`-related
work, a local [Ollama](https://ollama.com/download) install — tests that
don't touch `internal/instrument` run without it.

## Making a change

1. Fork the repo and branch from `main`.
2. Keep changes focused — one logical change per PR. Match the existing
   style in the package you're touching; see `docs/architecture.md` and
   `docs/requirements.md` for the project's own conventions and
   methodological invariants (in particular: D_pair is the only
   stochastic-layer gate, the deterministic layer stays network-free — see
   `internal/nonetwork_test.go`).
3. Add or update tests for any behavior change.
4. Update `docs/requirements.md`/`docs/architecture.md`/`docs/user-guide.md`
   (and their `.en.md` counterparts) if your change affects them — stale
   docs are worse than no docs.
5. Open a PR against `main` with a clear description of what changed and
   why. Reference any related issue.

## Code review

This repo uses an automated multi-model review pipeline
(`.claude/skills/fix-review/`) for PRs opened via Claude Code. If you're
contributing from outside that workflow, a manual review by the maintainer
covers the same ground — no special process required on your end.

## Reporting a security issue

Please **do not** open a public issue for a security vulnerability. Open a
private [security advisory](https://github.com/valpere/tumanomir/security/advisories/new)
on GitHub instead.

## License

By contributing, you agree that your contributions will be licensed under
the project's [Apache License 2.0](LICENSE).
