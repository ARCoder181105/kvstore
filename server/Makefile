# ============================================================
#  KVStore — Makefile
#  Run all targets from inside the server/ directory.
# ============================================================

# Binary output names
SERVER_BIN  := kvstore
CLI_BIN     := kvcli

# Entry-point packages
SERVER_PKG  := ./cmd/server
CLI_PKG     := ./cmd/kvcli

# All Go source packages (used for test / vet / fmt)
ALL_PKGS    := ./...

# Data directory created at runtime
DATA_DIR    := ./data

# ── Default target ───────────────────────────────────────────
.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo ""
	@echo "  KVStore — available targets"
	@echo ""
	@echo "  Build"
	@echo "    make build          build both server and cli binaries"
	@echo "    make build-server   build the TCP server binary  (./kvstore)"
	@echo "    make build-cli      build the CLI client binary  (./kvcli)"
	@echo ""
	@echo "  Run"
	@echo "    make run-server     build + start the server (data/ auto-created)"
	@echo "    make run-cli        open the interactive REPL"
	@echo ""
	@echo "  Test & Quality"
	@echo "    make test           run all tests with the race detector"
	@echo "    make test-verbose   same, with -v flag"
	@echo "    make test-cover     run tests and open an HTML coverage report"
	@echo "    make fmt            format all Go source files"
	@echo "    make vet            run go vet across all packages"
	@echo "    make lint           run go vet + shadow checker (no extra tools needed)"
	@echo ""
	@echo "  Clean"
	@echo "    make clean          remove built binaries"
	@echo "    make clean-data     remove the data/ directory (AOF log + snapshot)"
	@echo "    make clean-all      clean + clean-data"
	@echo ""

# ── Build ────────────────────────────────────────────────────
.PHONY: build
build: build-server build-cli
	@echo "  built:  ./$(SERVER_BIN)  ./$(CLI_BIN)"

.PHONY: build-server
build-server:
	@echo "  building server..."
	go build -o $(SERVER_BIN) $(SERVER_PKG)

.PHONY: build-cli
build-cli:
	@echo "  building cli..."
	go build -o $(CLI_BIN) $(CLI_PKG)

# ── Run ──────────────────────────────────────────────────────
.PHONY: run-server
run-server: build-server
	@mkdir -p $(DATA_DIR)
	@echo "  starting kvstore on :6379  (Ctrl+C to stop)"
	./$(SERVER_BIN)

.PHONY: run-cli
run-cli: build-cli
	@echo "  opening kvcli REPL  (type HELP for commands)"
	./$(CLI_BIN)

# ── Test & Quality ───────────────────────────────────────────
.PHONY: test
test:
	@echo "  running tests with race detector..."
	go test -race -count=1 $(ALL_PKGS)

.PHONY: test-verbose
test-verbose:
	@echo "  running tests (verbose) with race detector..."
	go test -v -race -count=1 $(ALL_PKGS)

.PHONY: test-cover
test-cover:
	@echo "  running tests with coverage..."
	go test -race -count=1 -coverprofile=coverage.out $(ALL_PKGS)
	go tool cover -html=coverage.out -o coverage.html
	@echo "  coverage report: coverage.html"

.PHONY: fmt
fmt:
	@echo "  formatting..."
	go fmt $(ALL_PKGS)

.PHONY: vet
vet:
	@echo "  running vet..."
	go vet $(ALL_PKGS)

.PHONY: lint
lint: fmt vet
	@echo "  lint done"

# ── Clean ────────────────────────────────────────────────────
.PHONY: clean
clean:
	@echo "  removing binaries..."
	rm -f $(SERVER_BIN) $(CLI_BIN) coverage.out coverage.html

.PHONY: clean-data
clean-data:
	@echo "  removing data/ (AOF log + snapshot)..."
	rm -rf $(DATA_DIR)

.PHONY: clean-all
clean-all: clean clean-data
	@echo "  all clean"