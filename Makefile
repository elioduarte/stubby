# ==================================================================================== #
# HELPERS
# ==================================================================================== #

LDFLAGS := -s -w
BIN_NAME = stubby
BIN_DIR = ./bin
PKG     = ./cmd/api 
args   ?=

## help: print this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
.PHONY: help

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy modfile
tidy:
	go run golang.org/x/tools/cmd/goimports@latest -l -w .
	go run mvdan.cc/gofumpt@latest -l -w .
	go mod tidy -v
.PHONY: tidy

## audit: run quality control checks
audit:
	go mod verify
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
.PHONY: audit

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## test: run all tests
.PHONY: test
test:
	go test -v -race -shuffle=on -vet=off -buildvcs ./...

## test/cover: run all tests and display coverage
test/cover:
	go test -v -race -shuffle=on -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out
.PHONY: test/cover

## build: build the application
build: tidy
	mkdir -p ${BIN_DIR}
	CGO_ENABLED=0 GOOS=linux go build -ldflags="${LDFLAGS}" -o ${BIN_DIR}/${BIN_NAME}-amd64 ${PKG} 
	CGO_ENABLED=0 GOOS=darwin go build -ldflags="${LDFLAGS}" -o ${BIN_DIR}/${BIN_NAME}-darwin ${PKG} 
.PHONY: build
	
## run: run the application
run: 
	go run ${PKG} ${args}
.PHONY: run

## run/live: run the application with reloading on file changes
run/live:
	go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "go build -o ${BIN_DIR}/${BIN_NAME} ${PKG}" \
		--build.bin "${BIN_DIR}/${BIN_NAME}" \
		--build.args_bin "${args}" \
		--build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go, tpl, tmpl, html, css, scss, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico" \
		--misc.clean_on_exit "true"
.PHONY: run/live
