# BadGoodTune

Каталог музыкальных альбомов и рецензий. Альбом добавляется в общий каталог один раз, после чего пользователи могут находить его, ставить оценки и писать рецензии.

## Запуск

Нужен Docker с Compose v2; для smoke-проверки — `curl` и `jq`.

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

Нужны Go из `go.mod` и Node.js 24 LTS из `.nvmrc`. Тесты эксплуатационных скриптов также используют `curl` и `jq`.

```bash
make generate       # обновить код из OpenAPI и SQL
make fmt-check      # проверить форматирование
make vet            # статический анализ Go
make test           # модульные и HTTP-тесты
make test-integration
make web-check      # lint, typecheck, tests, build
make web-check-container # lint, tests, build в закреплённом Node-контейнере
make compose-smoke  # проверить собранное приложение
```

## Minikube

Нужны Minikube, Helm, kubeconform, jq и совместимая с кластером версия `kubectl`.

Полный запуск автоматизирован командой:

```bash
make help
make k8s-up
make k8s-status
make k8s-open
```

`make help` выводит все команды разработки и эксплуатации. Цели `k8s-images`, `k8s-load`, `k8s-validate` и `k8s-deploy` позволяют выполнить этапы запуска отдельно, а `make k8s-down` удаляет локальное окружение вместе с его данными. Ниже приведены соответствующие команды без автоматизации.

```bash
minikube start --driver=docker --container-runtime=docker --cpus=4 --memory=4096mb

docker build -t badgoodtune-backend:local .
docker build -t badgoodtune-frontend:local -f web/Dockerfile .
minikube image load --daemon --overwrite badgoodtune-backend:local
minikube image load --daemon --overwrite badgoodtune-frontend:local

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

## Логи в Minikube

VictoriaLogs и Vector разворачиваются вместе с приложением:

```bash
make stack-up
make logs-status
make logs-generate
make logs-open
```

Последняя команда держит port-forward открытым. После неё vmui доступен на <http://127.0.0.1:9428/select/vmui/>. Выполнить LogsQL-запрос из терминала можно отдельно:

```bash
make logs-query LOGS_QUERY='_time:15m service.name:"badgoodtune-backend"'
```

`stack-up` также разворачивает vmalert и Alertmanager. Проверить алерт можно командами:

```bash
make alerts-trigger
make alerts-open
```

Первая команда создаёт тестовый трафик и дожидается состояния `firing`, вторая открывает vmalert UI на <http://127.0.0.1:8880/>. `make stack-down` удаляет оба namespace вместе с данными PostgreSQL и VictoriaLogs, затем останавливает Minikube.

## Метрики

`make stack-up` также устанавливает VictoriaMetrics и postgres-exporter. Для обновления уже работающего приложения:

```bash
make k8s-backend-reload
make metrics-deploy
make metrics-check
make metrics-generate
make metrics-open
```

VictoriaMetrics: <http://127.0.0.1:8428/vmui/>, цели сбора: <http://127.0.0.1:8428/targets>.
Backend отдаёт Go, HTTP и бизнес-метрики на внутреннем `:9090/metrics`; postgres-exporter — на `:9187/metrics`.
VictoriaMetrics находит каждый Pod по аннотациям, хранит данные на PVC и опрашивает цели каждые 15 секунд. Отдельный vmagent не используется.

```bash
make metrics-status
make metrics-targets
make metrics-query METRICS_QUERY='go_goroutines{job="badgoodtune-backend"}'
make metrics-query METRICS_QUERY='pg_up{job="postgres"}'
make metrics-query METRICS_QUERY='sum(rate(badgoodtune_reviews_created_total[5m]))'
```

`metrics-generate` сверяет значения до и после CRUD-сценария; запускать без параллельного трафика и перезапуска backend.
`k8s-backend-reload` и `k8s-frontend-reload` позволяют обновлять компоненты независимо.
`metrics-delete` удаляет workloads метрик, сохраняя PVC и роль мониторинга PostgreSQL.
`logs-delete` удаляет только логирование. `stack-down` удаляет весь стек, включая данные всех PVC в его namespace.
