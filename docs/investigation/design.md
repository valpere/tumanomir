# tumanomir — дизайн v0.1

> Український оригінал (джерело істини). Англійський переклад:
> [`design.en.md`](design.en.md).

Вимірювальний інструмент точності специфікацій для AI-проєктів.
Продуктизація методології зі статті «Джерело Невідомості»
(`docs/investigation/SourceOfTheUnknown.md`).

## Метрики

| Метрика | Шар | Що міряє | Прилад |
| --- | --- | --- | --- |
| `K_drift` | детермінований | вимоги без трасування `[REQ-*] -> [FUN/LOG/PHY-*]` | лінтер розмітки, без LLM |
| `D_const` | детермінований | lexical-щільність обмежень (маркери vs проза) | сканер, без LLM |
| `D_pair` | стохастичний | 1 − середня попарна AST-схожість N генерацій | LLM через Ollama |
| `H_norm` | стохастичний | ентропія кластерів / log₂N — ordinal-сигнал | те саме |

Методологічні інваріанти (зі статті, не відкочувати):
- D_pair — робоча метрика; H_norm (= H / log₂N) — ordinal («один кластер чи
  багато»), саме вона репортиться/гейтиться; сира H (біти) обчислюється
  внутрішньо, але сатурує на log₂N при малих N.
- Метрики instrument-relative: конфігурація (модель, промпт, temp, N, поріг) фіксується і звітується.
- invalid rate звітується, не ховається (retry з лічильником discards).
- Пороги — гіпотези за замовчуванням (0.2 / 0.35 / 0.30), калібруються користувачем.
- Для reasoning-моделей — `think: false`; `num_ctx` має вміщати специфікацію.

## CLI UX

```
tumanomir check <file.md|dir>       # детермінований шар, миттєво, git-hook-ready
tumanomir measure <file.md> \
  --instrument ollama:qwen3-coder:30b -n 10 --temp 1.0   # стохастичний шар
```

Вивід — людський у TTY; exit code: 0 ok / 1 gate failed / 2 error.

## Архітектура

```
cmd/tumanomir/          CLI (stdlib flag, subcommands)
internal/types.go       спільні типи (Report, Verdict, Thresholds)
internal/spec/          завантаження markdown-специфікацій
internal/metrics/       K_drift (лінтер), D_const (лексичний сканер)
internal/dispersion/    AST-фічі, cosine, single-linkage, ентропія, D_pair
internal/instrument/    інтерфейс Generator + Ollama-бекенд
```

Походження коду dispersion: порт `sanity/analyze/main.go` з експерименту статті.

## Roadmap (не в v0.1 — YAGNI)

- `.tumanomir.yaml` конфіг + `gate` команда (CI-режим)
- baseline-калібрування (`tumanomir calibrate`)
- bootstrap CI для D_pair
- RFLP-граф (Neo4j) для повного D_const; assisted-режим K_drift (LLM-парсер)
- інші інструменти (OpenAI/Anthropic API), інші проєкції (SQL DDL, OpenAPI)
