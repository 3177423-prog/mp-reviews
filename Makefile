# Цели качества (CLAUDE.md). Инструменты: Go 1.26, golangci-lint v2, govulncheck — см. README.md.

GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= govulncheck

.PHONY: all test test-integration lint vuln build build-windows check

all: check

# Юнит-тесты, включая приёмочные тесты пробника в headless Chromium
# (путь к браузеру — MP_REVIEWS_CHROME_PATH, иначе поиск в PATH и в браузерах Playwright).
test:
	$(GO) test ./... -race

# Этап 0 не использует БД: интеграционные тесты с PostgreSQL появляются с этапа 1 (SPEC §11).
test-integration:
	@echo "test-integration: интеграционных тестов на этапе 0 нет (БД не используется), появятся на этапе 1"

lint:
	$(GOLANGCI_LINT) run ./...

vuln:
	$(GOVULNCHECK) ./...

build:
	$(GO) build -o bin/ ./cmd/...

build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -o bin/windows-amd64/ ./cmd/mp-reviews-probe

check: test lint vuln build build-windows
