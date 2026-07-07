# tumanomir — архітектура

> Український оригінал (джерело істини). Англійський переклад:
> [`architecture.en.md`](architecture.en.md).
>
> Раніше цей файл жив як `docs/investigation/design.md` — під
> "дослідницькою" папкою, призначеною для провенансу методології
> (`docs/investigation/history.md`, зовнішні рев'ю), а не для живої
> архітектури інструмента. Перенесено на рівень `docs/requirements.md`:
> вимоги — що, архітектура — як, `investigation/` — чому і як
> перевірялось.

Вимірювальний інструмент точності специфікацій для AI-проєктів.
Продуктизація методології зі статті «Джерело Невідомості»
(`docs/investigation/SourceOfTheUnknown.md`).

Roadmap (що ще не збудовано і в якому порядку) — окремо, у
[`roadmap.md`](roadmap.md). Тактичний борг і дрібні задачі — у
[GitHub issues](https://github.com/valpere/tumanomir/issues), не тут.

## Метрики

| Метрика | Шар | Що міряє | Прилад |
| --- | --- | --- | --- |
| `K_drift` | детермінований | вимоги без трасування `[REQ-*] -> [FUN/LOG/PHY-*]` | лінтер розмітки, без LLM |
| `D_const` | детермінований | lexical-щільність обмежень (маркери vs проза) | сканер, без LLM |
| `D_pair` | стохастичний | 1 − середня попарна AST-схожість N генерацій | LLM через Ollama |
| `H_norm` | стохастичний | ентропія кластерів / log₂N — ordinal-сигнал | те саме |

Методологічні інваріанти (зі статті, не відкочувати без оновлення
`docs/requirements.md`):
- D_pair — робоча метрика й єдиний гейт стохастичного шару; H_norm
  (= H / log₂N) — ordinal («один кластер чи багато»), репортиться, але
  ніколи не гейтить; сира H (біти) теж друкується у звіті, але сатурує на
  log₂N при малих N.
- Метрики instrument-relative: повна конфігурація (backend, модель, temp, N,
  think, num_ctx, num_predict, sim_threshold, промпт) фіксується і
  друкується в кожному звіті `measure` (REQ-MSR-04).
- invalid rate звітується, не ховається (retry ≤2 на семпл, лічильник
  discards, попередження при discard rate > 40%).
- Пороги — гіпотези за замовчуванням (0.20 / 0.35 / 0.30), калібруються
  користувачем; лише K_drift і D_pair гейтять exit code, D_const і H_norm —
  ordinal/advisory (REQ-CHK-06 для D_const, REQ-MSR-02 для H_norm).
- Для reasoning-моделей — `think: false`; `num_ctx` перевіряється проти
  оцінки розміру промпту до HTTP-виклику (silent truncation = баг
  цілісності виміру, не попередження).

## CLI UX

```
tumanomir check [flags] <file.md|dir>   # детермінований шар: K_drift, D_const
tumanomir measure [flags] <file.md>     # стохастичний шар: D_pair, H_norm
tumanomir version                       # надрукувати версію і вийти

# check і measure
--config  string  шлях до .tumanomir.yaml (за замовчуванням: завантажити
                   ./.tumanomir.yaml, якщо є, лише поточна директорія, без
                   пошуку вгору; явний --config має існувати і парситись)

# check
--k-drift-max  float   gate: max fraction of untraced requirements (default 0.20)
--d-const-min  float   warn: min lexical constraint density (default 0.35)

# measure
--instrument     string  required, format backend:model (e.g. ollama:qwen3-coder:30b)
-n, --samples    int     number of generations to sample, must be >=2 (default 10)
--temp           float   sampling temperature (default 1.0)
--sim-threshold  float   single-linkage clustering threshold, in [0,1] (default 0.95)
--num-ctx        int     required: context window; must exceed the prompt token count
--num-predict    int     required: max generated tokens; must exceed natural output length
--think          bool    enable reasoning-model think mode (default false)
--d-pair-max     float   gate: max 1 − mean pairwise AST similarity (default 0.30)
```

Вивід — людський у TTY; exit code: 0 ok / 1 gate failed / 2 error.

## Архітектура пакетів

```
cmd/tumanomir/          CLI (stdlib flag, підкоманди check/measure/version)
internal/types.go       спільні типи (Verdict, Thresholds, InstrumentConfig,
                         KDriftResult, DConstResult, DispersionResult)
internal/config/        завантаження .tumanomir.yaml (REQ-CFG-02/03)
internal/spec/          завантаження markdown-специфікацій (файл або директорія)
internal/metrics/       K_drift (лінтер трасування), D_const (лексичний сканер)
internal/dispersion/    AST-фічі, cosine, single-linkage, ентропія, D_pair
internal/instrument/    інтерфейс Generator, Ollama-бекенд, PromptV1 + фрейм-екстрактор
internal/report/        рендеринг CheckResult/MeasureResult у TTY-звіт (REQ-OUT-01)
```

`internal/instrument` — єдиний пакет, якому дозволено мережу
(`internal/nonetwork_test.go` рантайм-перевіряє, що `internal/metrics`,
`internal/spec` і `internal/config` цього не порушують — REQ-CHK-05).

Рендеринг звітів винесено в `internal/report` (`RenderCheck`/`RenderMeasure`,
issue #82): пакет залежить лише від `internal`, ніколи від
`internal/metrics`/`internal/spec` — `aggregate()` (агрегація по файлах)
лишається в `cmd/tumanomir`, у `internal/report` переїхав тільки тип
`CheckResult`, який вона повертає.

Походження коду dispersion: порт `docs/investigation/_sanity/analyze/main.go`
з експерименту статті.
