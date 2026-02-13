# Subscribers Service

Микросервис для управления абонентами водоканала.

## Описание

Сервис предоставляет REST API для управления данными абонентов, включая:
- Личные данные (ФИО, email, телефон)
- Лицевой счёт
- Адрес
- Поиск и фильтрацию

## API

### Методы

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/health` | Health check |
| GET | `/api/v1/subscribers` | Список абонентов (пагинация) |
| GET | `/api/v1/subscribers/:id` | Получить по ID |
| GET | `/api/v1/subscribers/account/:account_number` | Получить по лицевому счёту |
| GET | `/api/v1/subscribers/search?q=` | Поиск |
| POST | `/api/v1/subscribers` | Создать абонента |
| PUT | `/api/v1/subscribers/:id` | Обновить абонента |
| DELETE | `/api/v1/subscribers/:id` | Удалить абонента |

### Query параметры

- `page` - номер страницы (по умолчанию 1)
- `limit` - элементов на странице (по умолчанию 20, максимум 100)
- `q` - поисковый запрос (поиск по номеру счёта, ФИО, email, телефону)

## Модель данных

```json
{
  "id": 1,
  "account_number": "12345",
  "last_name": "Иванов",
  "first_name": "Иван",
  "middle_name": "Иванович",
  "email": "ivanov@example.com",
  "phone": "+7 (999) 123-45-67",
  "address": "г. Москва, ул. Ленина, д. 1, кв. 1",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

## Локальный запуск

```bash
# Установка зависимостей
go mod download

# Запуск
go run cmd/main.go

# Или через docker-compose
docker-compose -f ../../deployments/compose/docker-compose.yml up subscribers
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| PORT | 8082 | Порт сервиса |
| ENV | development | Окружение |
| DATABASE_URL | - | URL подключения к PostgreSQL |
| REDIS_URL | redis://localhost:6379 | URL подключения к Redis |
| JWT_SECRET | - | Секрет для JWT |

## Тестирование

```bash
# Запуск тестов
go test ./...

# С покрытием
go test -cover ./...
```
