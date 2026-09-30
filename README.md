# mp-reviews

Автоответчик и жалобы на отзывы Wildberries и Ozon. Что строим — [`SPEC.md`](SPEC.md), правила
работы — [`CLAUDE.md`](CLAUDE.md). Проект потом встраивается в `mp-collector`.

Сейчас сделан **этап 0 — пробник кабинетов**: [`cmd/mp-reviews-probe`](cmd/mp-reviews-probe/README.md).

## Раскладка

| Путь | Что там |
|---|---|
| `cmd/mp-reviews-probe/` | пробник кабинетов (Windows и Linux), инструкция и чек-лист для пользователя — в его README |
| `internal/cdp/` | минимальный клиент Chrome DevTools Protocol (свой websocket по RFC 6455, без зависимостей) |
| `internal/chrome/` | поиск Chrome, каталоги профилей кабинетов (`%LOCALAPPDATA%\mp-reviews\chrome-profiles`), запуск |
| `internal/probe/` | запись событий, вычистка секретов, `summary.md`, zip |
| `legacy/`, `fixtures/` | справочник по старым системам и тестовые данные (не меняются) |

Внешних зависимостей нет — только стандартная библиотека Go.

## Проверка

Нужны Go 1.26 (`toolchain go1.26.6` подтянется сам при `GOTOOLCHAIN=auto`), golangci-lint v2,
собранный Go ≥ 1.26, govulncheck и Chromium/Chrome для приёмочных тестов пробника:

```
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
# Chromium: apt-get install chromium  (или любой Chrome; путь — в MP_REVIEWS_CHROME_PATH)
```

| Команда | Что делает |
|---|---|
| `make test` | `go test ./... -race`, включая тесты пробника в настоящем headless Chromium на локальной странице |
| `make lint` | `golangci-lint run ./...` с `.golangci.yml` |
| `make vuln` | `govulncheck ./...` (нужен доступ к vuln.go.dev) |
| `make build-windows` | сборка `mp-reviews-probe.exe` под Windows amd64 в `bin/windows-amd64/` |
| `make test-integration` | на этапе 0 пустая: БД не используется, интеграционные тесты с PostgreSQL — с этапа 1 |

Тесты пробника ищут браузер так: `MP_REVIEWS_CHROME_PATH` → `google-chrome`/`chromium`/… в `PATH` →
браузеры Playwright (`$PLAYWRIGHT_BROWSERS_PATH`, `/opt/pw-browsers`). Если браузер не найден, эти два
теста пропускаются (`SKIP`) — смотрите вывод `go test -v`. Под root Chromium запускается с `--no-sandbox`
(только в тестах).

Переменные окружения — в [`.env.example`](.env.example).
