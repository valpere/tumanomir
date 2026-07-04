# internal/dispersion/testdata

Representative fixture subset for the stochastic dispersion layer
(D_pair / H measurement) — vendored here in Go's standard `testdata/`
location so a future `internal/dispersion` test package can load it
directly (`testdata/` is auto-excluded from `go build`/`go vet`/`go list`
package scanning by Go convention).

This is a **2-of-6-spec, 2-instrument subset** of the full 120-file
archival corpus at `docs/investigation/_sanity/` — see that directory
(and its own `README.md`) for the complete corpus and full methodology.

## Contents

Both directories hold the "sharp" spec: a ТЗ (spec) written with
`@schema`-style requirement/trace annotations, generated N=10 times per
instrument. Each generation is a `<n>.go`/`<n>.md` pair.

- `out/sharp/` — Instrument A: `qwen3-coder:30b` (local)
- `out-cloud/sharp/` — Instrument B: `kimi-k2.7-code:cloud` (`think: false`)

## Protocol (summary)

1. N=10 generations per spec per instrument.
2. Each generation is reduced to an AST feature vector.
3. Pairwise cosine similarity across all C(10,2) generation pairs.
4. Single-linkage clustering at similarity threshold 0.95 for the
   ordinal H (Shannon entropy over cluster-size distribution) signal.
5. D_pair = 1 − mean(pairwise cosine similarity); the primary
   instrument-relative dispersion metric.

Full methodology: `docs/investigation/_sanity/README.md`.

## Reference numbers (spec: sharp)

| Instrument | mean pairwise sim | D_pair (= 1 − sim) | H@0.95 | H@0.80 |
|---|---|---|---|---|
| A: `qwen3-coder:30b` (local) | 0.730 | 0.270 | 1.77 bits | — |
| B: `kimi-k2.7-code:cloud` (think=false) | 0.682 | 0.318 | 3.32 bits | 2.12 bits |

These numbers come from `docs/investigation/_sanity/README.md` and are
the expected values a future `internal/dispersion` test should
reproduce (within tolerance) when run against the fixtures in this
directory.
