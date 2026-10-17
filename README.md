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
- Nginx, Docker Compose и Kubernetes.

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

## Minikube

Нужны Minikube и совместимая с кластером версия `kubectl`.

Полный запуск автоматизирован командой:

```bash
make help
make k8s-up
make k8s-status
make k8s-open
```

`make help` выводит все команды разработки и эксплуатации. Цели `k8s-images`, `k8s-load`, `k8s-validate` и `k8s-deploy` позволяют выполнить этапы запуска отдельно, а `make k8s-down` удаляет учебное окружение вместе с его данными. Ниже приведены соответствующие команды без автоматизации.

```bash
minikube start --driver=docker --container-runtime=docker --cpus=4 --memory=4096mb

docker build -t badgoodtune-backend:hw2 .
docker build -t badgoodtune-frontend:hw2 -f web/Dockerfile .
minikube image load --daemon --overwrite badgoodtune-backend:hw2
minikube image load --daemon --overwrite badgoodtune-frontend:hw2

kubectl apply -f deploy/kubernetes/00-namespace
kubectl apply -f deploy/kubernetes/10-configuration
kubectl apply -f deploy/kubernetes/20-database
kubectl rollout status statefulset/postgres -n badgoodtune --timeout=180s

kubectl apply -f deploy/kubernetes/30-migration
kubectl wait --for=condition=complete job/badgoodtune-migration -n badgoodtune --timeout=120s

kubectl apply -f deploy/kubernetes/40-services
kubectl apply -f deploy/kubernetes/50-application
kubectl rollout status deployment/badgoodtune-backend -n badgoodtune --timeout=180s
kubectl rollout status deployment/badgoodtune-frontend -n badgoodtune --timeout=180s

minikube service badgoodtune-frontend -n badgoodtune
```

Альтернативный способ — выполнить `eval "$(minikube docker-env)"` и собрать образы сразу во внутреннем Docker daemon. Тогда `minikube image load` не требуется. Эти способы не следует смешивать для одного тега: старый host-образ может перезаписать свежий образ кластера.

Масштабирование:

```bash
kubectl scale deployment/badgoodtune-backend --replicas=3 -n badgoodtune
kubectl scale deployment/badgoodtune-frontend --replicas=2 -n badgoodtune
kubectl get pods -n badgoodtune
```

Описание предметной области, 12 факторов и Kubernetes-развёртывания находится в [`Отчёт.md`](Отчёт.md).
