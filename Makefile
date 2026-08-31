# kno-examples.
#
# `make check` is the only target you need to remember. Everything else it
# runs is listed below and is independently runnable.
#
# Every target needs a released `kno` on PATH; `make install-kno` fetches one
# through the project's own install.sh, so a local run exercises the path a
# user exercises.

SHELL := /bin/sh
.DEFAULT_GOAL := help

KNO ?= $(shell command -v kno 2>/dev/null)
KNO_VERSION ?= latest
LOCAL_BIN := $(CURDIR)/bin

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: require-kno
require-kno:
	@if [ -z "$(KNO)" ]; then \
		echo "no kno on PATH. Run 'make install-kno', or set KNO=/path/to/kno."; \
		exit 1; \
	fi
	@echo "using $$($(KNO) --version)"

.PHONY: install-kno
install-kno: ## Install the latest released kno into ./bin
	@mkdir -p $(LOCAL_BIN)
	@curl -sSfL https://raw.githubusercontent.com/uknoAI/kno/main/install.sh \
		| KNO_INSTALL_DIR=$(LOCAL_BIN) KNO_VERSION=$(if $(filter latest,$(KNO_VERSION)),,$(KNO_VERSION)) sh
	@echo "add $(LOCAL_BIN) to your PATH, or run make with KNO=$(LOCAL_BIN)/kno"

.PHONY: check
check: lint flags scenarios test fmt shellcheck ## Everything

.PHONY: lint
lint: ## Front matter, credentials, and quoted-block fidelity. Needs no binary.
	go run ./cmd/verify lint

.PHONY: flags
flags: require-kno ## Every kno invocation against the released binary's own surface
	go run ./cmd/verify flags --kno $(KNO)

.PHONY: scenarios
scenarios: require-kno ## Every scenario end to end, twice, against committed expectations
	go run ./cmd/verify scenario --kno $(KNO) --repeat 2

.PHONY: fixtures
fixtures: ## The three copies of the demo fixtures: make fixtures KNO_SRC=../kno
	@if [ -z "$(KNO_SRC)" ]; then \
		echo "fixtures needs a checkout of uknoAI/kno: make fixtures KNO_SRC=../kno"; \
		exit 1; \
	fi
	go run ./cmd/verify fixtures --kno-src $(KNO_SRC)

.PHONY: update-expected
update-expected: require-kno ## Regenerate expected/*.json from a real run, preserving each projection's key set
	go run ./cmd/verify scenario --kno $(KNO) --update
	@echo "review the diff like code: a projection that grew is a projection that will start churning"

.PHONY: test
test: require-kno ## The runner's own tests, including the deliberately broken corpus
	KNO_BIN=$(KNO) go test -race -shuffle=on ./...

.PHONY: fmt
fmt: ## gofmt
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "gofmt: $$unformatted"; exit 1; fi

.PHONY: shellcheck
shellcheck: ## POSIX-sh lint on every script CI and readers run
	@command -v shellcheck >/dev/null 2>&1 || { echo "shellcheck not installed; skipping"; exit 0; }
	shellcheck --shell=sh scenarios/*/run.sh .github/scripts/*.sh

.PHONY: render
render: ## Print one recipe's verification block: make render RECIPE=recipes/zendesk.md
	go run ./cmd/verify render $(RECIPE)
