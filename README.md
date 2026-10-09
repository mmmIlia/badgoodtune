# BadGoodTune

Каталог музыкальных альбомов и рецензий. Альбом добавляется в общий каталог один раз, после чего пользователи могут находить его, ставить оценки и писать рецензии.

## Запуск

Нужен Docker с Compose v2.

```bash
docker compose up --build -d --wait
```

Приложение будет доступно на <http://localhost:8080>. Проверить полный сценарий регистрации и CRUD можно командой:

```bash
./scripts/smoke.sh
```

Остановка:

```bash
docker compose down
```

Локальные настройки можно переопределить через `.env`; пример находится в `.env.example`.

## Стек

- Go, `net/http`, OpenAPI и oapi-codegen;
- PostgreSQL, pgx, sqlc и goose;
- Vue 3, TypeScript и Vite;
- Docker Compose.

Backend разделён на независимые слои:

```text
internal/domain              бизнес-сущности и правила
internal/usecase             сценарии приложения и интерфейсы репозиториев
internal/transport/http      HTTP-обработчики и OpenAPI-типы
internal/repository/postgres реализация хранения в PostgreSQL
internal/app                 сборка и запуск приложения
```

Зависимости направлены внутрь: `domain` не знает о БД и HTTP, а `usecase` зависит только от `domain`. Правило проверяется архитектурным тестом.

## Разработка

```bash
make generate       # обновить код из OpenAPI и SQL
make fmt-check      # проверить форматирование
make vet            # статический анализ Go
make test           # модульные и HTTP-тесты
make test-integration
make web-check      # lint, typecheck, tests, build
make compose-smoke  # проверить собранное приложение
```

Описание предметной области и соответствия 12 факторам находится в [`Отчёт.md`](Отчёт.md).
