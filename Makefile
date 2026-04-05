# ======================================================================
#  KVStore — Root Makefile
#  Operates on the full stack (server + web + Docker cluster).
#  Run individual server targets from inside server/ using its own Makefile.
# ======================================================================

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo ""
	@echo "  KVStore — root targets"
	@echo ""
	@echo "  Docker Cluster"
	@echo "    make docker-build    build all images locally"
	@echo "    make docker-up       build + start 3-node cluster and web dashboard"
	@echo "    make docker-down     stop and remove containers (keeps volumes)"
	@echo "    make docker-clean    stop, remove containers AND named volumes"
	@echo "    make docker-logs     tail logs from all services"
	@echo "    make docker-ps       show status of all services"
	@echo ""
	@echo "  Individual services"
	@echo "    make docker-node1    tail logs for node1 only"
	@echo "    make docker-web      tail logs for the web dashboard only"
	@echo ""
	@echo "  Server (delegates to server/Makefile)"
	@echo "    make test            run all Go tests with the race detector"
	@echo "    make build           build server and CLI binaries"
	@echo ""

# ── Docker Cluster ──────────────────────────────────────────────────────────

.PHONY: docker-build
docker-build:
	docker compose build

.PHONY: docker-up
docker-up:
	docker compose up --build -d
	@echo ""
	@echo "  Cluster is running:"
	@echo "    node1 HTTP API  → http://localhost:8080"
	@echo "    node1 TCP       → localhost:6379"
	@echo "    Dashboard       → http://localhost:3000"
	@echo ""

.PHONY: docker-down
docker-down:
	docker compose down

.PHONY: docker-clean
docker-clean:
	docker compose down -v
	@echo "  All containers and volumes removed."

.PHONY: docker-logs
docker-logs:
	docker compose logs -f

.PHONY: docker-ps
docker-ps:
	docker compose ps

.PHONY: docker-node1
docker-node1:
	docker compose logs -f node1

.PHONY: docker-web
docker-web:
	docker compose logs -f web

# ── Server (delegate) ───────────────────────────────────────────────────────

.PHONY: test
test:
	$(MAKE) -C server test

.PHONY: build
build:
	$(MAKE) -C server build
