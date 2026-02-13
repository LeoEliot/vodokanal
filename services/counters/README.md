# Counters Service

Микросервис для учёта водосчётчиков.

## Описание

Сервис обеспечивает:
- Регистрацию водосчётчиков
- Учёт installation и verification дат
- Отслеживание поверок (верификаций)
- Активацию/деактивацию счётчиков
- Статусы и даты следующей поверки
- Dashboard статистику

## API

### Основные endpoints

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/health` | Health check |
| POST | `/api/v1/counters` | Создать счётчик |
| GET | `/api/v1/counters` | Список счётчиков (пагинация + фильтры) |
| GET | `/api/v1/counters/:id` | Получить по ID |
| GET | `/api/v1/counters/dashboard` | Dashboard статистика |
| PUT | `/api/v1/counters/:id` | Обновить счётчик |
| DELETE | `/api/v1/counters/:id` | Удалить счётчик |
| POST | `/api/v1/counters/:id/activate` | Активировать |
| POST | `/api/v1/counters/:id/deactivate` | Деактивировать |
| POST | `/api/v1/counters/:id/verify` | Верифицировать (поверка) |
| GET | `/api/v1/counters/:id/status` | Статус счётчика |

### Endpoints для абонента

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/v1/subscribers/:id/counters` | Все счётчики абонента |
| GET | `/api/v1/subscribers/:id/counters/active` | Активные счётчики |
| GET | `/api/v1/subscribers/:id/counters/statuses` | Статусы всех счётчиков |

## Создание счётчика

**Request:**
```json
POST /api/v1/counters
{
  "subscriber_id": 1,
  "serial_number": "ABCD-12345",
  "type": "cold",
  "installation_date": "2024-01-15",
  "initial_value": 100.0
}
```

**Response:**
```json
{
  "id": 1,
  "subscriber_id": 1,
  "serial_number": "ABCD-12345",
  "type": "cold",
  "installation_date": "2024-01-15T00:00:00Z",
  "is_active": true,
  "initial_value": 100.0,
  "created_at": "2024-01-15T10:30:00Z"
}
```

## Валидация

Сервис проверяет:
- Серийный номер: 4-30 символов, заглавные буквы и цифры
- Тип счётчика: `cold` или `hot`
- Дата установки: не в будущем, не старше 50 лет
- Уникальность серийного номера

## Статус счётчика

**Request:**
```
GET /api/v1/counters/1/status
```

**Response:**
```json
{
  "counter_id": 1,
  "serial_number": "ABCD-12345",
  "type": "cold",
  "is_active": true,
  "needs_verification": false,
  "days_until_verification": 1200
}
```

## Dashboard

**Request:**
```
GET /api/v1/counters/dashboard
```

**Response:**
```json
{
  "total_counters": 1234,
  "active_counters": 1200,
  "cold_counters": 700,
  "hot_counters": 534,
  "needs_verification": 45,
  "by_type": {
    "cold": 700,
    "hot": 534
  },
  "by_status": {
    "active": 1200,
    "inactive": 34
  }
}
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| PORT | 8087 | Порт сервиса |
| ENV | development | Окружение |
| DATABASE_URL | - | URL подключения к PostgreSQL |
| REDIS_URL | redis://localhost:6379 | URL подключения к Redis |
| VERIFICATION_DAYS | 1460 | Дней до обязательной поверки (4 года) |

## Локальный запуск

```bash
# Установка зависимостей
go mod download

# Запуск
go run cmd/main.go

# Или через docker-compose
docker-compose -f ../../deployments/compose/docker-compose.yml up counters
```

## Тестирование

```bash
# Запуск тестов
go test ./...

# С покрытием
go test -cover ./...
```
