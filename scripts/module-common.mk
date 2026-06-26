ROOT_DIR ?= $(shell git rev-parse --show-toplevel 2>/dev/null)
ifeq ($(ROOT_DIR),)
$(error No se pudo resolver ROOT_DIR desde este modulo)
endif

MODULE_PREFIX = github.com/EduGoGroup/wapp-shared/
MODULE_PATH ?= $(patsubst $(MODULE_PREFIX)%,%,$(MODULE_NAME))

RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[0;33m
BLUE = \033[0;34m
NC = \033[0m

.PHONY: help
help: ## Mostrar ayuda
	@echo "$(BLUE)$(MODULE_NAME) - Comandos disponibles:$(NC)"
	@echo ""
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-18s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Uso de versionado: make changelog VERSION=v0.1.0 | make release VERSION=v0.1.0$(NC)"

.PHONY: build
build: ## Verificar que el modulo compila
	@echo "$(BLUE)Compilando $(MODULE_NAME)...$(NC)"
	@go build ./...
	@echo "$(GREEN)Compilacion exitosa$(NC)"

.PHONY: test
test: ## Ejecutar tests unitarios
	@echo "$(BLUE)Ejecutando tests...$(NC)"
	@go test -short -v ./...
	@echo "$(GREEN)Tests completados$(NC)"

.PHONY: test-all
test-all: ## Ejecutar todos los tests del modulo
	@echo "$(BLUE)Ejecutando todos los tests...$(NC)"
	@go test -v ./...
	@echo "$(GREEN)Tests completados$(NC)"

.PHONY: test-race
test-race: ## Ejecutar tests con race detection
	@echo "$(BLUE)Ejecutando tests con race detection...$(NC)"
	@go test -race -short -v ./...
	@echo "$(GREEN)Tests con race detection completados$(NC)"

.PHONY: lint
lint: ## Ejecutar golangci-lint
	@echo "$(BLUE)Ejecutando linter...$(NC)"
	@golangci-lint run ./...
	@echo "$(GREEN)Linter completado$(NC)"

.PHONY: fmt
fmt: ## Formatear codigo
	@echo "$(BLUE)Formateando codigo...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)Codigo formateado$(NC)"

.PHONY: vet
vet: ## Ejecutar go vet
	@echo "$(BLUE)Ejecutando go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)Analisis completado$(NC)"

.PHONY: tidy
tidy: ## Ejecutar go mod tidy
	@echo "$(BLUE)Ejecutando go mod tidy...$(NC)"
	@go mod tidy
	@echo "$(GREEN)go.mod limpio$(NC)"

.PHONY: deps
deps: ## Actualizar dependencias del modulo
	@echo "$(BLUE)Actualizando dependencias...$(NC)"
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)Dependencias actualizadas$(NC)"

.PHONY: check
check: fmt vet lint test build ## Ejecutar validacion completa del modulo

.PHONY: changelog
changelog: ## Generar entrada versionada en CHANGELOG.md (requiere VERSION=vX.Y.Z)
	@if [ -z "$(VERSION)" ]; then \
		echo "$(RED)Debes indicar VERSION=vX.Y.Z$(NC)"; \
		exit 1; \
	fi
	@"$(ROOT_DIR)/scripts/update-module-changelog.sh" --module "$(MODULE_PATH)" --version "$(VERSION)"

.PHONY: changelog-dry-run
changelog-dry-run: ## Simular actualizacion de changelog (requiere VERSION=vX.Y.Z)
	@if [ -z "$(VERSION)" ]; then \
		echo "$(RED)Debes indicar VERSION=vX.Y.Z$(NC)"; \
		exit 1; \
	fi
	@"$(ROOT_DIR)/scripts/update-module-changelog.sh" --module "$(MODULE_PATH)" --version "$(VERSION)" --dry-run

.PHONY: release
release: check ## Crear y empujar el tag del modulo para disparar el GitHub release
	@if [ -z "$(VERSION)" ]; then \
		echo "$(RED)Debes indicar VERSION=vX.Y.Z$(NC)"; \
		exit 1; \
	fi
	@"$(ROOT_DIR)/scripts/module-release.sh" --module "$(MODULE_PATH)" --version "$(VERSION)"

.PHONY: release-dry-run
release-dry-run: check ## Simular el release del modulo
	@if [ -z "$(VERSION)" ]; then \
		echo "$(RED)Debes indicar VERSION=vX.Y.Z$(NC)"; \
		exit 1; \
	fi
	@"$(ROOT_DIR)/scripts/module-release.sh" --module "$(MODULE_PATH)" --version "$(VERSION)" --dry-run

.PHONY: clean
clean: ## Limpiar cache de tests
	@go clean -testcache
	@echo "$(GREEN)Limpieza completada$(NC)"

.DEFAULT_GOAL := help
