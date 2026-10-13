# Portway — план реализации

Актуально: **13.10.2026, Asia/Almaty**. Рабочая очередь разработки; исходные BL/FR/NFR и приоритеты сохраняются. [Backlog](01-product-backlog.md) задаёт объём, [SRS](03-software-requirements-specification.md) — поведение, [статус](implementation-status.md) и [протоколы](#evidence) — факты. План не переносит дату защиты и не обещает готовность всего MVP к ней. Окна 08–11.10 — история планирования, не будущий график; новый срок владельцем не задан.

## Текущий приоритет владельца: native Codex с телефона

13.10 владелец поручил первый реальный prompt/answer через установленный Codex на ноутбуке. **IP-C1** имеет приоритет перед дальнейшей подготовкой free-model TokenCounter. Контракт [DOC-036](36-api-events-and-messages.md#codex-demo)/[DOC-061](61-android-screens-flows-and-api.md#codex-demo-client), OpenAPI 0.7.0; readonly, отдельный `/v1/codex/*`, существующий ChatGPT login. Общие IP-02–08 и требования сохраняются.

| Подпункт | Статус | Критерий / следующий шаг |
|---|---|---|
| IP-C1a: очередь/API/история/stop | LOCAL_TESTED | SQL 009, encrypted prompt/events/result, current owner/device, idempotency, lease expiry без retry, cursor polling; PostgreSQL/native-process fixture PASS |
| IP-C1b: native CLI worker | LOCAL_TESTED | Реальный установленный Codex 0.159.3 с ChatGPT login; полный API → queue → native process → persisted answer → API PASS, process cancellation/environment/parser unit checks |
| IP-C1c: проверка поставки | DEPLOYED | Финальные Docker/race/contract/Kafka/build и API image smoke PASS; миграции 001–009, новые сервисы и native bridge работают на текущей установке. Live TLS/checksums/roles/HTTPS guards/worker readiness PASS |
| IP-C1d: Flutter/телефон | IN_PROGRESS | Первый клиент, debug APK и 21 локальный тест готовы ([evidence](evidence/flutter-client-2026-10-13.md)). Далее login → connected → prompt → progress/answer → stop на физическом Android; телефонная приёмка NOT_RUN |

Профиль не является обещанием существующего Desktop chat resume, записи файлов, rootless Runtime, token-budget Harness, reviewer, FCM или двух portable моделей. После минимального телефонного сценария следующая инженерная задача — ограниченный write profile с отдельными approvals/Runtime, если владелец выбирает этот путь; общая очередь ниже остаётся обязательным планом полного MVP.

## Картина прогресса

| Область | Реализовано и проверено | Что мешает полному сценарию |
|---|---|---|
| Foundation, identity, конфигурация | API, роли/версии, AES-GCM, DB-права, Docker/CI, Supabase, HTTPS Funnel | Реальные signup/email и телефон ещё не приняты |
| Execution, согласования, события | Run/chain, бюджеты, CAS/TTL, stop epochs, Kafka/outbox, SSE/replay/retention | Реальные Runtime/dispatch/revocation и измерения |
| Harness и Result | Цикл на тестовых адаптерах, альтернативы, атомарный результат, защищённое чтение | Реальный модельный шлюз, workspace revision/diff и reviewer |
| Оркестрация и перенос | Проектные контракты | Links/hooks, дети, пакеты, второй хост/модель |
| Android, FCM, итоговая приёмка | Требования и потоки | Клиент, device/push lifecycle, реальные прогоны и учебные материалы |

Публичный `POST /v1/runs` закрыт (`EXECUTION_NOT_READY`). Наличие внутреннего Harness не означает запуск реальных агентов. 13.10 подтверждено обновление текущей установки до миграций 001–009, новых API/execution/relay/audit и native Codex bridge; live checks PASS. Общий admission остаётся закрыт; реальный generic Harness/Runtime и удалённый CI этим не приняты.

## Ближайшая очередь

Статусы: `DEPLOYED` — поставлен именно указанный срез с live checks, не приёмка всего MVP; `NEXT` — следующий; `IN_PROGRESS` — выполняется; `LOCAL_TESTED` — код проверен локально, ещё не production/приёмка; `TODO` — ожидает зависимостей. Один этап можно делить на отмеченные подпункты; незаконченный этап не переводится в DONE по одному fixture-тесту.

| ID / порядок | Статус | Конкретный результат и критерий завершения | Зависимости / трассировка |
|---|---|---|---|
| IP-01. Протокольный адаптер LLM | LOCAL_TESTED | Один HTTP Chat Completions adapter: bounded SSE, сборка tool arguments, finish/usage/errors, cancellation, отсутствие retry/redirect. Подключение к Harness через интерфейс; HTTP fixture проверяет полный цикл и обрыв. Реальные capabilities не заявлять | BL-005/021/023/031; FR-012/014/015/027; TC-005/007/028 |
| IP-02. Проверки выбранных моделей и учёт токенов | IN_PROGRESS | Выбрать доступные M1/M2; привязать report к endpoint/model/config revision; реализовать private gateway contract без передачи control secret в execution; доказать подсчёт/верхнюю границу **полного** prompt, tool/result, stream, usage/error. Unknown/устаревший report блокирует dispatch. Реальная локальная модель обязательна | IP-01; BL-020/021/023/025; FR-011–016; доступные endpoints/ресурсы |
| IP-03. Runtime Manager и workspace | TODO | Узкий Unix API, rootless engine, журнал dispatch, scoped grants, epoch/expiry, prepare/read/write/shell/cancel/query. Negative tests: mount/socket/privilege/network; повторы не запускают второй процесс, restart останавливает остатки | BL-040/041/042/044/045/072; DOC-040 §6; IP-01 можно завершать до Runtime |
| IP-04. Первый публичный реальный run | TODO | Admission pins config/provider/report/revision; актуальный owner/device и restrictive policy перед внешним dispatch; worker вызывает Gateway/Runtime, `model.delta` публикуется как partial, stop отменяет работу. Только после этого открыть POST runs и operations.run. Run → approval → tool → final через HTTPS, реальные stop trials | IP-02/03; BL-030/033/034/036/050–054/065; FR-018–027/035–042 |
| IP-05. Coder → отдельный reviewer | TODO | Trusted result revision/diff/manifest, links/hooks/outbox/inbox, один read-only reviewer, hops/depth/chain budget, verdict без автоисправления coder. Дубликаты/stop/restart проверены | IP-04; BL-060–067/069; FR-043–055/059/070 |
| IP-06. FCM и Android вертикальный сценарий | IN_PROGRESS | Первый Flutter/Codex срез и debug APK готовы локально ([evidence](evidence/flutter-client-2026-10-13.md)); следующий шаг — телефонная проверка auth/Codex, затем обязательные общий run/SSE/approval/result и FCM.  API push token/link/revoke, delivery worker, APK login/profile/run/SSE/approval/result; background/terminated push на реальном телефоне. Начать клиент и FCM setup до завершения IP-05 при наличии доступов | BL-008/009/053/110–118/120–125; существующий API + Firebase/Android |
| IP-07. Пакеты и перенос | TODO | ZIP export/import, staging/mapping/activation, secret references, лимиты/отрицательные проверки, A/M1→A/M2→B/M2 с телефона | IP-02/04/06; BL-080–084; DOC-038 |
| IP-08. Приёмка и поставка | TODO | Audit/query/logs, реальные MCP/FCM/Runtime/две модели/host B, измерения, закреплённый candidate, CI и APK-прогоны, CW-001–004 | IP-04–07; BL-103/104/119; DOC-057/014 |

IP-06 не является разрешением отложить обязательный Android до конца проекта; ближайший вертикальный сценарий сейчас IP-C1, затем продолжается общий MVP. S/C: GitHub App/PR, Runner, schedules и дополнительные dashboards не начаты и не включаются в эту очередь автоматически.

### Закрытый локальный чеклист IP-01

- [x] Нормализованный Gateway contract и Harness с budget reservation до вызова.
- [x] Построение Chat Completions запроса из pinned snapshot и tool observations.
- [x] HTTP adapter с exact endpoint allowlist, no redirect/proxy/retry и timeout.
- [x] Bounded SSE, complete tool arguments, `[DONE]`, usage и safe errors.
- [x] Race/unit и HTTP integration fixture, включая пропавший usage/обрыв/отмену.
- [x] Актуальные SRS/контракт/статус и протокол проверок.

### Текущий этап IP-02

| Подпункт | Статус | Остаток |
|---|---|---|
| IP-02a: bounded probe и report lifecycle | LOCAL_TESTED | Реальный endpoint ещё не проверен; код/fixtures/PostgreSQL/Kafka проверены по [протоколу](evidence/provider-capabilities-2026-10-13.txt) |
| IP-02b: private Gateway authorization contract | LOCAL_TESTED | Count receipt + текущий Run/turn/reservation по HTTP, provider-version pin, current agent/config/report, live authority/cancellation, encrypted replay journal. Production worker ещё не подключён; real fencing/Runtime acceptance впереди |
| IP-02c: verified full-prompt TokenCounter | NEXT | Выбранные tokenizer/chat template либо доказанная верхняя граница; без heuristic fallback |
| IP-02d: реальные M1/M2 | IN_PROGRESS | Cloud OpenRouter key настроен; проверен живой zero-price каталог. LFM дала корректный tool call в одной пробе, но полный roundtrip пока не принят; free endpoints дают unavailable/rate limits/нестабильный tool. Локальная M1 и второй accepted model ещё требуются |

`WAITING_MODEL` относится к зависимым подпунктам; IP-02b и Runtime можно готовить без загрузки модели. Общий IP-02 остаётся незавершённым.

- [ ] Определить доступные M1/M2 и их ресурсы, выполнить ограниченные реальные capability-пробы.
- [ ] Реализовать подсчёт/доказуемую верхнюю границу prompt для выбранного tokenizer/chat template.
- [x] Сохранять report endpoint/model/config revision и инвалидировать при изменениях; ограниченная owner-triggered protocol probe, 30 мин TTL, durable acknowledgement/replay, config/device guards и CAS. Локальные fixtures; реальные модели ещё не проверены.
- [x] Реализовать private gateway contract и проверить ownership секретов/актуальные права; локальная HTTP/DB интеграция, encrypted cached response, stop/replay/stale pin. Не считать production worker/Runtime доказанными.
- [ ] Зафиксировать real-протокол отдельно от fixture; admission до IP-04 закрыт.

Следующий общий подпункт после приоритетного IP-C1 — **IP-02c: verified TokenCounter для выбранного free model/template**, затем **IP-03 Runtime и IP-04 admission/worker**. 13.10 владелец разрешил cloud-подключение с env-ключом, только бесплатные модели. OpenRouter key уже настроен; free-only transport и private Gateway реализованы; полный Docker/race/PG/Kafka/contract/build/smoke PASS, OpenRouter usage после проб 0 USD. Полная реальная совместимость ещё не подтверждена, локальная M1/два accepted model остаются требованиями. Расширять доступ к paid-моделям без нового указания нельзя.

### Условия, которые нужно установить для следующих этапов

- Локальной модели нет, weights не загружались. Позднее 13.10 владелец поручил подключить cloud Free: OpenRouter key добавлен, живой каталог/ограниченные probes выполнены, полный accepted model ещё отсутствует. Не подставлять фиктивные capabilities или оценку «символы / 4» вместо доказанного token bound.
- Runtime: API не получает Docker socket; доступ разработчика к Docker для тестов не доказывает rootless runtime isolation.
- Firebase/Android и второй хост: потребуются реальные настройки/устройство. До зависимой работы продолжать независимые серверные части.
- В текущем этапе **платные probes/модели запрещены владельцем**. Реальные бесплатные probes ограничены двумя fixed-prompt генерациями; no retry/fallback. Локальные fixtures не расходуют cloud quota.

## Правило обновления после каждого среза

1. Отметить подпункты и реальный остаток текущего IP, выбрать следующий NEXT.
2. Обновить фактический статус и BL-status; требования и приоритеты менять только при изменении согласованного поведения.
3. Приложить команды/результаты в evidence, различая unit/fixture, real integration и production deploy. Не объявлять TC/AC целиком PASS по частичному покрытию.
4. При изменении API/DB/events обновить OpenAPI/миграции/контракты в той же работе. Проверить ссылки и сохранение ID/приоритетов.
5. В отчёте пользователю: что стало доступно, проверки, ограничение, следующий конкретный пункт. Не использовать условный процент готовности без проверяемого знаменателя.

<a id="evidence"></a>
## Завершённые инженерные срезы

- [Native Codex bridge, 13.10](evidence/codex-bridge-2026-10-13.txt): реальный локальный Codex roundtrip, negative/race/PG/Kafka/smoke, live deployment + heartbeat; Android acceptance NOT_RUN.

- [Foundation / identity / config](evidence/backend-foundation-2026-10-11.txt).
- [Kafka и service isolation](evidence/kafka-audit-2026-10-11.txt), [реальное развёртывание](evidence/live-deployment-2026-10-11.txt).
- [Execution foundation](evidence/execution-foundation-2026-10-12.txt), [Funnel](evidence/docker-funnel-2026-10-12.txt).
- [SSE / retention / approvals](evidence/execution-sse-2026-10-12.txt).
- [Harness / durable Result](evidence/harness-2026-10-12.txt).

- [IP-01: потоковый Gateway](evidence/model-gateway-2026-10-12.txt) — локальный Docker/race/HTTP fixture PASS; реальные модели — IP-02.

- [IP-02a: provider capability lifecycle](evidence/provider-capabilities-2026-10-13.txt) — Docker/race/HTTP fixtures/PostgreSQL/Kafka и API image smoke PASS; реальные модели отсутствуют.

- [Free OpenRouter + private Gateway](evidence/free-model-gateway-2026-10-13.txt): env/provision/catalog, count/reservation RPC, encrypted dispatch journal; реальные free-пробы отдельно от fixtures.
