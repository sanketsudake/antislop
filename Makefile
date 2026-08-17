GOLANGCI ?= golangci-lint

.PHONY: build test lint dogfood smoke smoke-update deps-check plugin plugin-smoke gendocs gendocs-check check

build:
	go build -o bin/antislop ./cmd/antislop

test:
	go test ./...

lint:
	go vet ./...
	$(GOLANGCI) run

dogfood: build
	./bin/antislop ./...

smoke: build
	scripts/smoke.sh

smoke-update: build
	scripts/smoke.sh --update

deps-check:
	scripts/deps-check.sh

plugin:
	$(GOLANGCI) custom
	./custom-gcl run -c .golangci.dogfood.yml ./...

plugin-smoke:
	scripts/plugin-smoke.sh

gendocs:
	go run ./tools/gendocs

gendocs-check:
	go run ./tools/gendocs -check

check: lint test dogfood smoke deps-check gendocs-check
