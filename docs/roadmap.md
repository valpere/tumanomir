# tumanomir — roadmap

> Український оригінал (джерело істини). Англійський переклад:
> [`roadmap.en.md`](roadmap.en.md).
>
> Раніше це був невпорядкований список у кінці `design.md`. Тут —
> впорядковано за горизонтом, з обґрунтуванням чому саме такий порядок.
> **Тактичний борг** (баги, дрібні покращення, тестові прогалини) — не
> тут, а в [GitHub issues](https://github.com/valpere/tumanomir/issues);
> цей файл — тільки про функціональність, якої ще немає.

## v0.1 — зроблено

`check` (K_drift, D_const), `measure` (D_pair, H_norm, з 95% bootstrap CI
для D_pair — REQ-MSR-07) і `gate` (обидва шари за один прохід, один exit
code для CI, REQ-GATE-01..03) працюють end-to-end проти реального Ollama.
Деталі — [`architecture.md`](architecture.md).

## Mid-term — обговорено, не заплановано

1. **Дані для `tumanomir calibrate`.** Сам інструмент вже збудовано
   (`calibrate <corpus.jsonl>`, issue #94, REQ-CAL-01..05): читає JSONL
   корпус історичних специфікацій, кожна з парою (заздалегідь виміряний
   D_pair, визначений користувачем outcome), рахує Spearman-кореляцію
   K_drift/D_const/D_pair проти outcome і друкує median-split — інформує,
   не задає поріг сама. Що лишається відкритим — не інструмент, а дані:
   реального розміченого корпусу ще немає (ragivka/session-indexer ще не
   почали накопичувати рядки з outcome). Пункт лишається у roadmap, поки
   такий корпус не з'явиться.

## Exploratory — ідея зі статті, не оцінена

2. **RFLP-граф (Neo4j) для повного D_const.** Поточний D_const —
   лексичний проксі (маркери/проза). Повний граф
   Requirement-Flow-Linkage-Property дав би структурний вимір густини
   обмежень замість лексичного наближення.
3. **Assisted-режим K_drift (LLM-парсер).** Поточний K_drift вимагає явної
   розмітки `[REQ-*] -> [FUN/LOG/PHY-*]`. LLM-асистований парсер міг би
   виводити трасування зі специфікацій без розмітки — ціна: втрата
   детермінізму deterministic-шару (REQ-CHK-01..06 явно вимагають zero-LLM).
4. **Інші прилади.** OpenAI/Anthropic API поруч з Ollama —
   `instrument.Generator` вже спроєктований як pluggable interface саме
   для цього.
5. **Інші проєкції.** SQL DDL, OpenAPI замість тільки Go type definitions
   як ціль генерації для `measure` — розширює застосовність за межі
   Go-проєктів.
