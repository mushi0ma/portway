# Portway — [API][g-api], [события][g-event] и [сообщения][g-message]

> **Изменение 11.10.2026 (D-22):** по решению владельца целевая архитектура — Go-микросервисы и Kafka. [Принятое изменение архитектуры](21-microservices-kafka-amendment.md) заменяет прежний монолит/DB-only шину; PostgreSQL outbox сохраняется для надёжной публикации. Старые схемы одного процесса ниже — baseline до изменения, не текущая целевая топология. Первый реализованный срез Kafka описан в [DOC-022](22-kafka-first-slice.md); полный переход ещё не завершён.


> **Фактический срез 12.10:** [статус](implementation-status.md), [план](implementation-plan.md), [Harness/Result](26-harness-turns-and-results.md). Проектные структуры далее задают целевой контракт; реализованное подмножество описано в коде/миграциях и [OpenAPI JSON](../api/openapi.json). NOT_RUN baseline не отрицает выполненные инженерные проверки.

<a id="doc-036"></a>

| Поле | Значение |
|---|---|
| Артефакт | [DOC-036][id-doc-036] |
| Версия и дата | v0.3 · 14.10.2026 |
| Статус | Проектный контракт v1 полного MVP; реализованное подмножество и проверки — [статус кода](implementation-status.md) |
| Основания | [SRS](03-software-requirements-specification.md), [данные/состояния](29-data-and-state-model.md), [исполнение](40-agent-execution-and-security.md) |
| Потребители | [Android](61-android-screens-flows-and-api.md), [пакет](38-agent-package-format.md), [проверки](57-verification-and-traceability.md), [эксплуатация](52-build-deploy-and-operations.md) |

<a id="conventions"></a>
<a id="codex-demo"></a>
## Native Codex: контракт демонстрационного профиля, 13.10.2026

По поручению владельца ближайший вертикальный сценарий — телефон → Portway → установленный Codex CLI на ноутбуке → ответ. Профиль `codex` отделён от общего `/v1/runs`; его API не требует выбранного OpenRouter provider. Используется текущая авторизация CLI через ChatGPT и её квота; платные model API keys не используются. Каждый запрос создаёт независимый ephemeral CLI-сеанс, а не отправляет сообщение в существующий чат Codex Desktop. Продолжение разговоров и редактирование файлов пока не входят в профиль.

Все `/v1/codex/*` требуют существующие `Authorization: Bearer <access JWT>` и `X-Device-Session: <proof>`. DTO и envelope — [OpenAPI](../api/openapi.json), версия 0.7.0. Клиент не получает bridge key, CLI auth, локальный путь, PID или флаги процесса.

| Метод | Вход | `data` успешного ответа |
|---|---|---|
| `GET /v1/codex/status` | — | `connected`, `workspace="portway"`, `sandbox="read-only"`, `last_seen_at` (UTC/null) |
| `POST /v1/codex/runs` | `{"workspace":"portway","prompt":"..."}`; `Idempotency-Key` | 202: `id`, `state="queued"`, `workspace`, `sandbox` |
| `GET /v1/codex/runs` | — | `items`, последние 50 за 7 суток; краткие записи без prompt/answer |
| `GET /v1/codex/runs/{id}` | UUID | `id`, `state`, `workspace`, `sandbox`, `prompt`, опциональный `answer`, `error_code`, `created_at`, nullable `started_at`/`finished_at` |
| `GET /v1/codex/runs/{id}/events?after=N` | integer 0–256, default 0 | `items` (≤100), `next_after`; event=`sequence,type,text?` |
| `POST /v1/codex/runs/{id}/stop` | `{}`; `Idempotency-Key` | `id,state`; terminal не меняется, активный становится `stopped` |

Клиент опрашивает события и snapshot раз в 2 секунды, пока `queued/running`; сохраняет `next_after` и дедуплицирует по `sequence`. На reconnect начинает с последнего курсора; пустой ответ сохраняет курсор. Это JSON polling, не SSE и не token delta. `started` означает запуск процесса; `progress` — законченный agent message или безопасная отметка read-only команды; `completed` содержит конечный ответ; `failed` не содержит stderr. Snapshot — источник terminal state, включая `stopped` или потерю моста без нового события. Таймеры прекращаются при уходе приложения в background/с экрана; foreground восстанавливает snapshot и курсор.

Prompt: непустой UTF-8, ≤16384 байт, без NUL; workspace только `portway`, разрешения фиксированы `read-only`. Один активный запрос на установку. Ответ ≤65536 байт; encoded JSON одного события ≤69984 байт (слишком экранированный ответ завершает запрос с ошибкой); ≤256 событий. Повтор POST с тем же ключом возвращает ту же admission запись; новый ключ при активном запросе →409 `CODEX_BUSY`. Offline bridge →503 `CODEX_OFFLINE`; отключённый профиль →503 `CODEX_DISABLED`. `queued` → `running` → `completed|failed|stopped`; очередь ждёт ≤60 секунд, процесс ≤9 минут, весь запрос ≤10 минут. Потеря 30-секундного lease →`failed/CODEX_BRIDGE_LOST`, без повторного dispatch. Stop фиксируется сразу; локальный процесс отменяется на следующем heartbeat (2 секунды), либо при ошибке связи/таймауте heartbeat. Для ненулевого времени доставки нельзя обещать мгновенное физическое завершение процесса.

Внутренние `POST /internal/v1/codex/poll` и `/report` используют отдельный `X-Portway-Bridge-Key`. Мост сам устанавливает исходящее HTTPS/loopback соединение; на ноутбуке нет публичного listener. Poll получает одноразовый lease, повтор idle poll не выдаёт уже начатую задачу. Report содержит `run_id,lease_token,event`; повтор одинакового sequence/hash безопасен, другой payload того же sequence →409, пропуск sequence →409. Поздний новый report после stop/revocation/lease expiry отвергается. Generic run admission, бюджеты Harness, approvals, reviewer, FCM и контейнерный Runtime остаются отдельным обязательным объёмом.


<a id="agent-bridge-v2"></a>
### Agent Bridge v2: Codex CLI и Claude Code, 14.10.2026

Тот же opt-in мост на ноутбуке обслуживает два провайдера: `codex` (Codex CLI, аккаунт ChatGPT) и `claude` (Claude Code, подписка Claude). Телефон выбирает провайдера, модель и reasoning effort, видит лимиты (включая явное состояние `exhausted`) и подключает/переключает AI-аккаунты через собственные OAuth-потоки официальных CLI. Прежние маршруты и поля сохранены; мост первого среза продолжает работать только с Codex. [OpenAPI](../api/openapi.json) 0.8.0, миграция `010_agent_bridge.sql`.

| Метод | Вход | Ответ / ошибки |
|---|---|---|
| `GET /v1/codex/status` | — | прежние поля + `state_updated_at`, `providers[]` (id, name, installed, cli_version, active_account_id, accounts[], models[], usage), `login` (последний вход ≤15 мин или null), `commands[]` (≤10 за 10 мин). До первого snapshot: `providers=[]`, `login=null`, `commands=[]`; при `connected=false` — последний известный snapshot |
| `POST /v1/codex/runs` | + `provider` (`codex` по умолчанию), `model` (1–100, `^[A-Za-z0-9._:/\[\]-]+$`, null/`default` = модель провайдера), `effort` (`low…max`) | 202 Run DTO; `503 CODEX_OFFLINE`, `409 CODEX_BUSY`, `409 AI_PROVIDER_UNAVAILABLE`, `409 AI_ACCOUNT_REQUIRED`, `429 AI_LIMIT_EXHAUSTED` + `details{provider,reset_at}`, `400 INVALID_REQUEST` (провайдер, формат/неизвестная модель, effort не поддержан моделью Codex) |
| `GET /v1/codex/runs?provider=&state=` | необязательные фильтры | Run DTO с `provider, model, effort, resolved_model, account_label, tokens{input,output}, prompt_preview` (все новые поля nullable) |
| `POST /v1/codex/accounts/login` | `{"provider","method":"phone"\|"computer"}` | 202 `{command_id, login_id}`; `409 AI_LOGIN_ACTIVE` + `details{login_id}` |
| `POST /v1/codex/accounts/login/{login_id}/code` | `{"code"}` 1–2048 без пробелов внутри | 202 `{command_id}`; `404`, `409 AI_LOGIN_NOT_ACTIVE` |
| `POST /v1/codex/accounts/login/{login_id}/cancel` | `{}` | 202 `{command_id}` |
| `POST /v1/codex/accounts/{account_id}/activate` | `{"provider"}` | 202 `{command_id}`; `404` для аккаунта вне snapshot |
| `DELETE /v1/codex/accounts/{account_id}?provider=` | — | 202 `{command_id}`; `409 AI_ACCOUNT_SYSTEM` для системного входа ноутбука |
| `POST /v1/codex/usage/refresh` | `{}` | 202 `{command_id}` |

Все mutation требуют `Idempotency-Key`; команды при offline-мосте → `503 CODEX_OFFLINE`, при 10 незавершённых → `429 AI_COMMANDS_BUSY`. Результат команды клиент видит в `GET /v1/codex/status` (`commands[].state`: `queued → delivered → done|failed`; `COMMAND_EXPIRED`, если мост не забрал команду за 60 с; `CODEX_BRIDGE_LOST`, если нет ответа за 2 мин). Вход: `starting → waiting_user → verifying → completed|failed|cancelled|expired`; `phone` — device code ChatGPT (`url`+`user_code`) или вставка кода Claude (`url`, `needs_code=true`); `computer` — браузер открывается на ноутбуке. API отдаёт `url` только HTTPS-хостов {auth.openai.com, chatgpt.com, claude.ai, claude.com, platform.claude.com, console.anthropic.com}, иначе `null`. Успешный вход всегда создаёт новый изолированный профиль и делает его активным; системный вход (`codex-system`/`claude-system`) API не удаляет.

`usage.status`: `ok`, `warning` (любое окно ≥80 %), `exhausted` (окно ≥100 %, `ordinaryUsageAllowed=false`/`rateLimitReachedType` Codex или `limits[].percent≥100` Claude), `unknown`; `reset_at` — только для `exhausted`. Окна: `five_hour`, `weekly`, `weekly_opus`, `weekly_sonnet` (иное — по `window_minutes`). Лимиты читаются без inference: Codex `account/rateLimits/read` через `codex app-server`, Claude — OAuth usage/profile endpoint с токеном профиля, не чаще 1 раза в минуту на провайдера (принудительно — после команд, запусков и входа).

Ошибки запуска (`error_code` и `code` в событии `failed`, без текста CLI): `AI_LIMIT_EXHAUSTED`, `AI_AUTH_REQUIRED`, `AI_RUN_FAILED`, `AI_TIMEOUT`, `CODEX_BRIDGE_LOST`, `DEVICE_REVOKED`; исторические записи могут содержать `CODEX_EXEC_FAILED`. Форма событий прежняя; метаданные завершения (`resolved_model`, `account_label`, `tokens`) мост передаёт в `completed.meta`, API хранит их в колонках запуска и не повторяет в событиях.

Внутренний контракт моста: poll дополнительно отдаёт `commands[]` (`login_start`, `login_code`, `login_cancel`, `account_activate`, `account_remove`, `usage_refresh`) ровно один раз, после выдачи payload (включая код входа) стирается; Job получил `provider/model/effort`. `POST /internal/v1/codex/state` принимает snapshot ≤64 KiB со строгой проверкой enum/длин/идентификаторов (небезопасные URL отбрасываются) и хранит его AES-GCM ciphertext (в нём email). `POST /internal/v1/codex/commands/{id}/ack` `{state:"done"|"failed", error_code}` идемпотентен для одинакового исхода. Заявки на команды аудируются действием `codex.command_queued` (`codex_command`) без payload.

## 1. Общие правила HTTP

Base URL задаёт [владелец][g-owner]; [API][g-api] — `/v1`, только [HTTPS][g-https] на публичном маршруте. JSON — [UTF-8][g-identifier-formats], время [RFC 3339][g-identifier-formats] [UTC][g-identifier-formats], ID — [UUID][g-identifier-formats], размеры в байтах. Неизвестные поля mutation DTO отклоняются. Машинные имена в примерах являются контрактом; текстовые подписи [UI][g-ui] в него не входят. Реализованное подмножество [OpenAPI][g-api] 3.1 JSON хранится в `api/openapi.json`, синхронизируется с каждым срезом и проверяется contract test в [CI][g-ci-cd]; целевые таблицы ниже не означают наличие всего MVP в коде.

Защищённый запрос содержит `Authorization: Bearer <Supabase access JWT>` и `X-Device-Session: <opaque proof>`. Сервер проверяет signature/issuer/audience/expiry/subject, installation [owner][g-owner] и активную [device session][g-device-session] в БД. Refresh/logout имеют отдельные правила ниже. Чужой валидный [Supabase][g-supabase] [JWT][g-jwt-jwks] не разрешает доступ. Случайный [device proof][g-device-session] хранится в [APK][g-apk] защищённо, на сервере — hash; [секреты][g-secret] не передаются в query string. Подробности [auth][g-auth]/отзыва — [DOC-040][id-doc-040].

Mutation, создающая логический ресурс/решение, принимает `Idempotency-Key`; scope = installation+[owner][g-owner]+operation. Один ключ и канонически тот же body возвращают тот же ресурс/результат; другой body →409 IDEMPOTENCY_CONFLICT. Окно 7 суток. Неизвестный ответ на POST повторяется с исходным ключом, а не с новым. Значение ключа нельзя использовать как доказательство прав. [Auth][g-auth] login/refresh и confirmation callback не кешируют credential-ответы в `request_keys`: потерянный login/refresh завершается свежей проверкой сессии либо новым входом; logout идемпотентен по [device session][g-device-session]. Registration intent использует request key, но сохраняет только безопасный `registration_id`/status, без пароля/токенов. PATCH изменяемой конфигурации принимает `If-Match: <version>`; stale version →409 VERSION_CONFLICT. Создание новой [AgentVersion][g-agent-version] оставляет предыдущую неизменной.

Обычный ответ содержит `data`, `meta.request_id`, `meta.server_time`; коллекция — `data.items` и `next_cursor` только там, где он поддержан. Basic list не обещает расширенный поиск/архив [FR-010][id-fr-010]/028. [Stop][g-cancelled] возвращает 202 с `state=stopping`; terminal подтверждается последующим snapshot/[event][g-event]. Повтор [stop][g-cancelled] terminal-объекта возвращает его текущий исход. Ошибки не раскрывают stack trace или секретные значения.

```json
{
  "error": {
    "code": "APPROVAL_EXPIRED",
    "message": "Срок подтверждения истёк",
    "details": {"state": "expired", "expires_at": "2026-10-10T12:03:00Z"}
  },
  "meta": {"request_id": "req-example", "server_time": "2026-10-10T12:03:01Z"}
}
```

| HTTP | Стабильные коды / поведение |
|---|---|
| 400 | INVALID_REQUEST, INVALID_CURSOR; синтаксис/неверный формат |
| 401 | AUTH_REQUIRED, TOKEN_EXPIRED, DEVICE_REVOKED; обновить доступ или войти |
| 403 | OWNER_REQUIRED, SETUP_DENIED, POLICY_DENIED; повтор без изменения условий бессмысленен |
| 404 | NOT_FOUND; ресурс другой installation также не раскрывается |
| 409 | IDEMPOTENCY_CONFLICT, VERSION_CONFLICT, RUN_TERMINAL, APPROVAL_RESOLVED, OWNER_ALREADY_BOUND, ROUTE_AMBIGUOUS |
| 410 | HISTORY_EXPIRED, IMPORT_EXPIRED, ARTIFACT_EXPIRED; перечитать snapshot либо повторить [staging][g-staging] |
| 413 | PACKAGE_TOO_LARGE, MESSAGE_TOO_LARGE |
| 415 | UNSUPPORTED_PACKAGE_FORMAT / неверный Content-Type |
| 422 | INVALID_PACKAGE, UNRESOLVED_MAPPING, CAPABILITY_MISSING, UNSUPPORTED_SCHEMA, INVALID_TRANSITION, APPROVAL_EXPIRED |
| 429 | RATE_LIMITED, CAPACITY_LIMIT; Retry-After при известном времени |
| 503 | DEPENDENCY_UNAVAILABLE, RECONCILING; нет фиктивного успешного результата |

<a id="auth"></a>
## 2. Доступ, регистрация и устройства

Пароль проходит только через [HTTPS][g-https] [Go][g-go] [Auth][g-auth] adapter к [Supabase][g-supabase] [Auth][g-auth], не сохраняется и не логируется Portway. Первичная регистрация защищена [setup proof][g-device-session]; [owner][g-owner] bind создаётся после email confirmation в той же installation. Confirmation callback не логинит пользователя автоматически: после подтверждения [APK][g-apk] предлагает войти. Эта схема сохраняет обязательные регистрацию/подтверждение/callback без передачи access token в deep link.

| Операция | Доступ и вход | Результат / особенность | Требования |
|---|---|---|---|
| `GET /v1/client-config` | Публично; без [секретов][g-secret] | installation_id/name, api_version, setup_required; не credentials | [FR-090][id-fr-090] |
| `POST /v1/auth/register` | [setup proof][g-device-session] + key; name/email/password | 202 pending_confirmation, registration_id; один активный intent, нет пароля в БД | [FR-081][id-fr-081] |
| `GET /auth/confirm` | Одноразовый [Supabase][g-supabase] token_hash и фиксированный type=email | Сервер verify, сверка intent/user ID/email confirmation, атомарный [owner][g-owner] bind; redirect `portway://auth/confirmed` без токенов | [FR-081][id-fr-081]; [UC-001][id-uc-001] |
| `POST /v1/auth/login` | email/password; публично с rate limit | [Supabase][g-supabase] session + новый [device proof][g-device-session], [owner][g-owner] profile; чужой subject→403 | [FR-001][id-fr-001]/065 |
| `POST /v1/auth/refresh` | refresh token + [device proof][g-device-session] | Обновлённые пользовательские токены; subject/[auth][g-auth] session соответствуют device; revoked proof запрещён | [FR-001][id-fr-001]/003 |
| `POST /v1/auth/logout` | [device proof][g-device-session]; [JWT][g-jwt-jwks] если ещё действителен | Сначала durable revoke device+[push][g-push] unlink, затем попытка [Supabase][g-supabase] signout; ответ отдельно показывает upstream outcome | [FR-003][id-fr-003]/085 |
| `GET /v1/me` | [Owner][g-owner]+device | name/email; read-only профиль базы | [FR-084][id-fr-084] |
| `GET /v1/devices` | [Owner][g-owner]+device | Свои session ID, label, last_seen, revoked; без proof/[FCM token][g-push-token] | [FR-003][id-fr-003] |
| `DELETE /v1/devices/{id}` | [Owner][g-owner]+device+key | Revoke и unlink; поток этого device закрывается ≤5 с | [FR-003][id-fr-003]/085 |
| `PUT /v1/devices/current/push-token` | [Owner][g-owner]+device; token+permission+platform | Привязка/rotation; предыдущая запись inactive | [FR-086][id-fr-086] |
| `DELETE /v1/devices/current/push-token` | [Owner][g-owner]+device | Unlink текущего устройства | [FR-085][id-fr-085]/086 |

Callback URL задаётся в конфигурации сервера и email template, пользовательский `next` не используется. При неопределённом signup ответе registration intent сначала сверяется с [Auth][g-auth], новый [owner][g-owner] не создаётся. Повтор callback возвращает безопасную страницу/переход с текущим состоянием без выдачи сессии. Offline logout — локальная операция [APK][g-apk]; без ответа [Go][g-go] нельзя показать подтверждённый серверный отзыв. [Политика][g-policy] [Auth][g-auth] password/rate limits берётся из реального Supabase-проекта и отражается в клиентской валидации до приёмки.

<a id="agents"></a>
## 3. [Агенты][g-agent], [роли][g-role] и [provider][g-provider]

| Операция | Вход / выход | Правило |
|---|---|---|
| `GET /v1/roles` | groups main/subagent; catalog_version, [tool][g-tool]/[policy][g-policy] defaults | 5+3 presets, без редактора [ролей][g-role] |
| `GET /v1/agents` | Basic list: id/name/current_version_id | Обязательный выбор; не расширенный архив |
| `POST /v1/agents` | name + config + key → [Agent][g-agent] и первая [AgentVersion][g-agent-version] | Валидация инструкций/[skills][g-skill]/[MCP][g-mcp]/[tool][g-tool] refs; [unresolved][g-staging] зависимости видны |
| `GET /v1/agents/{id}` | metadata + current version reference | Не включает secret values |
| `POST /v1/agents/{id}/versions` | config + base_version_id + If-Match + key | Новая неизменяемая версия; stale base→409 |
| `GET /v1/agents/{id}/versions` | id/ordinal/[config_hash][g-config-hash] доступных версий | Минимум для выбора/чтения M; сравнение/архив S |
| `GET /v1/agent-versions/{id}` | Нормализованная config + [unresolved][g-staging]/capability needs | Историческая версия доступна по ID |
| `GET /v1/providers` | id/label/endpoint/model_id/capabilities/checked_at/protocol_valid | Secret ref, без значения |
| `POST /v1/providers` | endpoint/model_id/secret_ref + key | Allowlisted network destination; создаёт конфигурацию |
| `PUT /v1/providers/{id}` | Те же поля + If-Match | Старый capability report становится stale |
| `POST /v1/providers/{id}/capability-check` | `profile=chat-completions-v1` + If-Match + key | 202 acknowledgement; ограниченные [stream][g-model-protocol]/[tool][g-tool]/result/[usage][g-usage] probes; отчёт привязан к config и TTL, GET providers вычисляет protocol_valid. Код проверен fixtures; реальные M1/M2 ещё нужны. Не каталог [моделей][g-model]; частичное покрытие [FR-017][id-fr-017] S |
| `GET /v1/secret-refs` | alias/[state][g-uml] | Доступно для сопоставления, без ciphertext |
| `POST /v1/secret-refs` | alias/value + key | Write-only сохранение; ответ metadata, значение ни в одном GET |

[Agent][g-agent] config: `instructions`, `role_code`, `role_catalog_version`, `skills`, `tools`, `mcp`, `provider_ref`, `limits`, `policy`. Локальные привязки DTO конкретизированы в [DOC-038][id-doc-038]; [роль][g-role] — seed, эффективное разрешение вычисляет сервер. Протокол [provider][g-provider] — нормализованные messages, delta, [tool][g-tool] proposal, finish reason, [usage][g-usage]/source, typed error; сырой vendor JSON не протекает в [Harness][g-harness] или [UI][g-ui]. Probe не является разрешением оплатить произвольный внешний сервис.

<a id="runs"></a>
## 4. [Задачи][g-task], [run][g-run], [chain][g-chain] и результат

| Операция | Вход / выход | Правило |
|---|---|---|
| `POST /v1/runs` | agent_version_id, provider_id, task, repository_ref/revision, limits, key | 201 [run][g-run]+[chain][g-chain]+snapshot; сервер создаёт correlation_id, валидирует лимиты/привязки |
| `GET /v1/runs` | Basic list текущей работы/последних известных результатов | Расширенные фильтры историй [FR-028][id-fr-028] S не обещаются |
| `GET /v1/runs/{id}` | [state][g-uml]/reason, pinned config, budgets, result_ref, snapshot_cursor | Согласованное состояние и high-water cursor одной DB snapshot |
| `GET /v1/runs/{id}/events` | [SSE][g-sse]; Last-Event-ID или cursor | Replay/live в per-run порядке; детали ниже |
| `POST /v1/runs/{id}/stop` | key | 202 stopping; новые выдачи запрещены после durable [stop][g-cancelled] |
| `GET /v1/chains/{id}` | [Run][g-run] IDs, states, общий [бюджет][g-budget], routing outcomes | id — correlation_id; terminal [chain][g-chain] не оживает |
| `POST /v1/chains/{id}/stop` | key | [Stop][g-cancelled] всех активных [run][g-run]/будущих дочерних выдач |
| `GET /v1/runs/{id}/result` | summary/revisions/diff_hash/verdict/artifact manifest | M-часть [FR-059][id-fr-059]/070; PR поля отсутствуют в базе |
| `GET /v1/results/{id}/artifacts/{artifact_id}` | Защищённый поток выбранного artifact | ID по manifest, не произвольный путь; после cleanup 410 |

```json
{
  "agent_version_id": "10000000-0000-4000-8000-000000000001",
  "provider_id": "20000000-0000-4000-8000-000000000001",
  "task": "Добавить проверку пустого входа и тест",
  "repository_ref": "demo-repo",
  "revision": "PINNED_COMMIT_SHA",
  "limits": {"run_seconds": 900, "steps": 20, "tokens": 20000}
}
```

[Repository][g-repository] ref сопоставляет зарегистрированный сервером источник с разрешённой ревизией; путь телефона/URL из prompt не становится host mount. Новый ручной [run][g-run] после сбоя получает новый ID/[chain][g-chain] и может содержать reference на старый результат для пользователя; клиент не передаёт авторитетные `hops/depth/correlation_id` в create DTO.

<a id="approvals"></a>
## 5. [Апрувы][g-approval] и ручная топология

| Операция | Вход / выход | Правило |
|---|---|---|
| `GET /v1/approvals` | Pending + связанные [run][g-run]/[tool][g-tool] и server time | Не требует полных секретных args |
| `GET /v1/approvals/{id}` | [state][g-uml], tool_name, safe meaningful args, args_hash, expires_at, version | Перечитывается после [push][g-push] и перед решением |
| `POST /v1/approvals/{id}/decision` | decision approve/reject, expected_version, args_hash, key | [CAS][g-atomicity]; текущие [owner][g-owner]/device/[stop][g-cancelled]/budget/[TTL][g-timeout]; один исход |
| `GET /v1/links` | [Направленные связи][g-link] и enabled/version | Только installation [владельца][g-owner] |
| `POST /v1/links` | from/to [agent][g-agent] IDs, allowed_types, enabled, key | Оба [агента][g-agent] доступны [владельцу][g-owner], направление явное |
| `PATCH /v1/links/{id}` | enabled/allowed_types + If-Match | Отключение блокирует ещё не начатую доставку |
| `GET /v1/hooks` | event_type/action/link/target_version/enabled | Версионированная настройка |
| `POST /v1/hooks` | link_id, event_type, action, target_version_id, key | Только каталог и разрешённая комбинация |
| `PATCH /v1/hooks/{id}` | config/enabled + If-Match | Изменение не переписывает уже выполненный эффект |

```json
{"decision":"approve","expected_version":1,"args_hash":"SHA256_OF_IMMUTABLE_ARGS"}
```

[Approval][g-approval] отвечает за одно связанное действие. Изменённые args→409; deadline истёк→422 с актуальным expired; уже противоположное решение→409; повтор того же успешного запроса возвращает сохранённое решение. [Approval][g-approval] approved всё ещё может иметь [tool][g-tool] cancelled/unknown. Внешнего публичного endpoint «исполнить [tool][g-tool] без [run][g-run]» нет.

<a id="sse"></a>
## 6. [SSE][g-sse] и каталог [событий][g-event] v1

Endpoint per-run отвечает `text/event-stream`, `Cache-Control: no-cache`, отключает proxy buffering и регулярно flush. [Flutter][g-flutter] использует HTTP-клиент с заголовками [auth][g-auth], не переносит токены в URL. `id` — `<run UUID>:<seq>`; `event` — имя из каталога; `data` — JSON ниже. Комментарии heartbeat не имеют [event][g-event] ID и не считаются бизнес-событиями.

```text
id: 30000000-0000-4000-8000-000000000001:12
event: approval_requested
data: {"schema_version":1,"id":"40000000-0000-4000-8000-000000000001","run_id":"30000000-0000-4000-8000-000000000001","correlation_id":"50000000-0000-4000-8000-000000000001","seq":12,"type":"approval_requested","server_time":"2026-10-10T12:00:00Z","payload":{"approval_id":"60000000-0000-4000-8000-000000000001","tool_name":"shell.exec","args_hash":"EXAMPLE_HASH","expires_at":"2026-10-10T12:03:00Z"}}

```

Обязательные поля [event][g-event]: schema_version=1, id, run_id, correlation_id, seq, type, server_time, payload. Payload допускает только поля соответствующей строки каталога. Payload содержит references и safe previews; не credential, не полный prompt/stdout по умолчанию. delta текст санитизируется перед журналированием; полный результат выдаётся отдельным защищённым чтением.

| [Event][g-event] type | Payload v1 / семантика |
|---|---|
| `run.created`, `run.state_changed` | [state][g-uml], reason?, version, budgets; текущий snapshot доступен GET |
| `run.completed` | result_id, source_revision, result_revision?, diff_hash?, [config_hash][g-config-hash]; источник [review][g-review] [hook][g-hook] |
| `run.failed`, `run.cancelled` | reason, unknown_tool_ids, cleanup_state |
| `model.delta` | step_id, text, stream_index; незавершённый [stream][g-model-protocol] не равен final result |
| `model.completed`, `model.failed` | step_id, finish_reason/error_code, [usage][g-usage], usage_source |
| `tool.proposed`, `tool_result` | tool_call_id, tool_name, [state][g-uml]/outcome, result_ref?, safe_preview? |
| `approval_requested`, `approval.resolved` | approval_id, tool_name, args_hash, expires_at, [state][g-uml]/decision? |
| `agent_message` | message_id, from, to, message_type, target_run_id?, safe_preview |
| `message.delivered`, `message.rejected` | message_id, route_ref, reason?, target_run_id? |
| `hook.triggered`, `hook.rejected` | hook_id, hook_version, source_event_id, target_run_id?, reason? |
| `budget.updated` | scope, spent/reserved/limit, usage_source; не скрывать estimate |
| `workspace.cleanup` | workspace_id, cleanup_state, retain_until?, error_code? |

Поддержанные event_type для [hooks][g-hook]: ровно `run.completed`, `tool_result`, `approval_requested`, `agent_message`. Остальной каталог пригоден для наблюдения, но не автоматически для пользовательских [hooks][g-hook]. `deliver_message` вводит данные на границе [шага][g-step] конкретного активного адресата; `start_review` допустим только для успешного [coder][g-main-roles] `run.completed` с закреплённым [Result][g-result] и настроенной reviewer-version.

Replay возвращает сохранённые [события][g-event] с seq больше cursor, затем live из того же источника. Cursor другого [run][g-run]→400; cursor впереди high-water→409; ниже watermark после очистки→410 HISTORY_EXPIRED до начала [SSE][g-sse]. Клиент получает snapshot и его cursor и показывает «Часть истории недоступна». Повтор [event][g-event] ID не дублирует [UI][g-ui]; пропуск seq вызывает resync. Порядок меж разными [run][g-run] не обещается. Отзыв device закрывает [SSE][g-sse] ≤5 с; истёкший [JWT][g-jwt-jwks] также требует refresh/reconnect. [SSE][g-sse] не является каналом доставки [апрува][g-approval] закрытому [APK][g-apk].

<a id="messages"></a>
## 7. Межагентный [конверт][g-message] v1

[Сообщение][g-message] создаёт доверенный серверный adapter по реальному sender [run][g-run], а не произвольный HTTP JSON телефона. Envelope сохраняется неизменным после acceptance. Все обязательные поля перечислены в примере; `reply_to` nullable, payload schema определяется type. Неизвестные поля/версия, несовпадающий sender/[chain][g-chain], отключённая link и размер >65536 байт отклоняются. Размер считается по [UTF-8][g-identifier-formats] сериализации полного envelope.

```json
{
  "schema_version": 1,
  "id": "70000000-0000-4000-8000-000000000001",
  "run_id": "30000000-0000-4000-8000-000000000001",
  "correlation_id": "50000000-0000-4000-8000-000000000001",
  "from": "80000000-0000-4000-8000-000000000001",
  "to": "80000000-0000-4000-8000-000000000002",
  "type": "review.request",
  "payload": {"result_id":"90000000-0000-4000-8000-000000000001","source_revision":"PINNED_SHA","result_revision":"RESULT_SHA","diff_hash":"SHA256_DIFF"},
  "reply_to": null,
  "hops": 1,
  "depth": 1
}
```

| Message type | Payload / разрешённый эффект |
|---|---|
| `agent_message` | text, context_refs; только данные активному target [run][g-run] на границе [шага][g-step] |
| `review.request` | result_id, source_revision, result_revision, diff_hash; настроенный [hook][g-hook] создаёт отдельный [reviewer][g-main-roles] |
| `review.result` | result_id, reviewer_run_id, verdict approved/changes_requested/inconclusive, summary; результат [цепочки][g-chain], не auto-fix |
| `tool_result` | tool_call_id, result_ref, outcome; ограниченная передача данных при настроенном [hook][g-hook] |
| `approval_requested` | approval_id, expires_at, tool_name; уведомление/контекст, без полномочия approve |

Дополнительная route metadata вне envelope фиксирует link_id/version, hook_id/version и конкретный target_run_id для `deliver_message`. При нескольких подходящих активных [run][g-run] без однозначной привязки — ROUTE_AMBIGUOUS, сервер не выбирает произвольный «последний» [run][g-run]. Target должен принадлежать той же [chain][g-chain]/installation. `start_review` получает target_run_id из сохранённого receipt после создания дочернего [run][g-run].

Root [run][g-run] [depth][g-depth]=0, исходный route [hops][g-hops]=0. Каждая передача, включая reply, получает server [hops][g-hops]+1, новый дочерний [run][g-run] — parent [depth][g-depth]+1; максимумы 8/2, следующая передача/дочерний [run][g-run] сверх них запрещены. Caller-supplied counters не авторитетны. Same message ID+same canonical hash→тот же логический outcome; same ID+different hash→conflict. [Dedupe][g-idempotency] 7 суток, [redelivery][g-retry] не более 24 ч. Reply terminal [coder][g-main-roles] отображается как результат [chain][g-chain]; [terminal run][g-terminal-run] не меняется.

<a id="packages"></a>
## 8. [Экспорт][g-export-import], [staging][g-staging] и [activation][g-staging]

| Операция | Вход / результат |
|---|---|
| `GET /v1/agent-versions/{id}/package` | `application/zip`, безопасное имя файла, version/config hash; без [секретов][g-secret] |
| `POST /v1/package-imports` | Multipart один [ZIP][g-package] ≤10 MiB + key; [staging][g-staging]/validation, 201 import_id+[unresolved][g-staging]; не создаёт активного [агента][g-agent] |
| `GET /v1/package-imports/{id}` | [state][g-uml], manifest preview, dependencies, [mappings][g-staging], [unresolved][g-staging], [config_hash][g-config-hash]? |
| `PUT /v1/package-imports/{id}/mappings` | [provider][g-provider]/[model][g-model]/secret/[tool][g-tool]/[agent][g-agent] aliases + expected_version; проверка сервером |
| `POST /v1/package-imports/{id}/activate` | key+expected_version; при [unresolved][g-staging]→422, иначе один [AgentVersion][g-agent-version]+audit |
| `DELETE /v1/package-imports/{id}` | Удаление неактивного [staging][g-staging]; повтор безопасен |

Поля, JSON-пример и validation pipeline находятся в [DOC-038][id-doc-038]. Потеря upload/[activation][g-staging] ответа разрешается тем же key и GET [import][g-export-import]; новый ключ не создаётся молча. Выбор файла/сохранение через [Android][g-flutter] document picker входят в обязательную приёмку.

<a id="operations"></a>
## 9. [Аудит][g-audit], состояние и [health][g-readiness]

| Операция | Выход |
|---|---|
| `GET /healthz` | Публичный минимум: process alive; без конфигурации |
| `GET /readyz` | Публичный boolean/code; при [миграциях][g-environment]/[reconciliation][g-recovery]/dependency block→503; деталей/[секретов][g-secret] нет |
| `GET /v1/status` | Owner-only: [auth][g-auth]/db/[runtime][g-runtime]/[provider][g-provider]/[push][g-push] states, checked_at, причина, [readiness][g-readiness] по операции |
| `GET /v1/audit?run_id=&correlation_id=&cursor=` | Защищённые append-записи, safe metadata, server time; basic context query |
| `GET /v1/logs?run_id=&correlation_id=&cursor=` | Структурные диагностические записи без raw credentials/stdout |

Status dependency: `unknown/ready/degraded/unavailable`, отдельно `last_attempt/last_success/error_code`; [push][g-push] accepted by [FCM][g-push] не обозначается как доставленный на телефон. [Provider][g-provider] [ready][g-readiness] относится к конкретному [model][g-model]/config/probe, DB alive не доказывает готовность [Runtime][g-runtime]. Неверные context ID не снимают [owner][g-owner]/device guard. Расширенный поиск/immutable audit infrastructure [FR-079][id-fr-079] и [Prometheus][g-observability-tools] [FR-080][id-fr-080] вне базы.

<a id="sources"></a>
## 10. Внешние основания

Проверено по первичным источникам 10.10.2026: [Supabase JWT](https://supabase.com/docs/guides/auth/jwts), [Supabase password/email confirmation](https://supabase.com/docs/guides/auth/passwords), [Auth server-side session verification](https://supabase.com/docs/guides/auth/server-side/advanced-guide). Подпись [JWT][g-jwt-jwks] и локальная device revocation имеют разный смысл; конкретная настройка signing keys/confirmation template проверяется до интеграции. Прокси-операции Portway выше — проектный контракт, а не выдуманные endpoint-paths самого [Supabase][g-supabase].

## История

| Версия | Дата | Изменение |
|---|---|---|
| v0.3 | 14.10.2026 | Agent Bridge v2: Codex CLI и Claude Code, выбор модели/effort, лимиты, команды аккаунтов и snapshot моста ([раздел](#agent-bridge-v2)); полный MVP не заявлен |
| v0.2 | 13.10.2026 | Добавлен opt-in native Codex demo profile: контракт, ограничения, локальный мост; полный MVP и Android acceptance не заявлены |
| v0.1 | 10.10.2026 | Созданы HTTP DTO/операции, ошибки, SSE-каталог, envelope и контракты мобильного переноса |

[g-agent]: 07-glossary.md#term-agent

[g-agent-version]: 07-glossary.md#term-agent-version

[g-api]: 07-glossary.md#term-api

[g-apk]: 07-glossary.md#term-apk

[g-approval]: 07-glossary.md#term-approval

[g-atomicity]: 07-glossary.md#term-atomicity

[g-audit]: 07-glossary.md#term-audit

[g-auth]: 07-glossary.md#term-auth

[g-budget]: 07-glossary.md#term-budget

[g-cancelled]: 07-glossary.md#term-cancelled

[g-chain]: 07-glossary.md#term-chain

[g-ci-cd]: 07-glossary.md#term-ci-cd

[g-config-hash]: 07-glossary.md#term-config-hash

[g-depth]: 07-glossary.md#term-depth

[g-device-session]: 07-glossary.md#term-device-session

[g-environment]: 07-glossary.md#term-environment

[g-event]: 07-glossary.md#term-event

[g-export-import]: 07-glossary.md#term-export-import

[g-flutter]: 07-glossary.md#term-flutter

[g-go]: 07-glossary.md#term-go

[g-harness]: 07-glossary.md#term-harness

[g-hook]: 07-glossary.md#term-hook

[g-hops]: 07-glossary.md#term-hops

[g-https]: 07-glossary.md#term-https

[g-idempotency]: 07-glossary.md#term-idempotency

[g-identifier-formats]: 07-glossary.md#term-identifier-formats

[g-jwt-jwks]: 07-glossary.md#term-jwt-jwks

[g-link]: 07-glossary.md#term-link

[g-main-roles]: 07-glossary.md#term-main-roles

[g-mcp]: 07-glossary.md#term-mcp

[g-message]: 07-glossary.md#term-message

[g-model]: 07-glossary.md#term-model

[g-model-protocol]: 07-glossary.md#term-model-protocol

[g-observability-tools]: 07-glossary.md#term-observability-tools

[g-owner]: 07-glossary.md#term-owner

[g-package]: 07-glossary.md#term-package

[g-policy]: 07-glossary.md#term-policy

[g-provider]: 07-glossary.md#term-provider

[g-push]: 07-glossary.md#term-push

[g-push-token]: 07-glossary.md#term-push-token

[g-readiness]: 07-glossary.md#term-readiness

[g-recovery]: 07-glossary.md#term-recovery

[g-repository]: 07-glossary.md#term-repository

[g-result]: 07-glossary.md#term-result

[g-retry]: 07-glossary.md#term-retry

[g-review]: 07-glossary.md#term-review

[g-role]: 07-glossary.md#term-role

[g-run]: 07-glossary.md#term-run

[g-runtime]: 07-glossary.md#term-runtime

[g-secret]: 07-glossary.md#term-secret

[g-skill]: 07-glossary.md#term-skill

[g-sse]: 07-glossary.md#term-sse

[g-staging]: 07-glossary.md#term-staging

[g-step]: 07-glossary.md#term-step

[g-supabase]: 07-glossary.md#term-supabase

[g-task]: 07-glossary.md#term-task

[g-terminal-run]: 07-glossary.md#term-terminal-run

[g-timeout]: 07-glossary.md#term-timeout

[g-tool]: 07-glossary.md#term-tool

[g-ui]: 07-glossary.md#term-ui

[g-uml]: 07-glossary.md#term-uml

[g-usage]: 07-glossary.md#term-usage

[id-doc-036]: 36-api-events-and-messages.md

[id-doc-038]: 38-agent-package-format.md

[id-doc-040]: 40-agent-execution-and-security.md

[id-fr-001]: 03-software-requirements-specification.md#fr-001

[id-fr-003]: 03-software-requirements-specification.md#fr-003

[id-fr-010]: 03-software-requirements-specification.md#fr-010

[id-fr-017]: 03-software-requirements-specification.md#fr-017

[id-fr-028]: 03-software-requirements-specification.md#fr-028

[id-fr-059]: 03-software-requirements-specification.md#fr-059

[id-fr-079]: 03-software-requirements-specification.md#fr-079

[id-fr-080]: 03-software-requirements-specification.md#fr-080

[id-fr-081]: 03-software-requirements-specification.md#fr-081

[id-fr-084]: 03-software-requirements-specification.md#fr-084

[id-fr-085]: 03-software-requirements-specification.md#fr-085

[id-fr-086]: 03-software-requirements-specification.md#fr-086

[id-fr-090]: 03-software-requirements-specification.md#fr-090

[id-uc-001]: 12-use-case-diagrams-and-specifications.md#uc-001

Дополнительно реализованы `GET /v1/models`, `POST /v1/providers/openrouter`, private `POST /internal/v1/model-gateway/count|complete`; [cloud/Gateway contract](28-free-model-connections.md), [OpenAPI 0.6](../api/openapi.json). В отличие от `/v1/providers/{id}`, статический путь `/v1/providers/openrouter` не принимает UUID: это env provisioning с owner/device guard.

Текущий wire-профиль и реализованный `POST /v1/providers/{id}/capability-check` — [Gateway](27-model-gateway.md#capability-check). Executable контракт использует фиксированный `profile`, If-Match и 202 acknowledgement; результат читается через GET providers. Каталог моделей ещё не реализован; probe не открывает run admission.
