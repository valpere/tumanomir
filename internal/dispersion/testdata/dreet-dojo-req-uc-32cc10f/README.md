# dreet-dojo req-uc @ 32cc10f — per-declaration agreement fixture

Fixture for `dispersion.Decls` / `measure --decls` (REQ-MSR-12, issue #169):
10 generated Go blocks for one real spec, with known structural gaps buried
under naming noise.

## Provenance

- Spec snapshot: `context/tumanomir/specs/2026-11-23-req-uc-32cc10fc19e5.md`
  in the public valpere/dreet-dojo repo (`req-uc` at `32cc10f`).
- Samples: **hand-regenerated** with `instrument.PromptV1` and the dreet-dojo
  instrument config (`context/tumanomir/.tumanomir.yaml`: ollama
  `gemma4:31b-cloud`, temp 1.0, N=10, think false, num_ctx 32768,
  num_predict 4096, sim_threshold 0.95) to inspect where the divergence sits.
  They are **not** the samples of the measured run (reported D_pair ≈ 0.19),
  so do not compare against that number.
- Original names `s1.go..s10.go`; here `01.go..10.go` (REQ-MSR-11 naming).

## The unparseable sample

`05.go.invalid` is s5. It does not parse (`155:63: expected '}', found ')'`),
so `Analyze` and `Decls` drop it and the fixture's **N = 9**. The `.invalid`
suffix keeps `loadGoFixtures` (`*.go` glob) from picking it up.

## Reference numbers (pinned from `dispersion.Analyze`, sim threshold 0.95)

| N | D_pair | 95% CI | H (bits) | clusters |
|---|--------|--------|----------|----------|
| 9 | 0.2462 | [0.1554, 0.2616] | 3.1699 (H_norm 1.00) | 9 (all singletons) |

Asserted in `TestDeclsFixtureRender` (`internal/dispersion/decls_test.go`).

## What `Decls` must show (asserted in the same file)

- `Hint`, `Diff`, `RecordAttempt` are each one member slot, 9/9 present, with
  2 / 3 / 2 shapes (e.g. `Hint(ctx)` x7 vs `Hint(ctx, cardID)` x2).
- Naming noise folds into slots: `Store`/`SRSStore`, `Service`/
  `SessionService`, `CheckResult`/`Result`, `CheckStatus`/`Status`,
  `Format`/`CardFormat`, `GiveUp`/`Giveup`; local type names render as the
  slot label in shapes.
- s6/s10 declare `struct{}` types with receiver methods; those land in the
  same slots as the other samples' interface methods.
- Note for Task 3 (D_pair_shape go/no-go): `Next` (no `ahead` flag) is
  **unanimous** — a single shape, 9/9 — so the `ahead` gap suspected from the
  field case is not visible in this regenerated set.
