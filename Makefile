include .env
export

VERSION      := v1.0.0
GIT_HASH     := $(shell git rev-parse --short HEAD)
SERVICE      := infra-websocket-gateway-service
SRC          := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
TARGETS      := ${SERVICE}
TEST_TARGETS :=
ALL_TARGETS  := $(TARGETS) $(TEST_TARGETS)
CURR_DIR     := $(shell pwd)

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

.PHONY: help
help: ### Print this help screen.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	@git diff --exit-code

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

.PHONY: tidy
tidy: ### Format code and tidy mod file.
	@go fmt ./...
	@go mod tidy -v
	@go mod vendor

.PHONY: audit
audit: ### Run quality control checks.
	@go mod verify
	@go vet ./...
	@go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	@go test -race -buildvcs -vet=off ./...

.PHONY: pb-fmt
pb-fmt: ### Format proto files.
	@clang-format -i ${CURR_DIR}/protos/*.proto

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

ifeq ($(race), 1)
	BUILD_FLAGS := -race
endif

ifeq ($(gc_debug), 1)
	BUILD_FLAGS += -gcflags=all="-N -l"
endif

.PHONY: test
test: ### Run all tests.
	@env CI="true" \
		go test -count=1 -v -p 1 -race -buildvcs \
		$(shell go list ./... | grep -v /cmd | grep -v /internal/proto_gens) || true

.PHONY: test/cover
test/cover: ### Run all tests and display coverage.
	@env CI="true" \
		go test -count=1 -v -p 1 -race -buildvcs \
		$(shell go list ./... | grep -v /cmd | grep -v /internal/proto_gens) \
		-coverprofile unit_test_coverage.txt || true

.PHONY: build
build: clean tidy $(ALL_TARGETS) ### Build the application.

$(TARGETS): $(SRC)
	@GOOS=linux GOARCH=amd64 go build -mod vendor $(BUILD_FLAGS) $(CURR_DIR)/cmd/$@

$(TEST_TARGETS): $(SRC)
	@GOOS=linux GOARCH=amd64 go build -mod vendor $(BUILD_FLAGS) $(CURR_DIR)/test/$@

.PHONY: clean
clean: ### Clean the application.
	@rm -f $(ALL_TARGETS)

.PHONY: local_run
local_run: build ### Run the application locally.
	@${CURR_DIR}/${SERVICE} -conf ${CURR_DIR}/etc/${SERVICE}-dev.json 2>&1 | tee dev.log

# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

IMAGE_VERSION := ${VERSION}-${GIT_HASH}

.PHONY: image
image: confirm audit no-dirty ### Build the application image.
	@docker build -f ${CURR_DIR}/devops/docker/Dockerfile -t ${SERVICE}:${IMAGE_VERSION} .

.PHONY: check_compose
check_compose: ### Check the docker-compose configuration.
	@docker-compose -f "${CURR_DIR}/docker-compose.yml" config

.PHONY: run_compose
run_compose: confirm image check_compose ### Run the application with docker-compose.
	@mkdir -p ~/.infra-config/${SERVICE}
	@cp -f ${CURR_DIR}/etc/${SERVICE}-prod.json ~/.infra-config/${SERVICE}/${SERVICE}.json
	@mkdir -p ${CURR_DIR}/.logs
	@mkdir -p ${CURR_DIR}/.persistent
	@mkdir -p ${CURR_DIR}/.locks
	@mkdir -p ${CURR_DIR}/.shares
	@docker-compose -f "${CURR_DIR}/docker-compose.yml" up -d --build

.PHONY: shutdown_compose
shutdown_compose: confirm ### Shutdown the application with docker-compose.
	@docker-compose -f "${CURR_DIR}/docker-compose.yml" down

now=$(shell date "+%Y%m%d%H%M%S")
.PHONY: logs
logs: ### Show the logs of the running service.
	@docker logs -f ${SERVICE} 2>&1 | tee prod_${now}.log
