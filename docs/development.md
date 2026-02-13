# Руководство по разработке

## Установка

### Требования
- Go 1.23+
- Node.js 20+
- Docker & Docker Compose
- PostgreSQL 16+
- Redis 7+
- RabbitMQ 3+

### Локальная разработка

1. **Запуск инфраструктуры:**
```bash
make docker-up
```

2. **Установка зависимостей:**
```bash
# Go модули
cd services/gateway && go mod download
cd ../auth && go mod download
# ... и так далее для всех сервисов

# Frontend
cd web
npm install
```

3. **Запуск сервисов:**
```bash
# Каждый сервис в своём терминале
cd services/gateway && go run cmd/main.go
cd services/auth && go run cmd/main.go
# ... и так далее
```

## Структура Go сервиса

```
service-name/
├── cmd/              # Точка входа приложения
│   └── main.go
├── internal/         # Внутренний код (не экспортируется)
│   ├── handler/     # HTTP обработчики
│   ├── service/     # Бизнес-логика
│   ├── repository/  # Работа с БД
│   └── model/       # Модели данных
├── pkg/             # Публичные библиотеки
├── config/          # Конфигурация
├── go.mod
└── Dockerfile
```

## Конвенции

### API endpoints
- Используйте RESTful принципы
- Все API версиируйте: `/api/v1/...`
- Возвращайте JSON ответы
- Используйте правильные HTTP статус-коды

### База данных
- Используйте sqlc для типобезопасных SQL запросов
- Миграции храните в `migrations/`
- Названия таблиц в snake_case

### Логирование
- Используйте `zap` для структурированного логирования
- Логируйте все внешние вызовы
- Не логируйте чувствительные данные

## Тестирование

```bash
# Запуск тестов всех сервисов
make test

# Запуск тестов конкретного сервиса
cd services/auth && go test ./...
```

## Сборка

```bash
# Сборка всех сервисов
make build

# Сборка конкретного сервиса
cd services/gateway && go build -o ../../bin/gateway ./cmd/...
```
