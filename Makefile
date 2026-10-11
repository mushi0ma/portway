SHELL := /bin/sh
.PHONY: init-env bootstrap-check fmt fmt-check lint test-unit test-contract test-integration build-server verify docker-test docker-smoke docker-up docker-down migrate run-api smoke check-live

init-env:
	python3 scripts/init-env.py

bootstrap-check:
	go version
	docker version --format '{{.Server.Version}}'
	docker compose version
fmt:
	gofmt -w cmd internal tests api
fmt-check:
	@test -z "$$(gofmt -l cmd internal tests api)" || { gofmt -l cmd internal tests api; exit 1; }
lint:
	go vet ./...
test-unit:
	go test -race -count=1 ./internal/...
test-contract:
	go test -count=1 ./api/...
test-integration:
	@test -n "$$TEST_DATABASE_URL" || { echo 'Use make docker-test or set disposable TEST_DATABASE_URL'; exit 1; }
	PORTWAY_REQUIRE_INTEGRATION=1 go test -race -count=1 -v ./tests/integration/...
build-server:
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -o bin/portway ./cmd/portway
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -o bin/events ./cmd/events
verify: fmt-check lint test-unit test-contract test-integration build-server
docker-test:
	@sh scripts/docker-test.sh
docker-smoke:
	@sh scripts/docker-smoke.sh
docker-up:
	docker compose up --build -d
docker-down:
	docker compose down
migrate:
	docker compose run --rm migrate
run-api:
	go run ./cmd/portway api
smoke:
	@docker compose exec -T api /portway probe
	@echo 'API health probe passed'
check-live:
	python3 scripts/check-live.py
