.PHONY: help build up down restart logs clean test lint install-deps

# Default target
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m # No Color

help: ## Show this help message
	@echo '$(BLUE)Vodokanal - Makefile commands$(NC)'
	@echo ''
	@echo '$(GREEN)Development:$(NC)'
	@sed -n 's/^ \(.*\): ## \(.*\)/  \1\2/p' $(MAKEFILE_LIST) | column -t -s ':'
	@echo ''
	@echo '$(GREEN)Docker:$(NC)'
	@sed -n 's/^d\(.*\): ## \(.*\)/  d\1\2/p' $(MAKEFILE_LIST) | column -t -s ':'

# ==================== Development ====================

run: ## Run all services
	@echo "$(BLUE)Starting all services...$(NC)"
	docker-compose up --remove-orphans

run-bg: ## Run all services in background
	@echo "$(BLUE)Starting all services in background...$(NC)"
	docker-compose up -d --remove-orphans
	@echo "$(GREEN)All services started!$(NC)"
	@echo "Gateway: http://localhost:8080"
	@echo "Web: http://localhost:3000"
	@echo "RabbitMQ Management: http://localhost:15672 (vodokanal/vodokanal_password)"

logs: ## Show logs from all services
	docker-compose logs -f

logs-%: ## Show logs for specific service (e.g., make logs-auth)
	docker-compose logs -f $(subst logs-,,$@)

# ==================== Docker ====================

dup: docker-compose up -d ## Start all services in background

ddown: docker-compose down ## Stop all services

drestart: docker-compose restart ## Restart all services

dps: docker-compose ps ## Show running services

dbuild: ## Rebuild all services
	@echo "$(BLUE)Rebuilding all services...$(NC)"
	docker-compose build --no-cache

dprune: ## Remove unused Docker resources
	docker system prune -a --volumes

# ==================== Database ====================

db-shell: ## Open PostgreSQL shell
	docker-compose exec postgres psql -U vodokanal -d vodokanal

db-reset: ## Reset database (WARNING: deletes all data)
	@echo "$(YELLOW)This will delete all data!$(NC)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v; \
		docker-compose up -d postgres; \
		sleep 3; \
	fi

redis-cli: ## Open Redis CLI
	docker-compose exec redis redis-cli

# ==================== Utilities ====================

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf web/dist
	@echo "$(GREEN)Clean complete!$(NC)"

ps: ## Show running processes
	@docker-compose ps

health: ## Check health of all services
	@echo "$(BLUE)Checking service health...$(NC)"
	@echo "\n$(GREEN)Infrastructure:$(NC)"
	@curl -s http://localhost:8080/health > /dev/null && echo "✓ Gateway" || echo "✗ Gateway"
	@curl -s http://localhost:8081/health > /dev/null && echo "✓ Auth" || echo "✗ Auth"
	@curl -s http://localhost:8082/health > /dev/null && echo "✓ Subscribers" || echo "✗ Subscribers"
	@curl -s http://localhost:8083/health > /dev/null && echo "✓ Readings" || echo "✗ Readings"
	@curl -s http://localhost:8084/health > /dev/null && echo "✓ Billing" || echo "✗ Billing"
	@curl -s http://localhost:8085/health > /dev/null && echo "✓ Payments" || echo "✗ Payments"
	@curl -s http://localhost:8086/health > /dev/null && echo "✓ Tickets" || echo "✗ Tickets"
	@curl -s http://localhost:8087/health > /dev/null && echo "✓ Counters" || echo "✗ Counters"
	@curl -s http://localhost:8088/health > /dev/null && echo "✓ Notifications" || echo "✗ Notifications"
