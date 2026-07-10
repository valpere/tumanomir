# tumanomir — посібник користувача

> Український оригінал (джерело істини). Англійський переклад:
> [`user-guide.en.md`](user-guide.en.md).

Практичний, орієнтований на приклади посібник: як реально користуватись
`tumanomir` день у день. Для елеваторного пітчу й філософії методології —
[`README.md`](../README.md) і оригінальна стаття
[`docs/investigation/SourceOfTheUnknown.md`](investigation/SourceOfTheUnknown.md);
тут ця аргументація не повторюється.

## 1. Що таке tumanomir

`tumanomir` — інструмент вимірювання точності специфікацій для проєктів,
де реалізацію пише AI-агент. Він рахує дві незалежні метрики: детерміновану
(`K_drift`, `D_const` — без мережі, без LLM) і стохастичну (`D_pair`,
`H_norm` — генерує N Go-артефактів з вашого спека через Ollama і міряє,
наскільки далеко вони розійшлись). Чотири команди: `check` (детермінований
шар), `measure` (стохастичний шар), `gate` (обидва за один прохід, для CI)
і `calibrate` (кореляція метрик з історичним корпусом розмічених вимірів).
Усі чотири вже реалізовані.

## 2. Встановлення та збірка

Передумова: Go >= 1.26.

```bash
git clone https://github.com/valpere/tumanomir.git
cd tumanomir
make build     # -> bin/tumanomir
```

Make-цілі, що знадобляться користувачу (повний список — `Makefile`):

```bash
make build     # go build -o bin/tumanomir ./cmd/tumanomir
make test      # go test ./...
make dogfood   # build + bin/tumanomir check docs/requirements.md
               # (сам інструмент гейтить власну специфікацію — дим-тест)
make ci        # build + vet + test + lint + dogfood, все разом
```

Далі всі приклади припускають, що ви в корені репозиторію tumanomir і
`bin/tumanomir` вже зібраний; замініть шлях до вашого власного спека, де
доречно.

## 3. Швидкий старт

### 3.1. `check` — нуль налаштувань, нуль мережі

```bash
bin/tumanomir check docs/requirements.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.20, 0/30 requirements untraced)
  D_const:  0.03  [warn]   (threshold 0.35, 95 markers / 3161 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)
```

Це реальний вивід дим-тесту `make dogfood`: `tumanomir` вимірює власну
специфікацію. Запустіть те саме на своєму спеку (файл або директорія,
рекурсивно `*.md`):

```bash
bin/tumanomir check path/to/your-specs/
```

Жодного налаштування чи мережевого виклику не потрібно — `check`
безпечний для git pre-commit hook (див. §7.1).

### 3.2. `measure` — стохастичний шар, потребує запущеного Ollama

```bash
bin/tumanomir measure \
  --instrument ollama:qwen3-coder:30b \
  -n 3 --temp 1.0 --sim-threshold 0.95 \
  --num-ctx 8192 --num-predict 2048 \
  docs/investigation/_sanity/specs/sharp.md
```

Реальний вивід (жива генерація проти `ollama:qwen3-coder:30b`, `-n 3` —
для швидкості; типовий прогін бере `-n 10`, дефолт `--samples`):

```
Instrument config (REQ-MSR-04):
  backend:        ollama
  model:          qwen3-coder:30b
  temperature:    1.00
  samples (N):    3
  think:          false
  num_ctx:        8192
  num_predict:    2048
  sim_threshold:  0.95
  prompt:         PromptV1 (276 bytes)

  D_pair:   0.33  [block]  (95% CI [-0.00, 0.33]; threshold 0.30, mean sim 0.67, N=3 valid, 0 discarded)
  H:        1.58  bits (ordinal signal only, not gated)
  H_norm:   1.00  (ordinal signal only, not gated)

exit code: 1 (gate failed)
```

`--instrument`, `--num-ctx` і `--num-predict` обов'язкові — детальніше
у §4.2.

## 4. Довідник команд

Повний список прапорців з дефолтами — таблиця в
[`docs/architecture.md`](architecture.md#cli-ux) («CLI UX»); тут вона не
дублюється, лише пояснюється на прикладах.

### 4.1. `check`

```bash
bin/tumanomir check docs/investigation/_sanity/specs/sharp.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.20, 0/3 requirements untraced)
  D_const:  0.11  [warn]   (threshold 0.35, 10 markers / 84 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)
```

`D_const` тут `[warn]`, а не `[block]` — навмисно: D_const лексичний
проксі (маркери/проза), він ніколи не блокує exit code, лише K_drift може
(REQ-CHK-06).

Директорія замість файлу агрегує всі `*.md` рекурсивно (крім
`.`-/`_`-префіксних піддиректорій — `.git`, `_sanity` тощо):

```bash
bin/tumanomir check docs/
```

Власний поріг K_drift і `--format json`:

```bash
bin/tumanomir check --k-drift-max 0.5 --format json docs/requirements.md | jq .
```

```json
{
  "result": {
    "k_drift": {"requirements": 30, "hanging": 0, "hanging_ids": null, "value": 0},
    "d_const": {"constraint_markers": 95, "prose_tokens": 3161, "value": 0.029176904176904175},
    "k_drift_verdict": "ok",
    "d_const_verdict": "warn"
  },
  "thresholds": {"k_drift_max": 0.5, "d_const_min": 0.35, "d_pair_max": 0.3}
}
```

### 4.2. `measure`

`--instrument` (`backend:model`, наприклад `ollama:qwen3-coder:30b`),
`--num-ctx` і `--num-predict` — обов'язкові. `--num-ctx` мусить мати запас
і під промпт, і під `--num-predict` — інакше запуск відхиляється ще до
HTTP-виклику (silent truncation — це баг цілісності виміру, не warning,
REQ-MSR-06):

```bash
bin/tumanomir measure --instrument ollama:qwen3-coder:30b \
  --num-ctx 100 --num-predict 2048 \
  docs/investigation/_sanity/specs/sharp.md
```

```
measure: generation failed: instrument: estimated prompt tokens (427, len(prompt)/3 heuristic) + num_predict (2048) exceeds num_ctx (100); increase num_ctx or reduce num_predict
```

`-n`/`--samples` мусить бути `>= 2` (потрібна пара для попарної схожості).
`--sim-threshold` — поріг single-linkage кластеризації для H/H_norm,
дефолт 0.95.

**Лічильник discard і попередження >40% (REQ-MSR-05).** Кожен семпл
намагається згенеруватись до 3 разів (1 спроба + 2 ретраї); якщо жодна
спроба не дала валідного Go, семпл відкидається — лічильник ніколи не
ховається. Коли частка відкинутих семплів перевищує 40% (гіпотеза, не
калібрована константа — той самий статус, що й у 0.20/0.35/0.30), звіт
друкує окремий попереджувальний рядок над метриками:

```
⚠ discard rate: 50% (2/4 generations invalid) — exceeds the 40% hypothesis threshold (REQ-MSR-05); results may be unreliable
```

(Точні цифри `%d/%d` залежать від вашого прогону — вище наведено формат
рядка з `internal/report/report.go`, не вигаданий приклад числа; discard
rate 0% у прогоні §3.2 показаний прямо в рядку `D_pair` як `0 discarded`.)
Окремо, тим самим механізмом, звіт попереджає про `done_reason=length`
(генерація обрізана `num_predict`) і про генерації, чий фактичний
prompt-token count суттєво перевищив preflight-оцінку — обидва теж
REQ-MSR-06/issue #57, не приховуються.

**Instrument-relative вимірювання.** Кожен звіт `measure` починається з
повної конфігурації приладу (backend, model, temperature, N, think,
num_ctx, num_predict, sim_threshold, версія промпту) — не декоративно:
`D_pair`, виміряний під однією конфігурацією, **не порівнюваний**
з `D_pair` під іншою (інша модель, інша температура, інше N) без
перекалібрування. Число «D_pair = 0.33» саме по собі нічого не значить —
значення набуває лише разом із блоком `Instrument config` над ним.

**Пороги — некалібровані гіпотези.** `--d-pair-max` (дефолт 0.30), як і
`--k-drift-max`/`--d-const-min`, — стартові значення зі статті-джерела
методології, не перевірені емпірично константи. Перш ніж покладатись на
дефолт як на «пройшло/не пройшло» рішення для вашого проєкту, накопичте
власний розмічений корпус і прогоніть `calibrate` (§4.4) — саме для цього
він і існує.

### 4.3. `gate`

`gate` = `check` + `measure` (якщо прилад визначено) за один прохід, один
exit code — призначено для CI. Без `--instrument` і без секції
`instrument:` у `.tumanomir.yaml` працює лише детерміновано:

```bash
bin/tumanomir gate docs/investigation/_sanity/specs/sharp.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.20, 0/3 requirements untraced)
  D_const:  0.11  [warn]   (threshold 0.35, 10 markers / 84 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)

exit code: 0 (gates pass)
```

З приладом — обидва шари, `--format json`:

```bash
bin/tumanomir gate --instrument ollama:qwen3-coder:30b -n 3 \
  --num-ctx 8192 --num-predict 2048 --format json \
  docs/investigation/_sanity/specs/sharp.md | jq '.result.measure.dispersion'
```

```json
{
  "n": 3,
  "discarded": 0,
  "mean_sim": 0.812044357642638,
  "d_pair": 0.18795564235736195,
  "d_pair_ci_low": 0,
  "d_pair_ci_high": 0.1885752229329093,
  "clusters": 2,
  "sim_thresh": 0.95,
  "h": 0.9182958340544896,
  "h_norm": 0.579380164285695
}
```

**REQ-GATE-02: жодного тихого пониження.** Якщо ви передали
measure-специфічний прапорець (`--temp`, `-n`/`--samples`,
`--sim-threshold`, `--num-ctx`, `--num-predict`, `--think`,
`--d-pair-max`) явно, а прилад так і не визначився (ні CLI, ні
`.tumanomir.yaml`), `gate` не тихо ігнорує прапорець і не падає назад у
детермінований режим — він відмовляється з exit code 2:

```bash
bin/tumanomir gate --temp 0.5 docs/investigation/_sanity/specs/sharp.md
```

```
gate: --temp was passed but no instrument resolved (no --instrument and no .tumanomir.yaml instrument: section) — refusing to silently downgrade to deterministic-only (REQ-GATE-02)
```

Це той самий клас бага цілісності виміру, що й silent truncation у
REQ-MSR-06 — не зручність, яку можна вимкнути.

### 4.4. `calibrate`

`calibrate` не звертається до мережі й не запускає LLM — `d_pair` завжди
читається з корпусу, ніколи не переміряється. Корпус — JSONL, один рядок
на історичну специфікацію:

```jsonl
{"spec_path": "docs/investigation/_sanity/specs/sharp.md", "instrument": "ollama:qwen3-coder:30b", "d_pair": 0.19, "outcome": 0.2}
{"spec_path": "docs/investigation/_sanity/specs/fog.md", "instrument": "ollama:qwen3-coder:30b", "d_pair": 0.55, "outcome": 0.9}
{"spec_path": "docs/investigation/_sanity/specs/baseline.md", "instrument": "ollama:qwen3-coder:30b", "d_pair": 0.30, "outcome": 0.5}
```

```bash
bin/tumanomir calibrate corpus.jsonl
```

```
Calibration over 3 valid row(s), 0 skipped

⚠ fewer than 5 valid rows — correlation coefficients below are not statistically meaningful yet

K_drift   spearman=+0.00
  outcome <= median:  min=0.00 mean=0.00 max=0.00
  outcome >  median:  min=0.00 mean=0.00 max=0.00

D_const   spearman=-0.87
  outcome <= median:  min=0.00 mean=0.05 max=0.11
  outcome >  median:  min=0.00 mean=0.00 max=0.00

D_pair    spearman=+1.00
  outcome <= median:  min=0.19 mean=0.24 max=0.30
  outcome >  median:  min=0.55 mean=0.55 max=0.55

No threshold is auto-selected or written to .tumanomir.yaml — use these numbers to inform your own choice (REQ-NFR-03).
```

(Приклад вище — навмисно надто малий, 3 рядки замість рекомендованих ≥5, —
щоб показати попередження про малу вибірку; реальні `d_pair`/`outcome` у
вашому корпусі мають походити з реальних `measure`-прогонів і фактичного
подальшого результату, не бути придуманими.)

`spec_path` мусить вказувати на незмінний знімок специфікації (не на
живий робочий файл, що продовжує змінюватись) — саме той знімок, що дав
пару `d_pair`/`outcome`. Усі рядки одного прогону мають розділяти одне
значення `instrument`; друге, відмінне значення будь-де в корпусі —
жорсткий abort з exit code 2 (REQ-CAL-02), а не пропуск рядка:

```bash
bin/tumanomir calibrate corpus-mixed.jsonl
```

```
calibrate: corpus mixes instruments "ollama:qwen3-coder:30b" and "ollama:glm-5.1:cloud" — all rows in one run must share the same instrument (REQ-MSR-04)
```

Некоректні рядки (не парсяться, `spec_path` не читається,
`d_pair`/`outcome` поза `[0,1]`) — пропускаються й рахуються, ніколи не
відкидаються тихо; нуль валідних рядків — exit code 2.

`calibrate` **інформує, не задає поріг сам** — він ніколи не пише в
`.tumanomir.yaml` і не пропонує єдине число «встанови `--d-pair-max` на
X» (REQ-CAL-03/04). Той самий принцип §4.2 стосується й тут: дефолтні
0.20/0.35/0.30 — гіпотези зі статті; `calibrate` — інструмент, яким ви їх
перевіряєте на власних даних, а не заміна для людського рішення.

`calibrate` не приймає `--format` — лише текстовий вивід, JSON-режиму
немає (REQ-OUT-03 стосується тільки `check`/`measure`/`gate`).

## 5. Файл конфігурації `.tumanomir.yaml`

`check`/`measure`/`gate` шукають `./.tumanomir.yaml` (лише поточна
директорія, без пошуку вгору) і завантажують його, якщо є; явний
`--config <path>` — авторитетний, названий файл мусить існувати й
парситись, інакше exit code 2. Повна схема (дзеркалить
`internal/config/config.go`):

```yaml
thresholds:
  k_drift_max: 0.20    # float, [0,1]
  d_const_min: 0.35    # float, [0,1]
  d_pair_max: 0.30     # float, [0,1]
instrument:
  backend: ollama       # v0.1: тільки "ollama"
  model: qwen3-coder:30b
  temperature: 1.0
  samples: 10           # int, >= 2
  think: false
  num_ctx: 8192          # int, мусить мати запас під prompt + num_predict
  num_predict: 2048       # int
  sim_threshold: 0.95     # float, [0,1]
```

`prompt`/`prompt_version` навмисно відсутні в схемі — вони не
конфігуровані (REQ-MSR-04: промпт мусить бути відтворюваним зі звіту, не
довільним per-project значенням).

**Пріоритет: CLI-прапорець > файл конфігурації > вбудований дефолт**
(REQ-CFG-03). Приклад: з файлом вище (`k_drift_max: 0.10`) у поточній
директорії:

```bash
bin/tumanomir check docs/investigation/_sanity/specs/sharp.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.10, 0/3 requirements untraced)
  D_const:  0.11  [warn]   (threshold 0.40, 10 markers / 84 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)
```

Поріг узятий з файлу (`0.10`, `0.40`). Явний CLI-прапорець перекриває
його:

```bash
bin/tumanomir check --k-drift-max 0.5 docs/investigation/_sanity/specs/sharp.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.50, 0/3 requirements untraced)
  D_const:  0.11  [warn]   (threshold 0.40, 10 markers / 84 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)
```

`--k-drift-max 0.5` переміг файлове `0.10`; `d_const_min` (`0.40`) з
файлу лишився незмінним, бо CLI його не чіпав.

## 6. `--format json`

`check`, `measure` і `gate` приймають `--format json`: рівно один compact
JSON-об'єкт у stdout, нічого більше. Форма й імена полів визначаються
`json`-тегами Go-структур (`CheckResult`, `MeasureResult`, `Report` та
вкладені типи), не переописуються тут — авторитетне джерело:
[`docs/requirements.md`](requirements.md) REQ-OUT-03 і блоки `@schema
Report`/`@schema Thresholds`/`@schema InstrumentConfig` у §1. Будь-яке
значення `--format`, відмінне від `text`/`json`, — usage error (exit 2).

Приклади (`| jq` для витягування конкретного поля):

```bash
bin/tumanomir check --format json docs/requirements.md | jq '.result.k_drift.value'
# 0

bin/tumanomir measure --instrument ollama:qwen3-coder:30b -n 3 \
  --num-ctx 8192 --num-predict 2048 --format json \
  docs/investigation/_sanity/specs/sharp.md | jq '.result.dispersion.d_pair'
# 0.18795564235736195

bin/tumanomir gate --format json docs/investigation/_sanity/specs/sharp.md | jq '.result.exit_code'
# 0
```

`check`/`measure`'s JSON не несе поля `exit_code` — реальний
exit-код процесу лишається єдиним сигналом для цих двох команд;
`gate`'s JSON несе `result.exit_code` (REQ-GATE-03).

## 7. Патерни використання

### 7.1. `check` як git pre-commit hook

`check` — нуль мережі, нуль LLM — безпечний для pre-commit:

```bash
#!/bin/sh
# .git/hooks/pre-commit
bin/tumanomir check docs/ || {
  echo "tumanomir check failed — see above" >&2
  exit 1
}
```

(`chmod +x .git/hooks/pre-commit` після створення.) Не забувайте: exit
code 1 тут означає лише K_drift `[block]` — `D_const` ніколи не блокує
(REQ-CHK-06), тож hook не відмовить через самі лише лексичні
попередження.

### 7.2. `gate` як крок CI

```yaml
# .github/workflows/spec-gate.yml (фрагмент)
- name: tumanomir gate
  run: |
    bin/tumanomir gate --instrument ollama:qwen3-coder:30b \
      --num-ctx 8192 --num-predict 2048 \
      docs/spec.md
```

Один exit code (0/1/2), CI-composable за конструкцією (REQ-OUT-02).
Якщо в CI немає доступу до Ollama — не передавайте `--instrument` і
`gate` спрацює лише детерміновано (§4.3); просто не передавайте й жодного
іншого measure-специфічного прапорця, інакше REQ-GATE-02 відмовить рано.

### 7.3. Накопичення корпусу для `calibrate` — покроково, з часом

`calibrate` вимагає розміченого корпусу (`d_pair` + фактичний
downstream-результат), якого на старті проєкту ще немає. Загальний
рецепт, не прив'язаний до конкретного проєкту:

1. **Сьогодні**, коли специфікація готова й ви прогнали `measure`:
   зафіксуйте `d_pair` (`measure --format json | jq '.result.dispersion.d_pair'`)
   і збережіть незмінний знімок самої специфікації (скопіюйте файл у
   версійований шлях, наприклад `specs-archive/2026-07-10-feature-x.md`,
   або зафіксуйте git-ревізію — головне, щоб `spec_path` пізніше вказував
   на те саме, що було виміряно).
2. Додайте рядок у ваш `corpus.jsonl` з відомими на цей момент полями
   (`spec_path`, `instrument`, `d_pair`) — `outcome` поки невідомий.
3. **Пізніше**, коли фактичний результат став відомий (скільки ітерацій
   знадобилось агенту, чи довелось переписувати, скільки часу пішло на
   review) — визначте `outcome` за власною шкалою (вищий = гірше) і
   допишіть (або відредагуйте) той рядок.
4. Періодично прогоняйте `calibrate corpus.jsonl` — коли рядків
   назбирається ≥5, попередження про малу вибірку зникне, а коефіцієнти
   Spearman стануть змістовнішим сигналом про те, чи `D_pair` дійсно
   передбачає ваш `outcome` на вашому приладі.

## 8. Усунення несправностей

Усі рядки нижче — дослівні повідомлення з поточного коду (звірені під
час написання цього посібника), не переказ своїми словами.

| Повідомлення | Причина | Виправлення |
| --- | --- | --- |
| `measure: --instrument is required, format backend:model (e.g. ollama:qwen3-coder:30b)` | `measure` без `--instrument` | додайте `--instrument backend:model` |
| `measure: unsupported backend "openai"; v0.1 supports only "ollama"` | backend, відмінний від `ollama` | v0.1 підтримує лише `ollama` (roadmap: інші прилади) |
| `measure: --num-ctx is required (must exceed the prompt token count)` | не передано `--num-ctx` (або `<= 0`) | додайте `--num-ctx <N>` з запасом над розміром промпту |
| `measure: generation failed: instrument: estimated prompt tokens (427, ...) + num_predict (2048) exceeds num_ctx (100); increase num_ctx or reduce num_predict` | preflight-перевірка (REQ-MSR-06): промпт + `num_predict` не влазять у `num_ctx` | збільшіть `--num-ctx` або зменшіть `--num-predict` |
| `check: --format must be "text" or "json", got "xml"` | `--format` з невідомим значенням | лише `text` або `json` |
| `gate: --temp was passed but no instrument resolved (...) — refusing to silently downgrade to deterministic-only (REQ-GATE-02)` | measure-специфічний прапорець на `gate` без визначеного приладу | додайте `--instrument` (або приберіть цей прапорець, якщо хотіли лише детермінований прогін) |
| `calibrate: corpus mixes instruments "ollama:qwen3-coder:30b" and "ollama:glm-5.1:cloud" — all rows in one run must share the same instrument (REQ-MSR-04)` | другий, відмінний `instrument` у корпусі | розділіть корпус по приладу — один прогін `calibrate` на один прилад |
| `check: exactly one <file.md\|dir> argument required` | `check` викликано без аргументу або з кількома | передайте рівно один файл/директорію |
| `measure: exactly one <file.md> argument required` | `measure` викликано без аргументу або з кількома | передайте рівно один файл спека |
| `gate: exactly one <file.md> argument required` | `gate` викликано без аргументу або з кількома (і ніколи з директорією) | передайте рівно один файл спека |
| `calibrate: exactly one <corpus.jsonl> argument required` | `calibrate` викликано без аргументу або з кількома | передайте рівно один шлях до JSONL-корпусу |

## 9. Куди далі

- [`docs/architecture.md`](architecture.md) — як інструмент збудований:
  пакети, повна таблиця прапорців CLI, методологічні інваріанти.
- [`docs/requirements.md`](requirements.md) — авторитетна специфікація
  поведінки (`[REQ-*]`-трасування, `@schema`-блоки) — джерело істини для
  JSON-схеми й точної поведінки кожного прапорця.
- [`docs/roadmap.md`](roadmap.md) — що ще не збудовано і в якому
  порядку (тактичний борг — у [GitHub issues](https://github.com/valpere/tumanomir/issues)).
