SHELL := /bin/bash
.DEFAULT_GOAL := help

SERVER_DIR := apps/server
FRONTEND_DIR := apps/frontend

.PHONY: help setup dev dev-server dev-web test test-server test-web lint lint-server lint-web build build-server build-web fmt api-types

help:
	@printf '%s\n' \
	  'Oryn development targets:' \
	  '  make setup       Install toolchain dependencies (mise + npm)' \
	  '  make dev         Run API server and Vite dev server' \
	  '  make dev-server  Run only the Go API server' \
	  '  make dev-web     Run only the Vite dev server' \
	  '  make test        Run all Go and frontend tests' \
	  '  make lint        Run Go vet and frontend lint/type-check' \
	  '  make build       Build the frontend and server binary' \
	  '  make api-types   Regenerate TypeScript API types from Go' \
	  '  make fmt         Format Go and frontend sources'

setup:
	mise install
	cd $(FRONTEND_DIR) && npm install

dev:
	@trap 'kill 0' EXIT INT TERM; \
	  (cd $(SERVER_DIR) && go run ./cmd/oryn-server) & \
	  (cd $(FRONTEND_DIR) && npm run dev) & \
	  wait

dev-server:
	cd $(SERVER_DIR) && go run ./cmd/oryn-server

dev-web:
	cd $(FRONTEND_DIR) && npm run dev

test: test-server test-web

test-server:
	cd $(SERVER_DIR) && go test ./...

test-web:
	cd $(FRONTEND_DIR) && npm run test -- --run

lint: lint-server lint-web

lint-server:
	cd $(SERVER_DIR) && go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
	  cd $(SERVER_DIR) && golangci-lint run ./...; \
	else \
	  echo "golangci-lint not installed; skipped"; \
	fi

lint-web:
	cd $(FRONTEND_DIR) && npm run lint && npm run typecheck

build: build-web build-server

build-web:
	cd $(FRONTEND_DIR) && npm run build

build-server:
	mkdir -p $(SERVER_DIR)/bin
	cd $(SERVER_DIR) && go build -trimpath -o bin/oryn-server ./cmd/oryn-server

api-types:
	cd $(SERVER_DIR) && tygo generate --config tygo.yaml

fmt:
	cd $(SERVER_DIR) && gofmt -w .
	cd $(FRONTEND_DIR) && npm run format
