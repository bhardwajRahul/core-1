ENV_FILE ?= .env
GO_TEST_ENV ?= GOCACHE=/tmp/go-build
DOCKER_COMPOSE ?= $(shell if docker compose version >/dev/null 2>&1; then echo docker compose; else echo docker-compose; fi)
TEST_COMPOSE_PROJECT ?= staticbackend-test
TEST_COMPOSE_FILE ?= docker-compose-unittest.yml
TEST_COMPOSE = $(DOCKER_COMPOSE) -p $(TEST_COMPOSE_PROJECT) -f $(TEST_COMPOSE_FILE)

-include $(ENV_FILE)
export $(shell test -f $(ENV_FILE) && sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=.*/\1/p' $(ENV_FILE))

cleanup:
	@rm -rf dev.db && rm -rf backend/dev.db
	@rm -rf *.fts && rm -rf backend/*.fts

build:
	@cd cmd && rm -rf staticbackend && go build \
	-ldflags "-X github.com/staticbackendhq/core/config.BuildTime=$(shell date +'%Y-%m-%d.%H:%M:%S') \
	-X github.com/staticbackendhq/core/config.CommitHash=$(shell git log --pretty=format:'%h' -n 1) \
	-X github.com/staticbackendhq/core/config.Version=$(shell git describe --tags)" \
	-o staticbackend

plugin:
	@cd plugins/topdf && go build -buildmode=plugin -o ../topdf.so

start: build plugin
	@./cmd/staticbackend

alltest:
	@go clean -testcache && go test --cover ./...

test-local: cleanup
	@$(GO_TEST_ENV) go clean -testcache
	@$(GO_TEST_ENV) go test --cover . ./backend/... ./database/memory ./database/sqlite ./email ./extra ./function ./internal ./internal/query ./model ./search ./storage

thistest:
	go test -run $(TESTNAME) --cover

test-core: cleanup
	@go clean -testcache && go test --cover

test-pg:
	@cd database/postgresql && go test --race --cover

test-mdb:
	@cd database/mongo && go test --race --cover

test-mem:
	@rm -rf database/memory/mem.db
	@go test --race --cover ./database/memory

test-sqlite:
	@cd database/sqlite && go test --cover

test-dbs: test-pg test-mdb test-mem test-sqlite
	@echo ""

test-backend:
	@go test --cover ./backend/...

test-cache:
	@go test --cover ./cache/...

test-storage:
	@go test --cover ./storage/...

test-email:
	@go test --cover ./email/...

test-intl:
	@go test --cover ./internal

test-extra:
	@go test --cover ./extra

test-search:
	@cd search && rm -rf testdata && go test --race --cover

test-components: test-backend test-cache test-storage test-intl test-extra test-search
	@echo ""

test-services-up:
	@$(TEST_COMPOSE) up -d

test-services-wait:
	@echo "Waiting for PostgreSQL..."
	@timeout 60 sh -c 'until cid=$$($(TEST_COMPOSE) ps -q db) && [ -n "$$cid" ] && [ "$$(docker inspect -f "{{.State.Health.Status}}" "$$cid" 2>/dev/null)" = "healthy" ]; do sleep 1; done' || ($(TEST_COMPOSE) ps; $(TEST_COMPOSE) logs db; false)
	@echo "Waiting for MongoDB..."
	@timeout 60 sh -c 'until cid=$$($(TEST_COMPOSE) ps -q mongo) && [ -n "$$cid" ] && [ "$$(docker inspect -f "{{.State.Health.Status}}" "$$cid" 2>/dev/null)" = "healthy" ]; do sleep 1; done' || ($(TEST_COMPOSE) ps; $(TEST_COMPOSE) logs mongo; false)
	@echo "Waiting for Redis..."
	@timeout 60 sh -c 'until cid=$$($(TEST_COMPOSE) ps -q redis) && [ -n "$$cid" ] && [ "$$(docker inspect -f "{{.State.Health.Status}}" "$$cid" 2>/dev/null)" = "healthy" ]; do sleep 1; done' || ($(TEST_COMPOSE) ps; $(TEST_COMPOSE) logs redis; false)

test-services-down:
	@$(TEST_COMPOSE) down --remove-orphans

test-services-clean:
	@$(TEST_COMPOSE) down -v --remove-orphans

test-cache-flush:
	@$(TEST_COMPOSE) exec -T redis redis-cli FLUSHDB

test-ci-local: test-services-up test-services-wait
	@$(MAKE) ENV_FILE=.env.test.pg build
	@$(MAKE) ENV_FILE=.env.test.pg plugin
	@$(MAKE) ENV_FILE=.env.test.pg test-core
	@$(MAKE) test-cache-flush
	@$(MAKE) ENV_FILE=.env.test.mongo test-core
	@$(MAKE) ENV_FILE=.env.test.pg test-dbs
	@$(MAKE) ENV_FILE=.env.test.pg test-components

test-ci-local-clean:
	@set -e; trap '$(MAKE) test-services-clean' EXIT; $(MAKE) test-ci-local


stripe-dev:
	stripe listen --forward-to http://localhost:8099/stripe

lint:
	@golangci-lint run --timeout=10m

docker: build
	docker build . -t staticbackend:latest

pkg: build
	@mkdir -p dist
	@rm -rf dist/*
	@echo "building linux binaries"
	@cd cmd && CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o ../dist/binary-for-linux-64-bit
	@cd cmd && CGO_ENABLED=0 GOARCH=386 GOOS=linux go build -o ../dist/binary-for-linux-32-bit
	@echo "building mac binaries"
	@cd cmd && CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -o ../dist/binary-for-intel-mac-64-bit
	@cd cmd && CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -o ../dist/binary-for-arm-mac-64-bit
	@echo "building windows binaries"
	@cd cmd && CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -o ../dist/binary-for-windows-64-bit.exe
	@echo "copying plugins (if any)"
	@cp plugins/*.so dist/ 2>/dev/null || true
	@echo "compressing binaries"
	@gzip dist/*
