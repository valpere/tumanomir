# tumanomir — external documentation audit: docs/architecture.md (cursor)

> Model: cursor-agent --model auto. Full-codebase context (agent explored the repo
> directly via its own tools, not embedded doc text). Part of a
> 5-agent x 3-document audit round, 2026-07-04. Read-only: agent was
> explicitly instructed not to edit/create/delete any repo file;
> confirmed clean afterward via git status.

---

## What's good

**Структура і роль документа.** Чітко відокремлено від `roadmap.md` і `requirements.md`: що / як / що далі. Заголовок про колишнє `design.md` і перенесення на рівень `docs/` дає контекст без зайвого шуму.

**Таблиця метрик і пакетна карта.** Чотири метрики, шар (детермінований vs стохастичний), призначення — стисло і в цілому відповідає коду. Дерево пакетів збігається з репозиторієм: `cmd/tumanomir`, `internal/types.go`, `internal/spec`, `internal/metrics`, `internal/dispersion`, `internal/instrument`. Згадка `Generator`, `PromptV1`, fence-екстрактора, single-linkage і `internal/nonetwork_test.go` (REQ-CHK-05) — корисна для орієнтації.

**Методологічні інваріанти.** Блок про instrument-relative звіти (REQ-MSR-04), invalid rate з retry ≤2 і порогом 40%, гіпотезні пороги 0.20/0.35/0.30, advisory-роль `D_const`, `think: false`, перевірку `num_ctx` до HTTP — в основному збігається з `main.go`, `instrument/ollama.go` і `docs/requirements.md`.

**CLI UX (частково).** Синтаксис `check` / `measure`, обов’язкові прапорці `--instrument`, `--num-ctx`, `--num-predict`, exit codes 0/1/2 — відповідають `cmd/tumanomir/main.go`. Примітка про інлайн-рендеринг і `TODO(REQ-OUT-01)` чесно відображає поточний стан.

**Для нової сесії агента.** Разом з посиланням на `requirements.md` і `roadmap.md` документ дає швидку «карту місцевості»: які метрики, які пакети, де мережа дозволена, що ще не винесено в `internal/report/`.

---

## What's bad

**Внутрішня суперечність щодо gating `H_norm`.** Рядки 32–33 кажуть, що `H_norm` «репортиться/гейтиться», тоді як рядки 41–42 правильно стверджують: exit code гейтять лише `K_drift` і `D_pair`. Код підтверджує друге: `runMeasure` повертає 1 лише при `DPairVerdict == VerdictBlock`; `H`/`H_norm` явно «not gated». Формулювання «гейтиться» для `H_norm` вводить в оману і суперечить REQ-MSR-02.

**Сира `H` названа «внутрішньою», але друкується в CLI.** Інваріант каже, що H (біти) обчислюється внутрішньо; `printMeasureResult` виводить окремий рядок `H: … bits` поряд з `H_norm`. Це не критичний баг коду, але architecture.md неточно описує фактичний вивід `measure`.

**Неповний список CLI-прапорців.** У блоці CLI відсутній `--d-pair-max` (default 0.30), хоча він є в `usage`, у `runMeasure` і є частиною gating стохастичного шару. Без нього читач не зрозуміє, як налаштовується gate для `D_pair` (REQ-CFG-01).

**Відсутня підкоманда `version`.** У секції пакетів згадано `check/measure/version`, у CLI UX — ні. Код має `tumanomir version` → `0.1.0-dev`.

**Неповне REQ-MSR-06.** Документ згадує preflight `num_ctx` (оцінка розміру промпту), але не згадує виявлення **output truncation** через `done_reason=length` і попередження в звіті — це реалізовано в `runMeasureWithGenerator` / `printMeasureResult` і явно вимагається в requirements.

**Стислий, але неточний provenance dispersion.** «Порт `sanity/analyze/main.go`» — фактичний шлях у репо: `docs/investigation/_sanity/analyze/main.go`. Для зовнішнього рев’юера це легко знайти не там.

**Поведінка `skipped` не описана.** Код має `VerdictSkipped` для: (a) нуль `[REQ-*]` у `check`; (b) <2 валідних семплів у `measure`. Architecture.md про це мовчить — ризик, що `0.00 [ok]` і `— [skipped]` змішаються в розумінні.

**Мовна пара.** `architecture.en.md` існує як переклад; будь-яка суперечність (наприклад, про gating) дублюється в обох файлах. Для зовнішнього аудиту це не блокер, але підтримка синхронності — додатковий ризик дрейфу.

---

## What it doesn't cover

Відносно `docs/requirements.md` architecture.md **не дає читачеві повної картини відповідальності компонентів** за такі вимоги:

| Requirement | Gap |
|---|---|
| **REQ-CHK-01** | Немає правила «zero `[REQ-*]` → skipped, не `0.00 [ok]`» і зв’язку з `aggregate` у `main.go`. |
| **REQ-CHK-02** | Не сказано, що `check` друкує список `hanging:` ID (actionable output). |
| **REQ-CHK-03** | Немає операційного визначення `D_const` (набір маркерів, формула markers/(markers+prose)). |
| **REQ-CHK-04** | `spec.Load` згаданий лише побіжно; немає «рекурсивно, лише `*.md`», агрегації по корпусу. |
| **REQ-MSR-01** | Немає посилання на `astfeat.go` як авторитетну реалізацію bag-of-features / cosine / попарного середнього. |
| **REQ-MSR-05** | Retry-цикл (3 спроби на слот), відмінність discard vs truncation, prominent warning vs numeric summary — лише частково в інваріантах. |
| **REQ-MSR-06** | Output truncation (`DoneReason`) — відсутній (див. вище). |
| **REQ-OUT-01** | Формат «один рядок на метрику з verdict і порогом» для обох команд; структура `Report` з `@schema` не відображена. |
| **REQ-OUT-02** | Exit codes є, але не сказано: `check` блокує лише на `K_drift`; `measure` — лише на `D_pair`; skipped не дає exit 1. |
| **REQ-CFG-01** | `--d-pair-max` відсутній у CLI-блоці. |
| **REQ-NFR-01** | Performance (<100 ms на 1 MB corpus, benchmark у `internal/metrics`) — повністю поза architecture; benchmark у коді наразі відсутній. |
| **REQ-NFR-02** | Go ≥1.26, stdlib-only, один статичний бінарник — не згадано (хоча `go.mod` це підтверджує). |
| **REQ-NFR-03** | Лише частково через інваріанти; немає посилання на `CLAUDE.md` як дзеркало governance. |

Також не описано **поведінкові обмеження v0.1**, які є в коді: `measure` приймає лише один файл (директорії відхиляються); `check` друкує placeholder `D_pair: —`; pipeline генерації (prompt → Ollama → `ExtractGoBlock` → `ValidGo` → `dispersion.Analyze`) — читач architecture.md alone не зрозуміє end-to-end потік стохастичного шару.

Секція 4 requirements (out of scope) правильно делегована в roadmap — це не прогалина.

---

## Verdict

`docs/architecture.md` — **корисний, але недостатній** як єдиний опис поточної системи: пакетна карта, метрики й більшість інваріантів узгоджені з кодом, і документ добре орієнтує на високому рівні. Перед тим як покладатися на нього як на повний «how» без `requirements.md`, варто виправити суперечність про gating `H_norm`, узгодити опис сирої `H` з фактичним CLI-виводом, доповнити CLI (`--d-pair-max`, `version`), розширити REQ-MSR-06 і `skipped`-стани, і явно прив’язати ключові [REQ-*] до компонентів і потоків даних. У поточному вигляді це **точний скелет, а не повний і безсуперечний опис as-is** — для onboarding досить разом з `requirements.md`; як standalone architecture doc потребує доопрацювання.
