# CLAUDE.md — tumanomir

Вимірювальний інструмент точності специфікацій для AI-проєктів (CLI, Go).
Продуктизація методології статті «Джерело Невідомості».

## З чого починати в новій сесії

1. Прочитай `docs/requirements.md` — **специфікація первинна**; код пишеться
   під неї, а не навпаки. Вона в розмітці самого tumanomir
   (`[REQ-*] -> [FUN-*]`, `@schema`) — dogfooding.
2. `context/history.md` (у `../context/`, поза репо) — провенанс проєкту:
   звідки методологія, які рішення вже ухвалені й чому.
3. Поточний стан коду — spike детермінованого ядра + порт dispersion
   з експерименту статті; звіряй з requirements, розбіжність — це баг
   або в коді, або в requirements (спершу онови requirements).

## Методологічні інваріанти (не змінювати мовчки; спершу requirements)

- **D_pair** (1 − mean pairwise AST sim) — робоча метрика; **H** — лише
  ordinal-сигнал («один кластер чи багато»). H сатурує при малих N.
- Всі стохастичні виміри **instrument-relative**: конфігурація приладу
  (модель+версія, промпт, temp, N, think, num_ctx, поріг кластеризації)
  фіксується і друкується в кожному звіті.
- **invalid rate** звітується, не ховається (retry з лічильником discards).
- Пороги за замовчуванням (0.20/0.35/0.30) — **гіпотези** зі статті,
  не константи; в usage-тексті так і писати.
- Детермінований шар (`check`) — нуль мережі, нуль LLM. Git-hook-ready.
- Ollama: `think: false` для reasoning-моделей; `num_ctx` мусить вміщати
  промпт (мовчазне обрізання = баг цілісності виміру); `num_predict`
  вище природної довжини виходу.

## Побудова та перевірка

Бінарник збирається в `bin/` через `make`, не `go build`/`go run` напряму.

```bash
make build     # -> bin/tumanomir
make vet
make test
make dogfood   # bin/tumanomir check docs/requirements.md — dogfood-смоук
make lint      # golangci-lint run (потребує встановленого golangci-lint)
make ci        # build + vet + test + lint + dogfood, усе разом
```

## Конвенції

- Go ≥ 1.26, stdlib-only у v0.1 (без CLI-фреймворків і YAML-залежностей).
- Типи: спільні — `internal/types.go`; пакетні — `internal/<pkg>/` (типи
  вищого рівня мають пріоритет при конфліктах).
- Код/коментарі/повідомлення — English; спілкування в сесії — українська.
- Гілки: `<type>-<slug>` від main; у main напряму не комітити.
- Reference-дані для тестів dispersion: згенеровані файли з експерименту
  статті — `~/wrk/promo/source_of_the_unknown/sanity/out*/` (120 файлів,
  реальні еталонні числа в `sanity/README.md` там само).
