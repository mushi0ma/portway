# Portway — фактический статус реализации

> **Дополнение 11.10: реальное развёртывание.** [Первый срез Kafka](22-kafka-first-slice.md) запущен с рабочей Supabase. Миграции 001–004 применены, роли API/relay/audit подключаются с `verify-full`, приватные схемы закрыты для `anon/authenticated`. API, Kafka, relay и audit — healthy. Tailscale Funnel публикует `http://127.0.0.1:8080` по `https://cachyos-x8664.tail1443b8.ts.net`; новый `PORTWAY_PUBLIC_URL` применён к контейнеру API. Проверены локальные и публичные HTTP-маршруты, включая отдельную проверку через публичный DNS в обход MagicDNS. [Настройка callback в Dashboard](../deploy/supabase/README.md) и signup/email end-to-end ещё не подтверждены. [DB deployment](evidence/live-deployment-2026-10-11.txt), [текущий протокол Funnel](evidence/funnel-2026-10-11.txt).

**11.10.2026, Asia/Almaty. Первый срез: Go foundation / identity / agent configuration.** Нормативная база остаётся в ТЗ/SRS/MoSCoW; этот файл фиксирует код и доступные свидетельства. Реализация первого среза не означает приёмку полного MVP.

## Выполнено в коде

| Область | Код / проверка | Связь с проектом и остаток |
|---|---|---|
| Go и Docker | `cmd/portway`, Dockerfile, основной и тестовый Compose, Makefile, GitHub Actions | BL-001/002/006, DOC-020/052. Backend builds и проверки подготовлены; удалённый CI/APK не выполнены |
| БД | `internal/storage`, миграции 001–004; закрытая схема, отдельная app role, checksum, транзакционный migration lock | BL-004, DOC-029. Fresh/repeat/checksum/upgrade проверяются на disposable PostgreSQL; миграции применены в рабочей Supabase; роль API и TLS проверены |
| Identity | `internal/identity`, Supabase HTTP adapter, JWKS/legacy verification, setup/confirmation, owner/device guard, login/refresh/logout/revoke | BL-100/121/123/124, FR-001–004/081/084/085 частично. Auth settings endpoint проверен; реальный signup/email, Android, SSE и push unlink ещё не проверены/не реализованы |
| Агенты и роли | `internal/agents`, HTTP API, 5+3 presets, строгая config/policy валидация, immutable versions, ETag | BL-010/011/012/014. Проверены сохранение, чтение, идемпотентность и гонки; pinning в run и runtime policy ещё впереди |
| Providers/secrets | Allowlisted configuration, capabilities initially unknown, AES-256-GCM с AAD/nonce, metadata-only reads | BL-020/025 частично, DOC-040. Реального LLM/probe нет; rotation/restore ещё не реализованы |
| Kafka / доставка аудита | `cmd/events`, `internal/events`, `compose.kafka.yaml`, migration 004 | D-22 / DOC-022, BL-061 частично. Relay → реальный Kafka → private audit projection; duplicate/crash/offset и права проверены. Execution/FCM/query API/DLQ ещё впереди |
| Аудит и status | Append-only app permissions, атомарный audit с mutation, курсор, readiness по операциям | BL-103/104 частично. Run/chain context, persistent application logs, dependency probes и retention cleanup ещё впереди |
| HTTP контракт | `api/openapi.json`, strict UTF-8/JSON/duplicate/unknown checks, безопасные ошибки и request IDs | DOC-036: исполняемый OpenAPI 3.1 JSON для реализованного подмножества. Полный проектный каталог маршрутов не выдаётся за готовый API |

## Свидетельства

- [Tailscale Funnel / live checks](evidence/funnel-2026-10-11.txt): `make check-live` — PASS; отдельная проверка публичного DNS/HTTPS с сохранённой проверкой сертификата — PASS. Installation совпадает, auth guard — 401, callback без токена — 400, readiness — ожидаемые 503 до реализации execution. Проверка с физического телефона и SSE не выполнялись.

- [Kafka audit / Supabase / tunnel verification](evidence/kafka-audit-2026-10-11.txt): реальный Kafka + disposable PostgreSQL, duplicate/crash/offset и service-role isolation — PASS; API production smoke — PASS. Рабочая Kafka запущена. Исторический протокол до исправления DB credentials; последующий [live deployment](evidence/live-deployment-2026-10-11.txt) подтверждает рабочие DB-подключения и запуск.

- `make docker-test`: format, `go vet`, unit tests с race detector, сверка маршрутов OpenAPI, integration HTTP → Auth fixture + PostgreSQL, build-server. В первом прогоне выявлено отсутствие UPDATE-права для блокировки owners; исправлено блокировкой installation без расширения прав изменения identity. Во втором выявлена передача `[]byte` как bytea вместо JSON при pooler-compatible pgx mode; исправлено явным `json.RawMessage`. Последующий прогон прошёл.
- Интеграционные проверки: один signup при повторе, bind только после email confirmation, фиксированный callback без токенов, subject/session/device enforcement, refresh и revoke, локальный logout при недоступном Auth, AES-GCM roundtrip, отказ endpoint вне allowlist, 8 одновременных повторов create → один agent, race двух версий → один успех/один конфликт, сохранность исторической версии, запрет SQL update/delete версии и audit, закрытая anonymous схема, изоляция installation, отсутствие secret markers в ответах/логах.
- Unit-проверки: JWT signature/issuer/audience/expiry/session/algorithm/JWKS fetch bounds; role permissions/defaults/hash; AES-GCM tampering/AAD/nonces; secret-file permissions/TLS; строгий JSON.
- Финальный `make docker-test` — PASS, включая upgrade с identity-схемы, сохранение данных и запрет привилегированной роли API. `make docker-smoke` — PASS: реальный production image, non-root/read-only, миграции, health/auth guard, честный readiness=503, перезапуск. Основной `docker compose config --quiet` — PASS на публичных placeholders. Временные контейнеры/сети удалены. [Протокол и fingerprint исходников](evidence/backend-foundation-2026-10-11.txt).
- Git-репозиторий инициализирован, ветка `main`; коммита и remote пока нет. Реальная Supabase DB/Auth и Kafka запущены/проверены в объёме текущего протокола. Удалённые CI, FCM/LLM/MCP/Runtime и Android не запускались.

Ни один TC/AC из DOC-057/014 не переведён целиком в PASS: выполнены части TC-001/002/004/017/021/022/029/031 на тестовом окружении. Измерения p95/stop/подготовленного старта, два зелёных удалённых CI-прогона, перенос на хост B и демонстрация не заявлены.

## Следующий обязательный срез

1. Выделить сервис execution по D-22: `runs/chains/steps`, общий transactional budget, events/outbox → Kafka/inbox и SSE с durable cursor; fake LLM/Runtime с отменой для проверок Harness.
2. Отдельный trusted Runtime Manager через Unix socket: rootless engine, scoped grants, workspace, allowlist образов, limits и реальная изоляция. API по-прежнему без Docker socket.
3. Реальный LLM adapter/capability probes и MCP документации; policy, approval TTL 180 секунд, stop/fencing/recovery.
4. Links/hooks, отдельный reviewer, immutable Result; FCM/device tokens, пакеты staging/activation; затем Flutter Android и полные реальные TC/AC.

Рабочий foundation-стек развёрнут с реальной Supabase и доступен по HTTPS через Tailscale Funnel; owner signup/email ещё не проверен. Локальная PostgreSQL используется только для тестов, согласно ТЗ. `/readyz` остаётся 503 до появления готового исполнения; `/v1/status` показывает отдельно готовность конфигурационных операций.
