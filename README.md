# Водоканал - Микросервисная система

Микросервисная архитектура для системы управления абонентами водоканала.

## Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend (JS)                        │
│                      React + TypeScript                     │
└───────────────────────────┬─────────────────────────────────┘
                            │ HTTPS
┌───────────────────────────▼─────────────────────────────────┐
│                       API Gateway                            │
│              (Kong / Traefik / собственный)                  │
└─────┬───────┬───────┬───────┬───────┬───────┬───────┬─────────┘
      │       │       │       │       │       │       │
      ▼       ▼       ▼       ▼       ▼       ▼       ▼
   ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐
   │Auth│ │Subs│ │Read│ │Bill│ │Paym│ │Tckt│ │Cnts│ │Admn│
   └────┘ └────┘ └────┘ └────┘ └────┘ └────┘ └────┘ └────┘
```

## Микросервисы

| Сервис | Описание | Порт |
|--------|----------|------|
| gateway | API Gateway, маршрутизация, rate limiting | 8080 |
| auth | Авторизация, аутентификация, JWT, OAuth2 | 8081 |
| subscribers | Абоненты (ФИО, email, лицевой счёт, телефон) | 8082 |
| readings | Показания счётчиков, история | 8083 |
| billing | Начисления, квитанции | 8084 |
| payments | Интеграция с платёжными системами | 8085 |
| tickets | Обращения, заявки, поддержка | 8086 |
| counters | Учёт водосчётчиков | 8087 |
| admin | Панель оператора, отчёты | 8088 |

## Технологический стек

**Бэкенд (Go):**
- gin/fiber — HTTP framework
- gRPC + protobuf — межсервисное общение
- sqlc — SQL генератор
- wire — dependency injection
- pgx — PostgreSQL driver

**Frontend (JS):**
- React + TypeScript
- Vite
- TanStack Query
- React Router

**Инфраструктура:**
- PostgreSQL — основная БД
- Redis — кеш, сессии
- RabbitMQ — очереди сообщений

## Структура проекта

```
vodokanal/
├── services/           # Go микросервисы
├── web/               # Frontend (React)
├── shared/            # Общий код (proto, types)
├── deployments/       # Docker, K8s configs
└── docs/              # Документация
```

## Быстрый старт

```bash
# Запуск через Docker Compose
cd deployments/compose
docker-compose up -d

# Локальная разработка
make dev
```

## Разработка

См. [docs/development.md](docs/development.md)
