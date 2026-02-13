# Readings Service

Микросервис для управления показаниями водосчётчиков.

## Описание

Сервис обеспечивает:
- Приём показаний от абонентов
- Хранение истории показаний
- Валидацию показаний (проверка на нереалистичные значения)
- Верификацию показаний оператором
- Статистику потребления
- Группировку по периодам

## API

### Основные endpoints

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/health` | Health check |
| POST | `/api/v1/readings` | Создать показание |
| GET | `/api/v1/readings` | Список показаний (пагинация + фильтры) |
| GET | `/api/v1/readings/:id` | Получить по ID |
| GET | `/api/v1/readings/latest` | Последние показания |
| GET | `/api/v1/readings/monthly` | Показания по месяцам |
| GET | `/api/v1/readings/statistics` | Статистика потребления |
| PUT | `/api/v1/readings/:id` | Обновить показание |
| DELETE | `/api/v1/readings/:id` | Удалить показание |
| POST | `/api/v1/readings/:id/verify` | Верифицировать показание |

### Endpoints для абонента

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/v1/subscribers/:id/readings` | Показания абонента |
| GET | `/api/v1/subscribers/:id/readings/latest` | Последние показания |
| GET | `/api/v1/subscribers/:id/readings/monthly` | По месяцам |

### Endpoints для счётчика

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/v1/counters/:id/readings` | Показания счётчика |
| GET | `/api/v1/counters/:id/readings/latest` | Последнее показание |
| GET | `/api/v1/counters/:id/readings/statistics` | Статистика |

## Создание показания

**Request:**
```json
POST /api/v1/readings
{
  "subscriber_id": 1,
  "counter_id": 1,
  "value": 123.45,
  "reading_date": "2024-01-15"
}
```

**Response:**
```json
{
  "id": 1,
  "subscriber_id": 1,
  "counter_id": 1,
  "value": 123.45,
  "reading_date": "2024-01-15T00:00:00Z",
  "verified": false,
  "created_at": "2024-01-15T10:30:00Z"
}
```

## Валидация показаний

Сервис проверяет:
- Значение должно быть положительным
- Дата не может быть в будущем
- Дата не старше 90 дней (настраивается)
- Рост показаний не более 50% за период (настраивается)
- Отслеживание резкого снижения (для замены счётчиков)

## Статистика

**Request:**
```
GET /api/v1/readings/statistics?subscriber_id=1&counter_id=1
```

**Response:**
```json
{
  "subscriber_id": 1,
  "counter_id": 1,
  "current_value": 123.45,
  "previous_value": 100.00,
  "consumption": 23.45,
  "period": "2023-12"
}
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| PORT | 8083 | Порт сервиса |
| ENV | development | Окружение |
| DATABASE_URL | - | URL подключения к PostgreSQL |
| REDIS_URL | redis://localhost:6379 | URL подключения к Redis |

## Локальный запуск

```bash
# Установка зависимостей
go mod download

# Запуск
go run cmd/main.go

# Или через docker-compose
docker-compose -f ../../deployments/compose/docker-compose.yml up readings
```

## Тестирование

```bash
# Запуск тестов
go test ./...

# С покрытием
go test -cover ./...
```
