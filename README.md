# Portway

**Agent Bridge v2 (14.10.2026):** телефон отправляет задачи **Codex CLI или Claude Code** на ноутбуке, выбирает модель и уровень рассуждений, видит лимиты аккаунта (5 часов / неделя) с явным состоянием «лимит исчерпан» и подключает/переключает аккаунты ChatGPT и Claude через OAuth самих сервисов (официальные CLI на ноутбуке, код устройства или вставка кода). Маршруты `/v1/codex/*`, OpenAPI 0.8.0, миграция 010; развёрнуто на рабочем стеке, `check-live` PASS. [Контракт](docs/36-api-events-and-messages.md#agent-bridge-v2), [запуск моста](docs/52-build-deploy-and-operations.md#codex-demo-operations), [протокол](docs/evidence/agent-bridge-2026-10-13.txt). Клиент 0.2.0 с редизайном — [ниже](#android-клиент).


Go-бэкенд мобильного пульта для агентов разработки. Целевая архитектура с 11.10 — **микросервисы и Kafka**, по [решению владельца D-22](docs/21-microservices-kafka-amendment.md); текущая реализация включает control API, отдельные outbox relay и audit consumer; добавлен отдельный execution service для состояния и бюджетов, production run admission пока закрыт. Реализован первый срез **foundation + identity + agent configuration** по проектным документам в [docs](docs/README.md). Это начало приложения, а не готовый MVP. [Execution foundation](docs/23-execution-foundation.md), [Funnel в Docker](deploy/tailscale/README.md). [Первый срез Kafka и его ограничения](docs/22-kafka-first-slice.md).

## Что работает

- Первый Go-сервис (`control-api`, разделение ещё впереди): конфигурация, JSON HTTP API, корректное завершение процесса, безопасные структурные логи.
- PostgreSQL: восемь транзакционных миграций (008 пока локально) с checksum и advisory lock; отдельные права API, мигратора и workers; закрытые схемы `portway`, `portway_audit` и `portway_execution`.
- Supabase Auth adapter: регистрация единственного владельца с setup proof, email callback, login/refresh/logout; JWT RS256/ES256 через JWKS, legacy HS256 через Auth API; привязка subject + auth session + device proof, серверный отзыв устройств.
- Каталог 5 основных ролей и 3 субагентских пресетов, создание агента, неизменяемые версии, проверка `If-Match`, ограничения прав и численные defaults SRS.
- Конфигурация allowlisted провайдеров, write-only секреты AES-256-GCM, транзакционные idempotency keys и аудит; чтение статуса по операциям. Добавлен защищённый capability-check: bounded protocol probe, отчёт с config binding/TTL и invalidation, replay без повторной генерации. [Контракт](docs/27-model-gateway.md#capability-check).
- Отдельный execution service: run/chain snapshots, reservations/settlement, stop epochs, startup reconciliation, собственные history/outbox и Kafka topic; HTTP admission остаётся закрытым до Runtime/LLM.
- OpenAPI 3.1 в [api/openapi.json](api/openapi.json), тесты с race detector, изолированный PostgreSQL в Docker и GitHub Actions workflow.

**Пока не реализованы:** Runtime Manager и контейнеры выполнения агентов, production dispatch LLM Gateway и verified tokenizer, реальный dispatch run/chain, links/hooks, MCP, FCM, пакеты и Android. SSE, approvals и внутренний Harness с durable results уже добавлены; Harness пока работает только с trusted адаптерами в тестах. Создание конфигурации не запускает модель. `unresolved` явно показывает недостающие привязки. Подмножество BL-022 проверяет подключение в составе обязательного capability lifecycle; каталог моделей остаётся в S-остатке.

## Проверить без Supabase и платных сервисов

Требуются Docker Engine с Compose и `make`:

```sh
make docker-test
make docker-smoke
```

Первый target собирает Go-контейнер и выполняет `gofmt`, `go vet`, unit/contract/integration с `-race`, затем сборку сервера. Внешний Auth заменяет `httptest`-сервер **только в тестах**; PostgreSQL настоящий. Прогон проверяет регистрацию/confirmation/login/refresh/revoke, запрет чужой сессии, шифрование, ограничения DB-role, восемь одновременных повторов создания агента и гонку версий. Тесты без БД не подменяют интеграционную проверку: обязательный target завершится ошибкой при отсутствии DSN.

`docker-smoke` дополнительно запускает реальный финальный образ с временной БД: миграции, HTTP, auth guard, перезапуск. Ему нужны `curl`, `openssl`, Python 3. Все его данные временные. Тестовые стеки имеют уникальные имена; после прогона контейнеры, сети и БД удаляются. Первый прогон скачивает образы/Go-модули. Во время тестов сеть Compose изолирована от внешних сервисов.

Go в Docker: **1.27.2**, образ закреплён digest; PostgreSQL тестов — 17, закреплён digest. `go.mod`/`go.sum` фиксируют зависимости. На хосте достаточно Go ≥1.27 для `make test-unit test-contract lint build-server`. [Официальные Go downloads](https://go.dev/dl/) и [Supabase JWT](https://supabase.com/docs/guides/auth/jwts) использовались при выборе toolchain и Auth adapter.

## Запуск с Supabase Free

Подготовлен `.env`-workflow по запросу владельца. Основной Compose использует внешнюю Supabase PostgreSQL, а не тестовую локальную БД. Kafka и Go-сервисы будут запускаться отдельно на ПК/сервере; тариф Supabase не предоставляет вычисления для Kafka.

```sh
make init-env
```

Создаёт `.env` с правами `0600`, случайным `PORTWAY_INSTALLATION_ID`, `PORTWAY_MASTER_KEY` (AES-256) и `PORTWAY_SETUP_TOKEN`. Заполненный файл не перезаписывается. `.env` исключён из Git и Docker build context. Docker Compose автоматически читает его; `go run` сам `.env` не загружает.

Заполнить локально, не присылая значения в чат:

| Переменная | Где взять / что означает |
|---|---|
| `PORTWAY_AUTH_URL` | Project URL + `/auth/v1`, например `https://PROJECT-REF.supabase.co/auth/v1` |
| `PORTWAY_AUTH_PUBLISHABLE_KEY` | Settings → API Keys → Publishable key (`sb_publishable_…`) |
| `PORTWAY_MIGRATION_DATABASE_URL` | Connect → Session pooler → URI, пользователь `postgres.PROJECT-REF`, **пароль БД** проекта |
| `PORTWAY_DATABASE_URL` | Тот же pooler host/database, пользователь `portway_app.PROJECT-REF`, отдельный пароль этой роли |
| `PORTWAY_PUBLIC_URL` | Настоящий HTTPS URL туннеля для email callback/Android. `https://localhost` в шаблоне — только временное значение до публикации |

**Supabase `sb_secret_…`, legacy `service_role` и JWT signing secret текущему коду не нужны.** Publishable key не является паролем PostgreSQL. Локальные master key/setup token — секреты Portway, они уже генерируются отдельно.

Для Free/IPv4 используем **Session pooler, порт 5432**. Host копировать из Connect целиком: индекс pooler нельзя вывести из региона Mumbai. Формат DSN:

```text
postgres://postgres.PROJECT-REF:ENCODED-DB-PASSWORD@EXACT-POOLER-HOST:5432/postgres?sslmode=verify-full
postgres://portway_app.PROJECT-REF:ENCODED-APP-PASSWORD@EXACT-POOLER-HOST:5432/postgres?sslmode=verify-full
```

Пароли в URI должны быть percent-encoded (`@` → `%40`, `#` → `%23`, `%` → `%25`, и т.д.). Не используйте публичный online encoder для настоящих паролей. Целую строку в `.env` можно заключить в одинарные кавычки, чтобы Compose не интерпретировал `$`.

App role создаётся оператором в SQL Editor Supabase или через psql; `python3 scripts/provision-db.py` также умеет создать отсутствующие app/relay/audit роли, сохранив отдельные worker credentials в `.env`. Выбрать **свой отдельный случайный пароль** и выполнить один раз:

```sql
CREATE ROLE portway_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS;
-- Через psql: \password portway_app
-- Либо локально в SQL Editor: ALTER ROLE portway_app PASSWORD 'YOUR-NEW-APP-PASSWORD';
```

Если роли ещё нет, сначала можно заполнить Auth URL/publishable key/migration DSN; API не запускается до создания роли и заполнения app DSN. Мигратор выдаёт `portway_app` нужные grants. Роль API не должна владеть схемой, иметь `CREATE`, superuser или BYPASSRLS; это проверяется при старте. Схема `portway` не добавляется в exposed schemas Data API.

TLS verification обязательно. Если endpoint не проходит проверку системными CA, скачать root certificate из Database → Settings → SSL в `deploy/certs/supabase-ca.crt` и дописать в соответствующую строку `&sslrootcert=/etc/portway/certs/supabase-ca.crt`. Compose монтирует каталог сертификатов read-only; `sslmode=verify-full` сохраняется. При работе с shared pooler может быть достаточно системного CA bundle. [Документация подключений Supabase](https://supabase.com/docs/guides/database/connecting-to-postgres).

Пулы relay/audit — по **1 соединению**. Пул API по умолчанию **2 соединения**, `min=0`, idle timeout 2 минуты; `PORTWAY_DATABASE_MAX_CONNS` допускает 1–8. Мигратор всегда использует максимум 1 соединение. Это начальный профиль, а не обещание производительности бесплатного тарифа. С появлением сервисов общий бюджет соединений контролируется вместе.

Включить email confirmation в Supabase. Настроить Site URL и точный redirect URL `PORTWAY_PUBLIC_URL/auth/confirm`, затем вставить [готовый шаблон](deploy/supabase/confirm-signup.html) в Confirm signup. Signup передаёт настроенный callback через `redirect_to`:

```html
<a href="{{ .RedirectTo }}?token_hash={{ .TokenHash }}&amp;type=email">Confirm email</a>
```

Callback принимает только `type=email` и `token_hash`, затем перенаправляет на `portway://auth/confirmed` без токенов. После подтверждения нужен login. Настроить соответствующий URL в Auth settings; реальная отправка email/SMTP и callback проверяются после подключения проекта.

```sh
python3 scripts/provision-db.py
docker compose up --build -d
make smoke
make check-live
```

Compose сначала применяет миграции, затем запускает API. Он получает только app DSN/master/setup, migration DSN передаётся только мигратору. Секреты из `.env` видны оператору с доступом к Docker inspect: не публикуйте полный `docker compose config`, используйте `docker compose config --quiet`. Альтернативные `<NAME>_FILE` сохраняются в Go-конфигурации для развёртывания через secret-file mounts; одновременно value и `_FILE` запрещены. Основной Compose сейчас рассчитан на `.env`.

`make check-live` выполняет read-only проверки: TLS и отдельные DB-роли, checksum миграций, закрытые схемы для `anon/authenticated`, Supabase Auth, API локально/через туннель и готовность workers. Значения DSN/ключей и тела ответов не печатаются; регистрация и отправка email не выполняются. Для проверки только DB/Auth/workers — `python3 scripts/check-live.py --private-only`.

Порт доступен только на `127.0.0.1:${PORTWAY_PORT}` (8080 в текущей установке; порт 8000 занят другим проектом); туннель публикует API по HTTPS. API без Docker socket, финальный образ `scratch`, non-root/read-only, capabilities сняты. Для обновления существующей схемы — `make migrate`, затем пересоздать API на новой сборке. При изменении окружения не перегенерировать master key: процедура ротации ещё не реализована.

После запуска публичный `GET /v1/client-config` показывает `setup_required`. Для регистрации использовать `POST /v1/auth/register`, заголовки `X-Setup-Proof`, `Idempotency-Key`, JSON `name/email/password`. Setup proof расходуется после подтверждения владельца. Login выдаёт `session` и `device_proof`; защищённые запросы используют одновременно `Authorization: Bearer …` и `X-Device-Session: …`.

Текущий стек с реальной Supabase доступен по [HTTPS через Docker Funnel](https://portway.tail1443b8.ts.net/healthz). Отдельный Tailscale-узел `portway` проксирует `http://api:8080`; `PORTWAY_PUBLIC_URL` применён к API. Node ID и домен сохраняются после перезапуска благодаря persistent volume. `COMPOSE_PROFILES=execution,funnel` включает оба сервиса при обычном `docker compose up -d`. [Инструкция Funnel](deploy/tailscale/README.md); Site URL, callback `/auth/confirm` и шаблон письма в Supabase Dashboard — по [инструкции Auth](deploy/supabase/README.md).

`GET /healthz` — процесс жив. Docker healthcheck использует его. **`GET /readyz` сейчас всегда 503**, поскольку запуск агентов ещё не реализован. Защищённый `GET /v1/status` отдельно сообщает `operations.configuration` и `operations.run=false`; неизвестные зависимости не объявляются готовыми.

## API и ограничения первого среза

Контракт содержит только реализованные маршруты. POST создания, новая версия, обновление provider и отзыв другого device принимают `Idempotency-Key` длиной 8–200 символов. Повтор канонически того же запроса возвращает тот же ресурс; другой payload с тем же ключом — 409. Новая версия требует `base_version_id` и `If-Match` с текущим integer version/ETag. Неизвестные и повторяющиеся JSON-поля отклоняются, максимальный JSON body — 1 MiB.

Минимальная конфигурация агента:

```json
{
  "name": "Coder",
  "config": {
    "instructions": "Implement the requested change and test it.",
    "role_code": "coder",
    "role_catalog_version": 1,
    "tools": [{"alias":"read","kind":"workspace.read"}]
  }
}
```

Если `limits` не передан, применяются 900 секунд / 20 шагов / 20 000 токенов / 2048 output. Полностью заданные лимиты можно только уменьшать. Право `shell.exec` не может стать `allow`; роль reviewer/code_reader не получает write. Навыки в локальном DTO представлены `{alias, content}`, MCP — `{alias, adapter_kind, endpoint_alias, secret_ref?}`. Это декларации; ни навыки, ни инструменты не исполняются этим срезом.

Endpoint провайдера должен точно совпасть с одной из записей `PORTWAY_LLM_ALLOWED_ENDPOINTS` (через запятую). Настройка не делает сетевого probe и не подтверждает capabilities. Записанный secret возвращает только `id/alias/state`; шифротекст также не выдаётся API. Provider и MCP ссылки проверяются внутри installation.

Если исход внешнего signup неизвестен, intent остаётся зарезервированным, повтор не создаёт другой аккаунт; API возвращает `REGISTRATION_UNCERTAIN`. Автоматического освобождения claim и operator-reconcile CLI в этом срезе нет. Реальная проверка Supabase email/password policy, callbacks и key rotation ещё необходима. JWT/device тесты не заменяют её.

SSE запуска доступен на `GET /v1/runs/{id}/events`: replay по `Last-Event-ID: <run UUID>:<seq>` либо `?cursor=…`, heartbeat, повторная проверка JWT/device, явный 410 при истёкшей истории. [Контракт и ограничения](docs/24-execution-sse-retention.md). Execution очищает опубликованную историю и истёкшие request keys ограниченными batch-транзакциями; pending outbox сохраняется. Push token registration и общий purge таблиц control API ещё не реализованы; audit outbox/inbox/projection имеют отдельную очистку по retention. Просроченный control request key удаляется при повторном использовании того же ключа. Аудит поддерживает курсор по 100 строк, пока без run/chain-фильтров. Access logs не содержат request bodies, query strings, JWT, device/setup proof и raw upstream errors. За TLS-туннелем применяется общий лимит 30 auth запросов в минуту; `X-Forwarded-For` не считается доверенным.

Согласование действий: `GET /v1/approvals`, `GET /v1/approvals/{id}`, `POST /v1/approvals/{id}/decision`. Решение содержит `approve`/`reject`, ожидаемую версию и hash неизменяемых аргументов. Pending живёт 180 секунд; stop/restart отменяет допуск. Approved хранится отдельно от исхода команды; реальное исполнение ещё не подключено. [Описание сценария](docs/25-approval-workflow.md).

## Устройство проекта

```text
cmd/portway/             api, migrate, check, probe, version
internal/config/        env и secret files
internal/identity/      Supabase, JWT/JWKS, owner/device lifecycle
internal/agents/        каталог ролей, policy validation, immutable versions
internal/storage/       pgx, миграции, транзакции, idempotency, audit
internal/secure/        UUID/proofs, hashes, AES-GCM
internal/transport/     HTTP adapters, DTO, strict JSON, error envelope
api/openapi.json         исполняемый контракт реализованного API
tests/integration/      PostgreSQL + HTTP/Auth fixtures
compose.yaml            рабочий API + одноразовый мигратор
compose.test.yaml       disposable PostgreSQL + контейнер тестов
```

[Статус реализации и следующие срезы](docs/implementation-status.md) отдельно от проектной приёмки. Workflow подготовлен; удалённый GitHub CI не считается выполненным до публикации репозитория и реального прогона.

### Harness и результат запуска

Внутренний цикл и защищённый `GET /v1/runs/{id}/result` описаны в [срезе Harness](docs/26-harness-turns-and-results.md). Reservation предшествует модельному вызову, tool outcome сохраняется до следующего шага, final answer и completion event атомарны. Production admission остаётся закрытым до реальных Gateway/Runtime и dispatch fencing.

Рабочая очередь и критерии завершения — [implementation plan](docs/implementation-plan.md). IP-01: [потоковый Chat Completions adapter](docs/27-model-gateway.md) проверен на HTTP fixtures; IP-02a: capability-check/report lifecycle проверен локально; IP-02b: [free-only OpenRouter/private Gateway](docs/28-free-model-connections.md) добавлен и проверен локально (Docker/race/PG/Kafka/smoke). Следующие пункты — verified token counting и Runtime/worker. Реальный free-model roundtrip пока не принят.

## Бесплатные модели

`OPENROUTER_API_KEY` в `.env` передаётся только control API. `PORTWAY_MODEL_POLICY=free_only`; платные модели/fallback/plugins запрещены. После сборки `portway models` читает текущий zero-price каталог без генерации, `portway model-probe MODEL_ID:free` запускает ограниченную диагностику. [Подключение/API/private Gateway и текущие ограничения](docs/28-free-model-connections.md). Создание реальных agent runs ещё закрыто до verified tokenizer и Runtime; fixture counter не включается в production.

## Android-клиент

Flutter-клиент 0.2.0 в [apps/portway](apps/portway/README.md): Splash, подключение установки, регистрация/вход с проверкой полей, главная с лимитами ИИ, чат-композер задач (Codex / Claude Code, модель, effort), ответ в Markdown, история с поиском и фильтрами, аккаунты ИИ, профиль и выход; Dark/Light по переработанной системе Figma ([DESIGN.md](DESIGN.md)). APK: `output/android/portway-demo-0.2.0-3.apk`; показ — `python3 scripts/demo-android.py`. [Протокол проверки](docs/evidence/flutter-redesign-2026-10-13.md): analyze/test PASS, эмулятор; физический телефон и TalkBack — NOT_RUN.
