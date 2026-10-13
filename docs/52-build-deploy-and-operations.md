# Portway — сборка, развёртывание и эксплуатация

> **Изменение 11.10.2026 (D-22):** по решению владельца целевая архитектура — Go-микросервисы и Kafka. [Принятое изменение архитектуры](21-microservices-kafka-amendment.md) заменяет прежний монолит/DB-only шину; PostgreSQL outbox сохраняется для надёжной публикации. Старые схемы одного процесса ниже — baseline до изменения, не текущая целевая топология. Первый реализованный срез Kafka описан в [DOC-022](22-kafka-first-slice.md); полный переход ещё не завершён.


> **Фактический срез 12.10:** [статус](implementation-status.md), [план](implementation-plan.md), [Harness/Result](26-harness-turns-and-results.md). Проектные структуры далее задают целевой контракт; реализованное подмножество описано в коде/миграциях и [OpenAPI JSON](../api/openapi.json). NOT_RUN baseline не отрицает выполненные инженерные проверки.

<a id="doc-052"></a>

| Поле | Значение |
|---|---|
| Артефакт | [DOC-052][id-doc-052] |
| Версия и дата | v0.4 · 14.10.2026 |
| Статус | Runbook целевого MVP и реализованного подмножества; локальные проверки/native demo — evidence, полная приёмка NOT_RUN |
| Основания | [Архитектура](20-architecture-and-decisions.md), [безопасность/recovery](40-agent-execution-and-security.md), [SRS defaults](03-software-requirements-specification.md#defaults), [SMART](04-smart-goals-and-metrics.md#methods) |
| Проверка результата | [AC](14-demo-acceptance-criteria.md), [план/протокол](57-verification-and-traceability.md), [backlog](01-product-backlog.md) |

## Flutter Android: локальная сборка

Из `apps/portway`: `flutter pub get`, `flutter analyze`, `flutter test`, `flutter build apk --debug`. SDK Flutter 3.47.2/Dart 3.13.2, target Android API 36, lock-файл включён. APK: `apps/portway/build/app/outputs/flutter-apk/app-debug.apk`, установка `adb install -r <APK>`. Это debug signing; production signing ещё не настроен. Workflow `.github/workflows/flutter.yml` добавляет отдельные client gates; удалённый CI-прогон не заявляется по локальной сборке. Подробности [README клиента](../apps/portway/README.md), [evidence](evidence/flutter-client-2026-10-13.md).

<a id="environments"></a>
<a id="codex-demo-operations"></a>
## Native Codex и Claude Code на ноутбуке (Agent Bridge v2)

Проверенные CLI: Codex 0.159.3 (вход ChatGPT) и Claude Code 2.1.296 (подписка Claude), Linux x86_64. Авторизации должны принадлежать владельцу установки; используются квоты существующих подписок, платные model API keys не нужны и мостом не выбираются (Codex только `Logged in using ChatGPT`, Claude только `authMethod=claude.ai`). Один процесс `portway codex-bridge` обслуживает оба CLI; отсутствующий или неавторизованный CLI показывается в статусе (`installed=false` / `logged_in=false`) и не мешает запуску. Мост запускается на хосте обычным пользователем, API остаётся в Docker.

1. Собрать `bin/portway` (`make build-server`, Go 1.27), выполнить `make docker-test` и `make docker-smoke`.
2. `python3 scripts/configure-codex-bridge.py` подготовит dedicated random key, workspace и пустые `PORTWAY_CLAUDE_BINARY`/`PORTWAY_BRIDGE_DATA_DIR` в игнорируемом `.env` без вывода/перезаписи существующих секретов. Пустой путь CLI = поиск `codex`/`claude` в PATH; пустой data dir = `$XDG_DATA_HOME/portway/bridge` (обычно `~/.local/share/portway/bridge`).
3. После разрешения выполнить аддитивную миграцию 010 и пересобрать/пересоздать API **и** оба Kafka workers (`outbox-relay`, `audit`): новое audit action `codex.command_queued` старый relay отклонит и остановит очередь outbox. Старый API не стартует с новой схемой (Ready проверяет exact migration set); rollback — только binary с поддержкой 010, таблицы не удаляются.
4. `python3 scripts/start-codex-bridge.py --check` печатает состояние входа Codex и Claude (без email и секретов) и не падает, если один CLI не авторизован. Затем `python3 scripts/start-codex-bridge.py` в отдельном терминале. Скрипт передаёт мосту только его параметры из `.env`, базовые переменные ОС, `CODEX_HOME`/`CLAUDE_CONFIG_DIR` и переменные дисплея (последние используются только чтобы открыть браузер при входе методом «на компьютере»). Ctrl+C останавливает мост, активный процесс и незавершённый вход; restart не повторяет uncertain task.
5. В Flutter использовать HTTPS URL установки и обычный login; `GET /v1/codex/status` должен дать `connected=true` и `providers` с обоими CLI. Loopback URL `http://127.0.0.1:8080` относится только к мосту на ноутбуке.

Аккаунты. Системный вход ноутбука (`~/.codex`, `~/.claude`) отображается как `codex-system`/`claude-system`; мост его не разлогинивает и не удаляет. Новый аккаунт подключается из телефона: «на телефоне» — device code ChatGPT (ChatGPT может требовать включить device-code login в Settings → Security) или ссылка Claude + вставка кода со страницы platform.claude.com; «на компьютере» — браузер открывается на ноутбуке. Каждый успешный вход создаёт отдельный профиль `accounts/<provider>/<id>/` (0700, используется как `CODEX_HOME`/`CLAUDE_CONFIG_DIR`) и становится активным; реестр `accounts.json` — 0600. Удаление выполняет logout в этом профиле и удаляет каталог; незавершённые/отменённые входы не оставляют каталогов. Лимиты читаются без inference не чаще раза в минуту на провайдера: Codex через `codex app-server` (`account/rateLimits/read`), Claude через OAuth usage endpoint с токеном профиля (best-effort, иначе `unknown`).

Запуски: Codex — прежний `exec --json --sandbox read-only --ignore-user-config …` плюс `-m <model>` и `-c model_reasoning_effort=…`; Claude — `claude -p --output-format stream-json --verbose --tools Read,Grep,Glob --allowedTools Read,Grep,Glob --permission-mode dontAsk --strict-mcp-config --no-session-persistence --safe-mode --restricted [--model …] [--effort …]`, prompt через stdin, cwd = workspace, тот же 9-минутный лимит. `--bare` не используется: он отключает OAuth подписки. Окружение CLI собирается явно (HOME, PATH, USER, SHELL, XDG_*, SSL_*, профиль); ключи Portway, `*_API_KEY` и переменные родительского Claude Code не передаются.

Мост не устанавливается как автозапускаемая системная служба этим срезом. Ноутбук должен бодрствовать, CLI иметь рабочую авторизацию и сеть; offline видно клиенту. Endpoint `/readyz` и generic `operations.run` остаются закрыты для полного MVP; готовность demo определяется `/v1/codex/status`, а не общим ready. Протокол текущих тестов: [Codex bridge evidence](evidence/codex-bridge-2026-10-13.txt). 13.10 production обновлён до 001–009; live checks и native heartbeat PASS. Native bridge тогда был запущен в background; PID/log в игнорируемых `.tmp/codex-bridge.pid`/`.tmp/codex-bridge.log` (14.10 процесс с этим PID не работает). Agent Bridge v2: [evidence 14.10](evidence/agent-bridge-2026-10-13.txt); production до миграции 010 ещё не обновлён. Это не автозапуск после перезагрузки. Для остановки отправить SIGTERM именно PID из файла (после проверки процесса), для нового запуска использовать команду выше. Реальное Android устройство пока NOT_RUN; live deploy и локальный native roundtrip различаются в evidence.


## 1. Окружения и минимальный состав

| Окружение | Назначение / зависимости | Что не доказывает |
|---|---|---|
| `ci-fake` | Чистая сборка, unit/contract/integration, disposable [PostgreSQL][g-supabase] и [fake][g-fixture] [LLM][g-model]/[MCP][g-mcp]/[FCM][g-push]/[Runtime][g-runtime] boundary | Не доказывает контейнерную изоляцию, физический [Android][g-flutter]/[push][g-push], локальную [модель][g-model] и перенос B |
| `lab-A` | CachyOS i5-13420H/16 GB, [Go][g-go] [API][g-api], rootless [Runtime][g-runtime] Manager/engine, два [sandbox][g-sandbox], реальные внешние сервисы, [APK][g-apk] [Android][g-flutter] 16 | Не заменяет отдельную установку B |
| `lab-B` | Другой host/VM с отдельными installation_id/[API][g-api]/[Runtime][g-runtime]/config/secrets/[workspace][g-workspace]; [model][g-model] M2 | Смена каталога/имени A не является B |
| `candidate` | Закреплённые source revision, lock-файлы, [APK][g-apk] hash, image digests, DB migration version, config fingerprint | Не считается принятой без фактических [TC][g-acceptance]/[AC][g-acceptance] [evidence][g-evidence] |

[Go][g-go] [API][g-api] и trusted Manager работают под разными локальными аккаунтами. Manager владеет rootless engine и управляемыми томами, [API][g-api] получает только узкий Unix socket. HTTPS-туннель публикует loopback [API][g-api], не daemon/[model][g-model]/SQL/Manager. [Supabase][g-supabase] [Auth][g-auth]/[PostgreSQL][g-supabase] и [FCM][g-push] внешние; готовые аккаунты/доступы ещё надо проверить. Local [model][g-model] service не входит в лимиты [sandbox][g-sandbox]. Docker [compose][g-environment] может описывать вспомогательные lab/[fake][g-fixture] зависимости; [API][g-api] не получает socket/host mounts через [compose][g-environment] ради удобства.

До первого реального [run][g-run] проверить rootless engine/cgroups, двухконтейнерные CPU/RAM лимиты, egress enforcement, свободное место, доступ к images/[model][g-model] files, две реальные model-конфигурации и B. Начальная загрузка images/[models][g-model]/SDK не входит в ≤600 с, но её длительность/[шаги][g-step] записываются отдельно. 275 GB свободного места — исходная заявленная характеристика, не текущий замер.

<a id="configuration"></a>
## 2. Конфигурация и [секреты][g-secret]

| Переменная / источник | Смысл и правило |
|---|---|
| `PORTWAY_INSTALLATION_ID` | Стабильный [UUID][g-identifier-formats] данной установки; на B новый |
| `PORTWAY_HTTP_ADDR` | Loopback bind, например `127.0.0.1:8080`; наружу только через [HTTPS][g-https] |
| `PORTWAY_PUBLIC_URL` | Реальный [HTTPS][g-https] base URL; обновить email callback allowlist/template и [APK][g-apk] при смене |
| `PORTWAY_DATABASE_URL` / `PORTWAY_DATABASE_URL_FILE` | DSN или путь к файлу DSN с TLS verification; не значение в git/[APK][g-apk] |
| `PORTWAY_MIGRATION_DATABASE_URL` / `_FILE` | Отдельный operator DSN только для мигратора; API его не получает |
| `PORTWAY_DATABASE_MAX_CONNS` | Текущий API: 2 по умолчанию, 1–8; мигратор: 1 |
| `PORTWAY_AUTH_URL`, `PORTWAY_AUTH_PUBLISHABLE_KEY` | URL [Auth][g-auth] и публичный client key проекта; не service-role/[JWT][g-jwt-jwks] signing secret |
| `PORTWAY_MASTER_KEY` / `PORTWAY_MASTER_KEY_FILE` | Отдельный authenticated-encryption key; не хранить вместе с DB backup |
| `PORTWAY_SETUP_TOKEN` / `PORTWAY_SETUP_TOKEN_FILE` | Одноразовый [setup proof][g-device-session] для [owner][g-owner] claim; на сервере hash, не логировать |
| `PORTWAY_AUTH_CONFIRM_REDIRECT` | Фиксированный `portway://auth/confirmed`; arbitrary next запрещён |
| `PORTWAY_RUNTIME_SOCKET`, `PORTWAY_RUNTIME_CREDENTIAL_FILE` | Локальный Manager endpoint и отдельный scoped credential |
| `PORTWAY_FCM_PROJECT_ID`, `PORTWAY_FCM_CREDENTIALS_FILE` | Server credential только notifications по D-22; client Firebase config в [APK][g-apk] без server private key |
| `PORTWAY_LLM_ALLOWED_ENDPOINTS` | Явный набор configured [model][g-model] destinations; не prompt-driven SSRF |
| `PORTWAY_MCP_ALLOWED_ENDPOINTS` | Scoped documentation adapter destinations |
| `PORTWAY_RUNTIME_IMAGE_DIGESTS` | Allowlist подготовленных images, без произвольного Dockerfile из [пакета][g-package] |
| `PORTWAY_LOG_LEVEL`, retention/default profile | Safe fields; численные обязательства берутся из [SRS][g-requirement-ids], не переопределяются молча |

По решению владельца текущий Compose читает `.env` (0600), исключённый из Git и Docker context. Либо бинарь принимает secret-file mounts через `_FILE`; одновременно значение и файл запрещены. Docker-оператор может видеть env, поэтому полный `docker compose config` не публикуется; для проверки используется `--quiet`. Значения не печатаются при config validation. Для production с secret files — owner-only чтение для нужного процесса. App DB [role][g-role] отделён от migration/operator [role][g-role]; private schema/[RLS][g-supabase]/[grants][g-fencing] проверяются негативным anonymous запросом. Для IPv4 [Supabase][g-supabase] session pooler возможен по фактическим реквизитам проекта; pooler не отключает TLS verification. Master-key rotation сохраняет [key_version][g-encryption] и старый decrypt key до проверенной перешифровки/restore; удаление ключа без проверки восстановления недопустимо.

[Provider][g-provider] key вводится write-only [owner][g-owner] [API][g-api] и хранится шифротекстом, не env агентского [контейнера][g-sandbox]. [FCM][g-push]/DB/master/Manager secrets также не попадают в [tool][g-tool] environment. Публичная Firebase client configuration и [Supabase][g-supabase] publishable key не заменяют серверные access checks. Маркерные secret scans и restore входят в [DOC-057][id-doc-057].

<a id="build"></a>
## 3. Контракт сборки и [CI][g-ci-cd]

Следующие targets — интерфейс, который должна предоставить реализация. Сейчас Makefile, исходники и pipeline отсутствуют; команды ниже не выполнялись и не представлены как готовая инструкция [запуска][g-run] существующего кода.

```sh
make bootstrap-check
make fmt-check
make lint
make test-unit
make test-contract
make test-integration
make build-server
make build-apk
make migrate
make run-api
make run-runtime-manager
make smoke
```

| Target / этап | Обязательный результат |
|---|---|
| bootstrap-check | Версии [Go][g-go]/[Flutter][g-flutter]/Dart/[Android][g-flutter] SDK/JDK/engine/SQL [tools][g-tool], lock-файлы, ресурсы, safe config; [секреты][g-secret] не выводятся |
| fmt/lint/unit | Чистое форматирование, статический анализ и доменные проверки; контрольная ошибка ломает этап |
| contract | DTO/envelope/package schemas, HTTP errors, [SSE][g-sse] cursor examples, соответствие [OpenAPI][g-api] реализации [DOC-036][id-doc-036] |
| integration | Реальный disposable [PostgreSQL][g-supabase], migration fresh/upgrade, race/[dedupe][g-idempotency]/[outbox][g-outbox] с [fake][g-fixture] внешними ports |
| build-server | [API][g-api]/Manager modes одной codebase, version/revision в binary; воспроизводимый dependency lock |
| build-apk | [Release][g-ci-cd] [APK][g-apk] с нужной notification/deep-link/document-picker конфигурацией; checksum, installability |
| migrate | Один оператор [миграции][g-environment], checksum/version; incompatible schema блокирует [ready][g-readiness] |
| [smoke][g-fixture] | [Health][g-readiness], [auth][g-auth] guard, DB/[Runtime][g-runtime] [readiness][g-readiness] и простой controlled [run][g-run]; [fake][g-fixture]/real уровень подписан |

[CI][g-ci-cd] запускает безопасные checks без настоящих [provider][g-provider]/[FCM][g-push]/DB credentials и не делает внешние платные вызовы. [APK][g-apk] signing material хранится отдельно от git/[логов][g-audit]; для обновления установленного [APK][g-apk] используется тот же ключ. В pipeline сохраняются source revision, dependency locks, reports и artifact hashes. Нужны два последовательных зелёных CI-прогона одного кандидата; повтор только упавшей проверки не скрывает исходный FAIL. Точные toolchain/library versions фиксируются при реализации и проверяются по официальной совместимости, а не заявляются выбранными заранее.

<a id="install"></a>
## 4. Первичная подготовка и [запуск][g-run] lab

1. Подготовить и закрепить toolchains, images/[model][g-model] files; создать разные [API][g-api]/[runtime][g-runtime] accounts и права на Unix socket, настроить rootless daemon/cgroups/egress. Проверить, что [агент][g-agent]/[API][g-api] не видят daemon socket и host FS.
2. Создать/проверить Supabase-проект, закрытую схему/[grants][g-fencing], TLS DSN, migration [role][g-role] и [Auth][g-auth] signing/email confirmation. FCM-проект и [Android][g-flutter] package/signing config должны соответствовать [APK][g-apk]. Заполнить `.env` либо secret files без публикации значений.
3. Задать installation [UUID][g-identifier-formats], allowlists, Master/Manager/setup credentials. Применить [миграции][g-environment], запустить Manager, затем [Go][g-go] [API][g-api]. До завершения [reconciliation][g-recovery] [ready][g-readiness]=false; увидеть конкретные dependency states под [owner][g-owner] [API][g-api] после входа.
4. Поднять HTTPS-туннель к loopback [API][g-api]. По решению владельца от 11.10 текущая установка использует Tailscale Funnel с постоянным DNS-именем узла; настройка и Supabase callback описаны в [инструкции развёртывания](../deploy/supabase/README.md). Проверить сертификат, route с физического телефона, отсутствие buffering/обрывов [SSE][g-sse]. Проверка текущего HTTP API через Funnel не подтверждает работу SSE: streaming ещё не реализован и требует отдельной проверки после добавления.
5. Настроить [Supabase][g-supabase] email link на фиксированный [HTTPS][g-https] `/auth/confirm`, [Android][g-flutter] callback scheme; пройти setup/регистрацию/email confirmation/вход. После claim [setup proof][g-device-session] больше не принимает нового [owner][g-owner].
6. Установить [APK][g-apk], ввести текущий [HTTPS][g-https] [API][g-api] URL, войти, выдать notification permission по желанию [владельца][g-owner], проверить token registration. Server [push][g-push] accepted не равен delivery на телефоне.
7. Реализованный probe вызывается с `If-Match`, `Idempotency-Key` и `{"profile":"chat-completions-v1"}`; 202 acknowledgement, текущий результат/`protocol_valid` — GET providers. Не повторять запрос с новым key для восстановления потерянного ответа: тот же key безопасно воспроизводит acknowledgement. Перед реальным probe выбрать endpoint и учитывать максимум две генерации по 128 запрошенных output tokens плюс input. [Детали и сроки отчёта](27-model-gateway.md#capability-check). Создать [provider][g-provider] bindings для M1/M2, выполнить реальные capability probes, настроить [MCP][g-mcp] и [repository][g-repository] source/commit. Создать [coder][g-main-roles]/[reviewer][g-main-roles], направленную link и completion [hook][g-hook]. Проверить negative [policy][g-policy] до настоящих shell [tools][g-tool].
8. Подготовить B с отдельным [API][g-api]/Manager и local bindings, затем мобильный перенос по [DOC-038][id-doc-038]. Только после этого выполнить полный сценарий/измерения [DOC-057][id-doc-057].

Подготовленный старт ≤600 с измеряется трижды из одинакового состояния с уже скачанными images/[models][g-model] и заведёнными аккаунтами; старт включает миграционные проверки, [reconciliation][g-recovery] и реальные проверки нужных зависимостей. Недоступная зависимость делает trial неуспешным, даже если процесс запустился.

<a id="usage"></a>
## 5. Короткая инструкция [владельцу][g-owner]

Войти в нужную установку → выбрать/создать [агента][g-agent] и неизменяемую версию → выбрать task/[repository][g-repository] revision/[model][g-model]/лимиты → начать [run][g-run]. На экране [задачи][g-task] видны [state][g-uml]/причина, [события][g-event], расход с source, [stop][g-cancelled] [run][g-run]/[chain][g-chain]. Опасный [tool][g-tool] исполняется только после актуального server [approval][g-approval]; при отказе возможна новая проверенная [безопасная альтернатива][g-safe-alternative]. Результат [coder][g-main-roles] и отдельный [review][g-review] доступны без [GitHub][g-github] PR. После замечаний/сбоя новую [задачу][g-task] создаёт [владелец][g-owner] вручную.

[Экспорт][g-export-import]: выбрать версию и сохранить [ZIP][g-package] системным picker. [Импорт][g-export-import]: выбрать [ZIP][g-package], проверить preview, сопоставить [model][g-model]/[tools][g-tool]/[MCP][g-mcp]/secret/[agent][g-agent] aliases, создать версию и запустить контрольную [задачу][g-task]. При [unresolved][g-staging] [mapping][g-staging] нельзя запускать [агента][g-agent]. Смена [API][g-api] URL на B очищает контекст A и требует нового входа; local logout не означает remote revoke без ACK. Профиль даёт управление устройствами/уведомлениями, состояние — причины недоступности и audit/logs.

<a id="release"></a>
## 6. Candidate, обновление и [rollback][g-runbook]

Candidate manifest: git revision/dirty [state][g-uml], [Go][g-go]/[Flutter][g-flutter]/Dart/JDK/SDK versions, [APK][g-apk] [SHA-256][g-config-hash]/signing identity, [API][g-api]/Manager build version, image digests, DB migration version, safe configuration fingerprint, [model][g-model] IDs и endpoint labels, установка A/B, [evidence][g-evidence] index. Секретные значения и обратимо закодированные [секреты][g-secret] в fingerprint не включаются.

Обновление: сохранить [evidence][g-evidence]/backup → остановить admission новых [run][g-run] → [stop][g-cancelled] и дождаться controlled terminal/явного unknown → применить совместимую [миграцию][g-environment] → обновить [API][g-api]/Manager/[APK][g-apk] → [reconciliation][g-recovery]/[readiness][g-readiness] → [smoke][g-fixture] и затронутые [TC][g-acceptance] → полный Android-прогон кандидата. [APK][g-apk] обновляется тем же signing key; изменение callback/[API][g-api] URL требует проверки [auth][g-auth]/[push][g-push]/deep links.

[Rollback][g-runbook] binary допустим только при совместимой DB-схеме и [Runtime][g-runtime] [grant][g-fencing]/version protocol. При несовместимости не запускать старый binary поверх новой схемы: forward-fix или восстановление из проверенного snapshot с отдельно сохранённым [evidence][g-evidence]. Restore в другую installation не переносит действующие device/[grants][g-fencing] автоматически; все старые [approvals][g-approval]/[run][g-run] permissions инвалидируются. Неопределённые внешние эффекты сначала сверяются, не переисполняются.

<a id="runbook"></a>
## 7. [Runbook][g-runbook] отказов

| Симптом | Проверить / действие | Критерий восстановления |
|---|---|---|
| [APK][g-apk] не видит [API][g-api] | URL/TLS/tunnel/phone route; loopback [API][g-api] [health][g-readiness]; запрет cleartext | [HTTPS][g-https] с телефона, [auth][g-auth] guard и [SSE][g-sse] работают |
| [Health][g-readiness]=ok, [run][g-run] blocked | Protected status: schema/[reconciliation][g-recovery]/DB/[Runtime][g-runtime]/[model][g-model] capabilities | Нужная операция [ready][g-readiness], причины/checked_at актуальны |
| [Supabase][g-supabase] DB unavailable | TLS/DSN/pooler/доступность; не печатать credential; остановить новые [grants][g-fencing] | DB доступна, сверены durable intents, unknown не повторены |
| [LLM][g-model] [timeout][g-timeout]/incompatible | [Model][g-model]/config/capability report/ресурс; типизированная ошибка | Реальный probe и [run][g-run], [usage][g-usage]/source не выдуманы |
| [MCP][g-mcp] недоступен | Allowlist/adapter credential/network/[timeout][g-timeout] | Реальный docs lookup, не [fake][g-fixture] для demo |
| [FCM][g-push] accepted, уведомления нет | Permission/token/FCM-project/phone lifecycle/[TTL][g-timeout]; [force-stop][g-app-lifecycle] отдельно | Реальные [background][g-app-lifecycle]/[terminated][g-app-lifecycle] trials с timestamps |
| Pending [approval][g-approval] «не работает» | Server deadline/version/[owner][g-owner]/device/[stop][g-cancelled]/args hash | GET latest, корректный outcome без второго dispatch |
| Manager ACK потерян | Query того же [dispatch_id][g-fencing] и ОС-процессы; не выдавать новый ID | Saved outcome либо explicit unknown и отмена |
| [API][g-api] restart | [Ready][g-readiness]=false; reconcile ≤2 interrupted [runs][g-run]; pending invalidated | Нет старого [Harness][g-harness]/[grant][g-fencing], новый [run][g-run] только вручную |
| Disk pressure/cleanup error | Managed volumes/retention, сначала копия материалов защиты | Не удалена active работа/пользовательский [ZIP][g-package], cleanup outcome виден |
| Offline logout | Локальная очистка, [retry][g-retry] только server revocation либо другой device/operator | Durable device revoke+[push][g-push] unlink подтверждены |

Доверенные operator-команды provision/revoke/reconcile/backup должны иметь локальный доступ и safe audit; их CLI syntax определяется реализацией. [Runbook][g-runbook] не предлагает обойти [policy][g-policy], включить privileged или вручную переписать [terminal run][g-terminal-run] ради успешного демо.

<a id="data-safety"></a>
## 8. Сохранность данных и материалов

Перед [миграцией][g-environment]/кандидатом/cleanup сохраняется проверяемый snapshot нужной бизнес-схемы и отдельно [evidence][g-evidence] artifacts. БД содержит шифротекст [секретов][g-secret]; master key копируется отдельным защищённым каналом, backup не публикуется. Процедура operator backup/restore проверяется на disposable установке: схема/версии/конфигурация читаются, hashes совпадают, [секрет][g-secret] расшифровывается нужным key, старые device/[grants][g-fencing] не становятся действующими.

[SRS][g-requirement-ids] retention: [events][g-event]/logs/[dedupe][g-idempotency]/request keys 7 суток, audit 30 суток, [workspace][g-workspace] result 24 ч после terminal; материалы защиты копируются раньше. Пользовательский [ZIP][g-package] не считается disposable [staging][g-staging]. Restore test не вводит новый обязательный продуктовый интерфейс резервного копирования. Пока нет реального backup/restore, статус [TC-031][id-tc-031] NOT_RUN.

Защита остаётся 12.10, кандидат/репетиция 11.10; фактический остаток часов [владельца][g-owner] не известен. С 11.10 новые S/C не начинаются. Завершение этих восьми документов не закрывает [BL][g-work-ids]/[CW][g-work-ids]: нужны код, реальные интеграции, глава/источники курсовой и фактическая репетиция.

## История

| Версия | Дата | Изменение |
|---|---|---|
| v0.4 | 14.10.2026 | Agent Bridge v2: Codex CLI + Claude Code на ноутбуке, аккаунты, лимиты, порядок миграции 010 и пересборки API/relay/audit |
| v0.3 | 13.10.2026 | Добавлены сборка, тесты, APK и CI первого Flutter-среза |
| v0.2 | 13.10.2026 | Добавлен opt-in native Codex demo profile: контракт, ограничения, локальный мост; полный MVP и Android acceptance не заявлены |
| v0.1 | 10.10.2026 | Созданы контракты сборки/[CI][g-ci-cd]/config, [deployment][g-ci-cd] [steps][g-step], инструкции, [runbook][g-runbook] и сохранение [evidence][g-evidence] |

[g-acceptance]: 07-glossary.md#term-acceptance

[g-agent]: 07-glossary.md#term-agent

[g-api]: 07-glossary.md#term-api

[g-apk]: 07-glossary.md#term-apk

[g-app-lifecycle]: 07-glossary.md#term-app-lifecycle

[g-approval]: 07-glossary.md#term-approval

[g-audit]: 07-glossary.md#term-audit

[g-auth]: 07-glossary.md#term-auth

[g-cancelled]: 07-glossary.md#term-cancelled

[g-chain]: 07-glossary.md#term-chain

[g-ci-cd]: 07-glossary.md#term-ci-cd

[g-config-hash]: 07-glossary.md#term-config-hash

[g-device-session]: 07-glossary.md#term-device-session

[g-encryption]: 07-glossary.md#term-encryption

[g-environment]: 07-glossary.md#term-environment

[g-event]: 07-glossary.md#term-event

[g-evidence]: 07-glossary.md#term-evidence

[g-export-import]: 07-glossary.md#term-export-import

[g-fencing]: 07-glossary.md#term-fencing

[g-fixture]: 07-glossary.md#term-fixture

[g-flutter]: 07-glossary.md#term-flutter

[g-github]: 07-glossary.md#term-github

[g-go]: 07-glossary.md#term-go

[g-harness]: 07-glossary.md#term-harness

[g-hook]: 07-glossary.md#term-hook

[g-https]: 07-glossary.md#term-https

[g-idempotency]: 07-glossary.md#term-idempotency

[g-identifier-formats]: 07-glossary.md#term-identifier-formats

[g-jwt-jwks]: 07-glossary.md#term-jwt-jwks

[g-main-roles]: 07-glossary.md#term-main-roles

[g-mcp]: 07-glossary.md#term-mcp

[g-model]: 07-glossary.md#term-model

[g-outbox]: 07-glossary.md#term-outbox

[g-owner]: 07-glossary.md#term-owner

[g-package]: 07-glossary.md#term-package

[g-policy]: 07-glossary.md#term-policy

[g-provider]: 07-glossary.md#term-provider

[g-push]: 07-glossary.md#term-push

[g-readiness]: 07-glossary.md#term-readiness

[g-recovery]: 07-glossary.md#term-recovery

[g-repository]: 07-glossary.md#term-repository

[g-requirement-ids]: 07-glossary.md#term-requirement-ids

[g-retry]: 07-glossary.md#term-retry

[g-review]: 07-glossary.md#term-review

[g-role]: 07-glossary.md#term-role

[g-run]: 07-glossary.md#term-run

[g-runbook]: 07-glossary.md#term-runbook

[g-runtime]: 07-glossary.md#term-runtime

[g-safe-alternative]: 07-glossary.md#term-safe-alternative

[g-sandbox]: 07-glossary.md#term-sandbox

[g-secret]: 07-glossary.md#term-secret

[g-sse]: 07-glossary.md#term-sse

[g-staging]: 07-glossary.md#term-staging

[g-step]: 07-glossary.md#term-step

[g-supabase]: 07-glossary.md#term-supabase

[g-task]: 07-glossary.md#term-task

[g-terminal-run]: 07-glossary.md#term-terminal-run

[g-timeout]: 07-glossary.md#term-timeout

[g-tool]: 07-glossary.md#term-tool

[g-uml]: 07-glossary.md#term-uml

[g-usage]: 07-glossary.md#term-usage

[g-work-ids]: 07-glossary.md#term-work-ids

[g-workspace]: 07-glossary.md#term-workspace

[id-doc-036]: 36-api-events-and-messages.md

[id-doc-038]: 38-agent-package-format.md

[id-doc-052]: 52-build-deploy-and-operations.md

[id-doc-057]: 57-verification-and-traceability.md

[id-tc-031]: 57-verification-and-traceability.md#tc-031
