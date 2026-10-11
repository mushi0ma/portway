# Portway

Go-бэкенд мобильного пульта для агентов разработки. Целевая архитектура с 11.10 — **микросервисы и Kafka**, по [решению владельца D-22](docs/21-microservices-kafka-amendment.md); текущая реализация включает control API, отдельные outbox relay и audit consumer; execution ещё предстоит выделить. Реализован первый срез **foundation + identity + agent configuration** по проектным документам в [docs](docs/README.md). Это начало приложения, а не готовый MVP. [Первый срез Kafka и его ограничения](docs/22-kafka-first-slice.md).

## Что работает

- Первый Go-сервис (`control-api`, разделение ещё впереди): конфигурация, JSON HTTP API, корректное завершение процесса, безопасные структурные логи.
- PostgreSQL: четыре транзакционные миграции с checksum и advisory lock; отдельные права API, мигратора и workers; закрытые схемы `portway` и `portway_audit`.
- Supabase Auth adapter: регистрация единственного владельца с setup proof, email callback, login/refresh/logout; JWT RS256/ES256 через JWKS, legacy HS256 через Auth API; привязка subject + auth session + device proof, серверный отзыв устройств.
- Каталог 5 основных ролей и 3 субагентских пресетов, создание агента, неизменяемые версии, проверка `If-Match`, ограничения прав и численные defaults SRS.
- Конфигурация allowlisted провайдеров, write-only секреты AES-256-GCM, транзакционные idempotency keys и аудит; чтение статуса по операциям.
- OpenAPI 3.1 в [api/openapi.json](api/openapi.json), тесты с race detector, изолированный PostgreSQL в Docker и GitHub Actions workflow.

**Пока не реализованы:** Runtime Manager и контейнеры выполнения агентов, LLM Harness/capability probes, run/chain, SSE, бюджеты исполнения, approvals, links/hooks и события исполнения, MCP, FCM, пакеты и Android. Создание конфигурации не запускает модель. `unresolved` явно показывает недостающие привязки. Новые S/C-расширения из backlog не добавлялись.

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

Текущий стек с реальной Supabase доступен по [HTTPS через Tailscale Funnel](https://cachyos-x8664.tail1443b8.ts.net/healthz); origin — `http://127.0.0.1:8080`. `PORTWAY_PUBLIC_URL` применён к API. Для постоянной публикации на уже авторизованном хосте используется `tailscale funnel --bg http://127.0.0.1:8080`; Funnel работает на хосте, Go-сервисы и Kafka — в Docker. Site URL, callback `/auth/confirm` и шаблон письма в Supabase Dashboard настроить по [инструкции](deploy/supabase/README.md).

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

Push token registration, отзыв через SSE и общий purge основных таблиц ещё не реализованы; outbox и inbox/projection имеют отдельную очистку по retention. Просроченный request key удаляется при повторном использовании того же ключа; общий cleanup добавляется с событиями. Аудит поддерживает курсор по 100 строк, пока без run/chain-фильтров. Access logs не содержат request bodies, query strings, JWT, device/setup proof и raw upstream errors. За TLS-туннелем применяется общий лимит 30 auth запросов в минуту; `X-Forwarded-For` не считается доверенным.

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
