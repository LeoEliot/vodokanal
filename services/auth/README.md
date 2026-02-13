# Auth Service

Микросервис авторизации и аутентификации для Водоканала.

## Описание

Сервис обеспечивает:
- Регистрацию новых пользователей
- Вход в систему (email/password)
- JWT токены (access + refresh)
- Обновление токенов
- Выход из системы
- Смену пароля

## API

### Методы

| Метод | Endpoint | Описание | Auth |
|-------|----------|----------|------|
| GET | `/health` | Health check | - |
| POST | `/api/v1/auth/register` | Регистрация | - |
| POST | `/api/v1/auth/login` | Вход | - |
| POST | `/api/v1/auth/refresh` | Обновить токен | - |
| POST | `/api/v1/auth/logout` | Выход | Bearer |
| POST | `/api/v1/auth/logout-all` | Выход со всех устройств | Bearer |
| POST | `/api/v1/auth/change-password` | Сменить пароль | Bearer |
| GET | `/api/v1/users/me` | Текущий пользователь | Bearer |

## Регистрация

**Request:**
```json
POST /api/v1/auth/register
{
  "email": "user@example.com",
  "password": "SecurePass123",
  "name": "Ivan Ivanov"
}
```

**Response:**
```json
{
  "user_id": 1,
  "email": "user@example.com",
  "role": "subscriber",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

## Вход

**Request:**
```json
POST /api/v1/auth/login
{
  "email": "user@example.com",
  "password": "SecurePass123"
}
```

**Response:**
```json
{
  "user_id": 1,
  "email": "user@example.com",
  "role": "subscriber",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

## Обновление токена

**Request:**
```json
POST /api/v1/auth/refresh
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

## Требования к паролю

- Минимум 8 символов
- Максимум 128 символов
- Хотя бы одна заглавная буква
- Хотя бы одна строчная буква
- Хотя бы одна цифра
- Не должен быть распространённым паролем

## Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| PORT | 8081 | Порт сервиса |
| ENV | development | Окружение |
| DATABASE_URL | - | URL подключения к PostgreSQL |
| REDIS_URL | redis://localhost:6379 | URL подключения к Redis |
| JWT_SECRET | - | Секрет для JWT (обязательно!) |
| ACCESS_TOKEN_DURATION | 15m | Время жизни access токена |
| REFRESH_TOKEN_DURATION | 168h | Время жизни refresh токена (7 дней) |

## Локальный запуск

```bash
# Установка зависимостей
go mod download

# Запуск
go run cmd/main.go

# Или через docker-compose
docker-compose -f ../../deployments/compose/docker-compose.yml up auth
```

## Тестирование

```bash
# Запуск тестов
go test ./...

# С покрытием
go test -cover ./...
```
