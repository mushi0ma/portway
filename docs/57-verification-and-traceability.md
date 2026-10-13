# Portway — проверки и трассировка

> **Изменение 11.10.2026 (D-22):** по решению владельца целевая архитектура — Go-микросервисы и Kafka. [Принятое изменение архитектуры](21-microservices-kafka-amendment.md) заменяет прежний монолит/DB-only шину; PostgreSQL outbox сохраняется для надёжной публикации. Старые схемы одного процесса ниже — baseline до изменения, не текущая целевая топология. Первый реализованный срез Kafka описан в [DOC-022](22-kafka-first-slice.md); полный переход ещё не завершён.


## Фактические проверки, 13.10.2026

Native Codex demo проверяет отдельное подмножество запуска/истории/stop/recovery: [BL-030][id-bl-030]/033/035/036, [FR-018][id-fr-018]/021/024 остаются PARTIAL; [TC-007][id-tc-007]/008/009/030 не переводятся в полный PASS. [Реальный native roundtrip и fixture negative checks](evidence/codex-bridge-2026-10-13.txt) не доказывают rootless Runtime, reviewer, FCM или Android acceptance. CLI login — существующий ChatGPT аккаунт, API keys не использованы.


`PARTIAL` в RTM означает подтверждённое инженерное подмножество, **не PASS полного TC/AC**. `NOT_RUN` означает отсутствие полного прогона проектного сценария; отдельные unit/fixture тесты перечислены ниже. Состав 32 TC, 20 AC и требования сохраняются.

| Проверенное подмножество | Свидетельства | Остаток для приёмки |
|---|---|---|
| Foundation, identity/config, DB isolation | [Foundation](evidence/backend-foundation-2026-10-11.txt), [live](evidence/live-deployment-2026-10-11.txt) | Полный signup/email, Android, candidate CI |
| Kafka outbox/inbox и audit | [Kafka](evidence/kafka-audit-2026-10-11.txt), [execution](evidence/execution-foundation-2026-10-12.txt) | Agent routing, hooks, downstream consumers |
| SSE/retention, approvals TTL/CAS | [SSE/approvals](evidence/execution-sse-2026-10-12.txt) | FCM/Android, реальные grants, p95 |
| Harness, budgets, alternatives, durable result | [Harness](evidence/harness-2026-10-12.txt) | Реальные LLM/Runtime, токенизация, workspace artifacts/reviewer |
| Chat Completions HTTP/SSE adapter, tool roundtrip, usage/errors | [Gateway](evidence/model-gateway-2026-10-12.txt) | Две реальные модели, tokenizer, capabilities, production wiring |

| Provider probe/report lifecycle, replay, TTL, config/device invalidation | [Capability lifecycle](evidence/provider-capabilities-2026-10-13.txt) | Реальные модели, tokenizer, private dispatch/Runtime; models list отсутствует |

| Free-only cloud routing/env provision, private RPC/Harness/dispatch journal | [Free Gateway](evidence/free-model-gateway-2026-10-13.txt) | Full accepted real tool/result probe, verified tokenizer, Runtime/worker/admission; actual local model/host transfer |

Окончательные TC/AC не переводятся в PASS этой таблицей. [IP-01–08](implementation-plan.md) описывают, какие зависимости ещё закрыть.

<a id="doc-057"></a>

| Поле | Значение |
|---|---|
| Артефакт | [DOC-057][id-doc-057]; единые [RTM][g-traceability], тест-план, [ПМИ][g-traceability] и форма протокола |
| Версия и дата | v0.5 · 13.10.2026 |
| Статус | Частичные backend-проверки выполнены; полные TC/AC ещё не приняты |
| Источник приёмки | [20 AC](14-demo-acceptance-criteria.md); критерии здесь не заменяются |
| Основания | [SRS](03-software-requirements-specification.md), [UC](12-use-case-diagrams-and-specifications.md), [QS](13-quality-attribute-scenarios-and-utility-tree.md), [SMART/MX](04-smart-goals-and-metrics.md#methods) |
| Контракты | [Архитектура](20-architecture-and-decisions.md), [данные](29-data-and-state-model.md), [API](36-api-events-and-messages.md), [безопасность](40-agent-execution-and-security.md), [пакет](38-agent-package-format.md), [Android](61-android-screens-flows-and-api.md), [операции](52-build-deploy-and-operations.md) |

<a id="levels"></a>
## 1. Уровни, gates и ответственность

Обязательная база — 77 [FR][g-requirement-ids] (включая только M-части [FR-059][id-fr-059]/070), все 24 [NFR][g-requirement-ids], 20 [UC][g-use-case] и 21 [QS][g-quality-scenario]. 13 чистых S/C [FR][g-requirement-ids] не объявляются выполненными и не требуют реализации для базы. Матрица ниже содержит все 90 [FR][g-requirement-ids] и 24 [NFR][g-requirement-ids], включая явный OUT_OF_BASE. 79 [BL][g-work-ids] и 4 [CW][g-work-ids] сохраняют приоритеты [backlog][g-work-ids]/[MoSCoW][g-priorities].

Уровни: C — contract/unit на детерминированных данных; I — integration с настоящим disposable [PostgreSQL][g-supabase] и [fake][g-fixture] внешними ports; R — настоящие [Runtime][g-runtime]/[LLM][g-model]/[MCP][g-mcp]/[HTTPS][g-https]/host; A — физический [Android][g-flutter]/[FCM][g-push]/мобильный перенос/репетиция [владельца][g-owner]. Один [TC][g-acceptance] может иметь несколько уровней; PASS его [fake][g-fixture] части не закрывает R/A. Исполнитель кода готовит/запускает [CI][g-ci-cd] и протоколы, [владелец][g-owner] участвует в реальных устройствах/аккаунтах и объясняет курсовую. Документ не назначает второго человека.

Порядок gates: G1 сборка и isolated rules → G2 durable [state][g-uml]/concurrency/security → G3 реальные adapters и телефон → G4 три ячейки переноса/количественные выборки → G5 один закреплённый candidate, два [CI][g-ci-cd] и два полных Android-прогона, курсовые материалы. Blocker обязательного требования не заменяется планом исправления или [fake][g-fixture] [свидетельством][g-evidence]. S/C включаются только по существующему допуску [MoSCoW][g-priorities]; новые с 11.10 не начинаются.

<a id="fixtures"></a>
## 2. [Фикстуры][g-fixture] и доказательства

| [Фикстура][g-fixture] | Воспроизводимый контроль |
|---|---|
| Repo F-REPO | Небольшой repo с pinned commit и failing assertion для известной [задачи][g-task]; заранее зафиксирован expected functional result, не совпадение текста [LLM][g-model] |
| F-LLM | Scripted delta/[tool][g-tool]/final/error/unknown [usage][g-usage], управляемые задержки; не выдаётся за M1/M2 |
| F-RUNTIME | Детерминированный port [fake][g-fixture] + отдельный real managed-engine test; marker files для prohibited effects |
| F-MCP / F-PUSH | [Fake][g-fixture] [timeout][g-timeout]/error/token rotation для [CI][g-ci-cd]; отдельные настоящие [MCP][g-mcp]/[FCM][g-push] trials |
| F-DB / CLOCK | Disposable [PostgreSQL][g-supabase], crash points/controlled clock для7/30дней; production clock нельзя менять через публичный [API][g-api] |
| F-PACKAGE | Valid [ZIP][g-package], corrupted hashes, hostile entries и точные граничные размеры/число файлов |
| F-IDENTITY | [Owner][g-owner]/other subject, два device, подтверждённый/неподтверждённый email, revoked proof |
| F-SECRETS | Уникальные безопасные marker values для каждого класса credential; отсутствие настоящих ключей в отчёте |

[Evidence][g-evidence] каждого [запуска][g-run]: candidate/config fingerprint, [TC][g-acceptance]/[AC][g-acceptance]/[FR][g-requirement-ids] IDs, уровень, [UTC][g-identifier-formats]/часовой offset, actor, предусловия, последовательность действий, expected/actual, raw report/artifact refs+hashes, FAIL/defect IDs. [Лог][g-audit] опасного эффекта проверяется независимым marker/process/result assertion, а не только возвращаемым [state][g-uml]. Секретные values, [auth][g-auth] bodies и callback token_hash в отчёты не копируются.

<a id="cases"></a>
## 3. Значимые [тест-кейсы][g-acceptance]

В каждой строке сначала устроить указанные условия, затем выполнить действия и сравнить наблюдаемый исход. [TC-001][id-tc-001]–020 задают основные прогоны; [TC-021][id-tc-021]–032 — отрицательные/конкурентные случаи. Все строки пока NOT_RUN.

| [TC][g-acceptance] / уровень | Сценарий | Условия и действия | Наблюдаемый oracle / [evidence][g-evidence] | [AC][g-acceptance] |
|---|---|---|---|---|
| <a id="tc-001"></a>[TC-001][id-tc-001] / C/I | Поставка и [CI][g-ci-cd] | Чистый checkout, lock-файлы, disposable DB и [fake][g-fixture] ports. Собрать server/[APK][g-apk], выполнить fmt/lint/unit/contract/integration, затем контрольную ошибку в каждом обязательном gate. | Два полных CI-прогона одного revision зелёные; контрольная ошибка даёт nonzero; команды/версии/startup [evidence][g-evidence] сохранены. | [AC-001][id-ac-001] |
| <a id="tc-002"></a>[TC-002][id-tc-002] / I/R/A | Регистрация, вход и отзыв | Новая installation, [setup proof][g-device-session], другой [Supabase][g-supabase] subject и два device. Пройти email confirmation/callback/вход; отозвать один device при открытом [SSE][g-sse]; попробовать чужой [JWT][g-jwt-jwks]. | Один [owner][g-owner]; без setup/confirmation нет bind; чужой subject запрещён; revoked HTTP запрещён, [SSE][g-sse] закрыт ≤5 с, [push][g-push] unlink. | [AC-002][id-ac-002] |
| <a id="tc-003"></a>[TC-003][id-tc-003] / A | Полный учебный Android-путь | Установить candidate [APK][g-apk] на [Android][g-flutter] 16; Splash→вход→главная→[агенты][g-agent]→[задача][g-task]→[апрув][g-approval]→[пакеты][g-package]→профиль→выход; повторить с error/empty и сменой [HTTPS][g-https] URL. | Все основные действия доступны; name/email и ошибки читаемы, stale session не даёт действий; КМ07-02–10 имеют [evidence][g-evidence]. | [AC-003][id-ac-003] |
| <a id="tc-004"></a>[TC-004][id-tc-004] / I/R | Версии, [роли][g-role] и права | Создать [coder][g-main-roles] по main preset и code_reader по subagent; изменить инструкции в новой версии; начать старую версию и ужесточить server [policy][g-policy]. | Старая config/hash неизменна, [run][g-run] pinned; каталог 5+3; текущий deny действует, read-only [роль][g-role] не получает write. | [AC-004][id-ac-004] |
| <a id="tc-005"></a>[TC-005][id-tc-005] / C/R | Два реальных [model][g-model] adapter | Для M1 и M2 выполнить [stream][g-model-protocol], [tool][g-tool] proposal/result, finish, [usage][g-usage]/estimate и controlled error; хотя бы M1 локальная. | Capability report привязан к [model][g-model]/config; [Harness][g-harness] получает единый DTO; недостающая capability блокирует [шаг][g-step]; traces без [секретов][g-secret]. | [AC-005][id-ac-005] |
| <a id="tc-006"></a>[TC-006][id-tc-006] / R | Настоящий [Runtime][g-runtime] и [MCP][g-mcp] | Два managed [workspace][g-workspace]; pinned repo, allowlisted image и документационный [MCP][g-mcp]. Выполнить read/write разрешённой [роли][g-role], [tool][g-tool] result, cleanup; отдельно запрещённые image/[hooks][g-hook]/путь/socket. | Настоящая изоляция/лимиты/egress, другой [workspace][g-workspace] не читается; real docs result; запреты до исполнения; [fake][g-fixture] не зачтён. | [AC-006][id-ac-006] |
| <a id="tc-007"></a>[TC-007][id-tc-007] / I/R/A | Управляемый [run][g-run] и request key | POST одной [задачи][g-task] дважды/конкурентно с тем же key, затем с изменённым body; провести [model][g-model]→[tool][g-tool]→result, controlled [provider][g-provider] failure. | Один [run][g-run], conflict для другого body; [state][g-uml]/result/error доступны; partial [stream][g-model-protocol] не выдан за final; context IDs сохранены. | [AC-007][id-ac-007] |
| <a id="tc-008"></a>[TC-008][id-tc-008] / I/A | [SSE][g-sse]/replay и порядок | Наблюдать [run][g-run], оборвать сеть и reconnect по cursor; отправить duplicate [event][g-event]; проверить watermark-expired и foreign/future cursor. | Per-run seq без потерь/дублированного отображения; корректные 400/409/410; snapshot+cursor согласованы, [UI][g-ui] сообщает пропуск. | [AC-008][id-ac-008] |
| <a id="tc-009"></a>[TC-009][id-tc-009] / I/R/A | [Stop][g-cancelled] [run][g-run] и [chain][g-chain] | Два активных [run][g-run], [tool][g-tool]/[model][g-model]/[approval][g-approval] и queued [hook][g-hook]; принять [stop][g-cancelled] [chain][g-chain], отслеживать процессы/dispatch, затем проверить через 10 с. | Новых выдач/детей нет; controlled работа прекратилась; pending invalidated; unknown отдельно; ≤5 с проверяет [TC-019][id-tc-019]. | [AC-009][id-ac-009] |
| <a id="tc-010"></a>[TC-010][id-tc-010] / I/R | [Бюджеты][g-budget] и время | Конкурентно резервировать последний остаток, пройти ровно лимит/превышение [шагов][g-step], time, output cap; оборвать [usage][g-usage] response; reply/delegation на границах counters. | Нет двойного reserve; defaults 15/30 мин,20/40,20k/40k,2048,120/60 с,8/2 соблюдены; uncertain не равен нулю; альтернативы тоже остановлены. | [AC-010][id-ac-010] |
| <a id="tc-011"></a>[TC-011][id-tc-011] / I/R/A | [Апрув][g-approval] и альтернативы | Shell marker: до approve не должен появиться. Approve неизменных args, reject/expiry, затем одна разрешённая и две отказанные альтернативы; race уточняет [TC-021][id-tc-021]. | Одно действие только по актуальному [grant][g-fencing]; исходное denied не исполнено; safe alternative заново проверена, после двух отказов explained terminal. | [AC-011][id-ac-011] |
| <a id="tc-012"></a>[TC-012][id-tc-012] / A/R | [FCM][g-push] на физическом [Android][g-flutter] | Три [background][g-app-lifecycle] и три normal [terminated][g-app-lifecycle] trials с permission; каждый pending [approval][g-approval] породить на сервере и tap notification; [force-stop][g-app-lifecycle] вынести отдельно. | 6/6 получены до [TTL][g-timeout] 180 с; reauth+GET latest; expired tap не исполняет; timestamps/lifecycle/phone [state][g-uml] записаны, accepted не равен delivered. | [AC-012][id-ac-012] |
| <a id="tc-013"></a>[TC-013][id-tc-013] / C/I/R/A | Топология и [сообщения][g-message] | Создать link/[hook][g-hook] с телефона; доставить допустимые [event][g-event]/envelope и active message на границе [шага][g-step]; duplicate/conflict/disable, неподдержанная schema/type. | Направление/права/counters/≤64 KiB проверены; один logical effect; messages/[hooks][g-hook] видны; текущий [tool][g-tool] args не изменился. | [AC-013][id-ac-013] |
| <a id="tc-014"></a>[TC-014][id-tc-014] / I/R/A | Отдельный [reviewer][g-main-roles] | [Coder][g-main-roles] меняет pinned [repository][g-repository], completion [hook][g-hook] создаёт [reviewer][g-main-roles]; повтор source [event][g-event], положительный/отрицательный verdict; terminal reply. | Ровно один новый [reviewer][g-main-roles] ID и та же [chain][g-chain], read-only fixed diff; result/verdict на [APK][g-apk]; negative не создаёт auto-fix/PR. | [AC-014][id-ac-014] |
| <a id="tc-015"></a>[TC-015][id-tc-015] / C/I/A | Мобильный [ZIP][g-package] | [Экспорт][g-export-import] выбранной версии системным picker; [импорт][g-export-import], preview, [mappings][g-staging] и [activation][g-staging]; отменить picker/неполный [mapping][g-staging]; повреждения в [TC-026][id-tc-026]. | [ZIP][g-package] без [секретов][g-secret]/local IDs, portable config сохранена; activate атомарна/идемпотентна; [unresolved][g-staging] блокирует; реальный [UI][g-ui], не API-only. | [AC-015][id-ac-015] |
| <a id="tc-016"></a>[TC-016][id-tc-016] / R/A | Три ячейки переноса | Одна [задача][g-task]/commit: A/M1→[ZIP][g-package] с телефона→A/M2→[ZIP][g-package] с телефона→B/M2, B отдельная installation/host или VM. | 3/3 functional assertions; portable fields/hash совпадают; target bindings новые; [model][g-model] IDs,host/[run][g-run]/revision/config [evidence][g-evidence] записаны. | [AC-016][id-ac-016] |
| <a id="tc-017"></a>[TC-017][id-tc-017] / I/R/A | [Секреты][g-secret], audit/status | Уникальные marker secrets [LLM][g-model]/DB/[FCM][g-push]/App, controlled DB/[model][g-model]/[push][g-push] fault; прочитать protected audit/log/status; storage/[API][g-api]/[APK][g-apk]/[ZIP][g-package]/[tool][g-tool] env scan. | Шифротекст+отдельный key; zero marker leaks; audit actor/object/time/outcome/context; dependency cause различима; [Prometheus][g-observability-tools] не требуется. | [AC-017][id-ac-017] |
| <a id="tc-018"></a>[TC-018][id-tc-018] / I/R | Restart/[reconciliation][g-recovery] | Остановить [API][g-api] при active [tool][g-tool]/awaiting [approval][g-approval]; сохранить Manager [state][g-uml]; поднять сервер, попытаться late approve, создать ручной новый [run][g-run]. | До [ready][g-readiness] старые [grants][g-fencing] invalid; old [run][g-run] failed/interrupted, new ID; unknown/cleanup видны; target ≤10 с при ≤2 [run][g-run], no auto-resume. | [AC-018][id-ac-018] |
| <a id="tc-019"></a>[TC-019][id-tc-019] / R/A | Количественные ориентиры | [PERF-01][g-smart]: prepared host, 2 active [run][g-run]/1 [APK][g-apk], real [HTTPS][g-https],10 [events][g-event]/s/[chain][g-chain]; 3×≥1000 [событий][g-event]; 10 [stop][g-cancelled] trials; 3 prepared starts. | Каждый [p95][g-evidence]≤2 с, каждый [stop][g-cancelled]≤5 с/no resurrection через10 с, каждый старт≤600 с; raw samples/clock error сохранены; [FCM][g-push]/перенос отдельно. | [AC-019][id-ac-019] |
| <a id="tc-020"></a>[TC-020][id-tc-020] / A | Кандидат и курсовые материалы | Один candidate revision/config: два [CI][g-ci-cd] и два полных [Android][g-flutter] сценария с [push][g-push]/переносом; проверить введение, теорию, источники, КМ07-01–14 и репетицию. | [CW-001][id-cw-001]–004 имеют фактические материалы/объяснение [владельца][g-owner]; hashes/протоколы/дефекты сохранены; design docs не заменяют главу. | [AC-020][id-ac-020] |
| <a id="tc-021"></a>[TC-021][id-tc-021] / I/R | Гонки [approval][g-approval] / injection | На одном pending выполнять approve/reject/expiry/[stop][g-cancelled] конкурентно, менять args/hash; в prompt/[tool][g-tool]/[MCP][g-mcp]/message просить игнорировать [policy][g-policy] и дать write [reviewer][g-main-roles]. | Один terminal decision и максимум один dispatch; stale/altered/expired [grant][g-fencing] не действует; запрещённый marker ни разу не создан. | [AC-004][id-ac-004], [AC-011][id-ac-011] |
| <a id="tc-022"></a>[TC-022][id-tc-022] / I/A | Device revoke и offline logout | Открыть HTTP/[SSE][g-sse] на device1, отозвать с device2; refresh старым proof; offline logout и последующее возвращение сети; другой subject. | Новый доступ запрещён, [SSE][g-sse]≤5 с; local clean сразу, remote revocation не помечена успешной без ACK; tombstone не даёт вход. | [AC-002][id-ac-002], [AC-003][id-ac-003], [AC-017][id-ac-017] |
| <a id="tc-023"></a>[TC-023][id-tc-023] / R | [Sandbox][g-sandbox] escape и egress | Настоящий container: попытки host socket/path/device/privileged/lifecycle [hook][g-hook], другой volume, прямая межsandbox/[LLM][g-model]/произвольная сеть, [MCP][g-mcp] injection. | Все hard denies, нет marker на host/чужом [workspace][g-workspace]; CPU/RAM/time измерены; read-only [reviewer][g-main-roles] и scoped adapter сохранены. | [AC-006][id-ac-006], [AC-011][id-ac-011] |
| <a id="tc-024"></a>[TC-024][id-tc-024] / I | Crash points [outbox][g-outbox]/inbox | Инъекция падения до commit/после commit/после effect-before-ACK; [redelivery][g-retry] и тот же ID с иным payload; disabled link, target ambiguity, hops9/depth3. | Source [state][g-uml] согласован с intent; один logical [review][g-review]/message effect; conflict/deny/[DLQ][g-retry] наблюдаемы; [retry][g-retry]≤24ч,dedupe7дней. | [AC-010][id-ac-010], [AC-013][id-ac-013] |
| <a id="tc-025"></a>[TC-025][id-tc-025] / I/R | Pinned [review][g-review] и terminal reply | Изменить живой [coder][g-main-roles] [workspace][g-workspace] после [Result][g-result] snapshot; [reviewer][g-main-roles] читает [Result][g-result]; прислать reply/negative verdict после terminal, повторить [hook][g-hook]. | [Reviewer][g-main-roles] видит закреплённый diff/hash; нет повторного [review][g-review]/возобновления [coder][g-main-roles]; ручное исправление создаёт новый [run][g-run]. | [AC-014][id-ac-014], [AC-018][id-ac-018] |
| <a id="tc-026"></a>[TC-026][id-tc-026] / C/I | Враждебный [пакет][g-package] и границы | Traversal/absolute/backslash/NUL/symlink/duplicate/case-Unicode collision, [ZIP][g-package] bomb с ложным header, JSON duplicate keys, bad hash/schema, 10/25MiB и1000±1; missing [mapping][g-staging]/[role][g-role]. | Никакой записи вне [staging][g-staging]/исполнения/частичной [activation][g-staging]; точная граница допустима, превышение отвергнуто; portable поля не теряются молча. | [AC-015][id-ac-015], [AC-016][id-ac-016] |
| <a id="tc-027"></a>[TC-027][id-tc-027] / I/A | Replay retention и пропуск | Проверить последний retained cursor, cursor на watermark/ниже/впереди, seq gap и duplicate после reconnect; concurrent commits одному [run][g-run]. | Не переставлены seq, 410 до начала [stream][g-model-protocol]; snapshot не изображает полную историю; expired token/guard закрывает соединение. | [AC-008][id-ac-008], [AC-019][id-ac-019] |
| <a id="tc-028"></a>[TC-028][id-tc-028] / I | Reserve/[stop][g-cancelled]/settle concurrency | Одновременно два запроса на последний [chain][g-chain] budget, [stop][g-cancelled] между reserve/dispatch, duplicate settlement, обрыв с unknown [usage][g-usage] и длительное [approval][g-approval]. | Один допустимый reserve, нет double spend/[release][g-ci-cd]; после [stop][g-cancelled] нет новых admissions; ожидание входит в wall-clock; unknown платный вызов блокирован. | [AC-009][id-ac-009], [AC-010][id-ac-010], [AC-019][id-ac-019] |
| <a id="tc-029"></a>[TC-029][id-tc-029] / I/R/A | Secret и public schema scan | Marker secrets во всех credential stores; прочитать anonymous Data [API][g-api], чужой installation, [API][g-api] errors/logs/audit/[ZIP][g-package]/[APK][g-apk]/[tool][g-tool] env; callback access log. | Closed business schema и [owner][g-owner]/device guard; zero known marker values во всех запрещённых поверхностях; ограничения scanner отмечены. | [AC-002][id-ac-002], [AC-017][id-ac-017] |
| <a id="tc-030"></a>[TC-030][id-tc-030] / I/R | Unknown external dispatch | Прервать связь [API][g-api]↔Manager после admission до ACK, затем restart/lease expiry; попробовать повтор execute того же и нового ID для того же действия. | Query возвращает saved outcome либо unknown; нет слепого повторного side effect, pending old [approvals][g-approval] недействительны; audit сохраняет контекст. | [AC-007][id-ac-007], [AC-013][id-ac-013], [AC-018][id-ac-018] |
| <a id="tc-031"></a>[TC-031][id-tc-031] / I/R | [Readiness][g-readiness]/retention/restore | DB/Manager/[provider][g-provider]/[push][g-push] fault по отдельности; test-clock retention7/30дней и workspace24ч; backup+отдельный key→disposable restore. | [Health][g-readiness] не маскирует dependency block; очистка не удаляет active/[evidence][g-evidence]/[export][g-export-import]; snapshot/version/hash читаемы, decrypt работает, old [grants][g-fencing]/device inactive. | [AC-001][id-ac-001], [AC-017][id-ac-017], [AC-018][id-ac-018] |
| <a id="tc-032"></a>[TC-032][id-tc-032] / A | Мобильные края и доступность | Rotation/клавиатура/200% font/[TalkBack][g-accessibility]/light-dark, длинные args, repeat [push][g-push] в expired/revoked/[auth][g-auth] [state][g-uml], permission deny/rotation token, picker cancel/network loss. | Действия достижимы, нет double approve/[import][g-export-import], безопасный deep link, нужные args читаемы; системные состояния/ошибки объяснены. | [AC-003][id-ac-003], [AC-012][id-ac-012], [AC-015][id-ac-015] |

<a id="measurements"></a>
## 4. [ПМИ][g-traceability] количественных проверок

Числа/профиль не вводятся заново: источник [SRS][g-requirement-ids] §4.2 и [SMART][g-smart] [MX-01][id-mx-01]–06. [PERF-01][g-smart] — CachyOS/i5-13420H/16GB, Android16, 2 active [run][g-run] в одной [chain][g-chain],1 [APK][g-apk],10 [events][g-event]/s/[chain][g-chain]. Последовательный [coder][g-main-roles]→[reviewer][g-main-roles] отдельно проверяет функцию; он не подменяет испытание двух одновременно active [run][g-run]. Один [inference][g-model-protocol] endpoint может иметь очередь.

[SSE][g-sse]: три независимых прогретых trials, каждый ≥1000 committed [events][g-event] по реальному [HTTPS][g-https]. Записать server commit и [APK][g-apk] display timestamp, clock offset/погрешность; latency после коррекции, PASS только если консервативная верхняя граница [p95][g-evidence]≤2с в каждом trial. [p95][g-evidence] — nearest rank `ceil(0.95*N)` отсортированных наблюдений; lost [events][g-event] не выкидываются из denominator/отчёта, trial с потерями/неподтверждённым измерением не PASS. [LLM][g-model] generation и [FCM][g-push] вне интервала. Сырые samples, network/versions/load и расчёт сохраняются.

[Stop][g-cancelled]:10 самостоятельных [chain][g-chain] trials, отсчёт от server durable [stop][g-cancelled] до прекращения всех controlled работ; максимум каждого≤5с, ни одного нового dispatch после [stop][g-cancelled], дополнительная проверка отсутствия оживления через10с. Unknown external effects перечисляются и не выдаются за cancelled. Старт:3 раза одинаковый подготовленный стенд, каждый≤600с до реальной [readiness][g-readiness]; первоначальные загрузки/аккаунты отдельно, отсутствие обязательной зависимости→не PASS.

[Push][g-push]:3 [background][g-app-lifecycle]+3 normal [terminated][g-app-lifecycle], реальный physical [Android][g-flutter] и [FCM][g-push],6/6 до server [approval][g-approval] TTL180с; normal [terminated][g-app-lifecycle] и [force-stop][g-app-lifecycle] различаются в протоколе. Это [критерий приёмки][g-acceptance], не SLA [FCM][g-push]. Перенос:3/3 ячейки A/M1,A/M2,B/M2 с новой работой на каждой, normalized config+functional assertions; одна [model][g-model] локальная, B отдельный host/VM. Candidate-прогоны не заменяют эти специальные выборки.

<a id="rtm"></a>
## 5. Единая [RTM][g-traceability]

Указан проектный [владелец][g-owner] реализации, а не уже существующий код. [UC][g-use-case]/[QS][g-quality-scenario] взяты из канонических спецификаций, [AC][g-acceptance] — из [DOC-014][id-doc-014]; [TC][g-acceptance] — строки выше. `M/S` означает только обязательную часть в текущей базе. Для optional [FR][g-requirement-ids] статус OUT_OF_BASE, а тесты появятся при принятом включении функции. Таблица не создаёт новые обязательства.

| Требование / приоритет | Модуль и контракт | [UC][g-use-case] / [QS][g-quality-scenario] | [AC][g-acceptance] → [TC][g-acceptance] | [BL][g-work-ids] | Статус |
|---|---|---|---|---|---|
| [FR-001][id-fr-001] / M | identity/audit; [auth](36-api-events-and-messages.md#auth) | [UC-001][id-uc-001], [UC-002][id-uc-002] | [AC-002][id-ac-002] → [TC-002][id-tc-002], [TC-022][id-tc-022], [TC-029][id-tc-029] | [BL-100][id-bl-100] | PARTIAL |
| [FR-002][id-fr-002] / M | identity/audit; [auth](36-api-events-and-messages.md#auth) | [UC-001][id-uc-001], [UC-002][id-uc-002] | [AC-002][id-ac-002] → [TC-002][id-tc-002], [TC-022][id-tc-022], [TC-029][id-tc-029] | [BL-100][id-bl-100] | PARTIAL |
| [FR-003][id-fr-003] / M | identity/audit; [auth](36-api-events-and-messages.md#auth) | [UC-002][id-uc-002], [UC-004][id-uc-004] | [AC-002][id-ac-002] → [TC-002][id-tc-002], [TC-022][id-tc-022], [TC-029][id-tc-029] | [BL-100][id-bl-100] | PARTIAL |
| [FR-004][id-fr-004] / M | identity/audit; [auth](36-api-events-and-messages.md#auth) | [UC-018][id-uc-018] | [AC-017][id-ac-017] → [TC-017][id-tc-017], [TC-022][id-tc-022], [TC-029][id-tc-029], [TC-031][id-tc-031] | [BL-103][id-bl-103] | PARTIAL |
| [FR-005][id-fr-005] / M | [agents][g-agent]; [versions/roles](36-api-events-and-messages.md#agents) | [UC-005][id-uc-005] | [AC-004][id-ac-004] → [TC-004][id-tc-004], [TC-021][id-tc-021] | [BL-010][id-bl-010] | PARTIAL |
| [FR-006][id-fr-006] / M | [agents][g-agent]; [versions/roles](36-api-events-and-messages.md#agents) | [UC-005][id-uc-005] | [AC-004][id-ac-004] → [TC-004][id-tc-004], [TC-021][id-tc-021] | [BL-010][id-bl-010], [BL-110][id-bl-110] | PARTIAL |
| [FR-007][id-fr-007] / M | [agents][g-agent]; [versions/roles](36-api-events-and-messages.md#agents) | [UC-005][id-uc-005] | [AC-004][id-ac-004] → [TC-004][id-tc-004], [TC-021][id-tc-021] | [BL-011][id-bl-011], [BL-014][id-bl-014] | PARTIAL |
| [FR-008][id-fr-008] / M | [agents][g-agent]; [versions/roles](36-api-events-and-messages.md#agents) | [UC-005][id-uc-005] | [AC-004][id-ac-004] → [TC-004][id-tc-004], [TC-021][id-tc-021] | [BL-012][id-bl-012] | PARTIAL |
| [FR-009][id-fr-009] / M | [agents][g-agent]; [versions/roles](36-api-events-and-messages.md#agents) | [UC-005][id-uc-005], [UC-007][id-uc-007] | [AC-004][id-ac-004] → [TC-004][id-tc-004], [TC-021][id-tc-021] | [BL-011][id-bl-011], [BL-012][id-bl-012], [BL-030][id-bl-030] | PARTIAL |
| [FR-010][id-fr-010] / S | [agents][g-agent]; [versions/roles](36-api-events-and-messages.md#agents) | — | При включении S/C | [BL-013][id-bl-013] | OUT_OF_BASE |
| [FR-011][id-fr-011] / M | gateway/providers; [provider](36-api-events-and-messages.md#agents) | [UC-006][id-uc-006] | [AC-005][id-ac-005] → [TC-005][id-tc-005] | [BL-020][id-bl-020], [BL-025][id-bl-025] | PARTIAL |
| [FR-012][id-fr-012] / M | gateway/providers; [provider](36-api-events-and-messages.md#agents) | [UC-006][id-uc-006] | [AC-005][id-ac-005] → [TC-005][id-tc-005] | [BL-021][id-bl-021] | NOT_RUN |
| [FR-013][id-fr-013] / M | gateway/providers; [provider](36-api-events-and-messages.md#agents) | [UC-006][id-uc-006] | [AC-005][id-ac-005] → [TC-005][id-tc-005] | [BL-023][id-bl-023] | PARTIAL |
| [FR-014][id-fr-014] / M | gateway/providers; [provider](36-api-events-and-messages.md#agents) | [UC-006][id-uc-006] | [AC-005][id-ac-005] → [TC-005][id-tc-005] | [BL-023][id-bl-023] | PARTIAL |
| [FR-015][id-fr-015] / M | gateway/providers; [provider](36-api-events-and-messages.md#agents) | [UC-006][id-uc-006] | [AC-005][id-ac-005] → [TC-005][id-tc-005] | [BL-023][id-bl-023] | PARTIAL |
| [FR-016][id-fr-016] / M | gateway/providers; [provider](36-api-events-and-messages.md#agents) | [UC-006][id-uc-006], [UC-016][id-uc-016] | [AC-005][id-ac-005] → [TC-005][id-tc-005] | [BL-021][id-bl-021], [BL-023][id-bl-023], [BL-082][id-bl-082] | PARTIAL |
| [FR-017][id-fr-017] / S | gateway/providers; [provider](36-api-events-and-messages.md#agents) | [UC-006][id-uc-006] | При включении S/C | [BL-022][id-bl-022] | PARTIAL |
| [FR-018][id-fr-018] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-007][id-uc-007], [UC-020][id-uc-020] | [AC-007][id-ac-007] → [TC-007][id-tc-007], [TC-030][id-tc-030] | [BL-030][id-bl-030] | PARTIAL |
| [FR-019][id-fr-019] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-007][id-uc-007], [UC-020][id-uc-020] | [AC-007][id-ac-007] → [TC-007][id-tc-007], [TC-030][id-tc-030] | [BL-030][id-bl-030], [BL-117][id-bl-117] | PARTIAL |
| [FR-020][id-fr-020] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-007][id-uc-007] | [AC-007][id-ac-007] → [TC-007][id-tc-007], [TC-030][id-tc-030] | [BL-031][id-bl-031] | PARTIAL |
| [FR-021][id-fr-021] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-007][id-uc-007], [UC-008][id-uc-008] | [AC-007][id-ac-007] → [TC-007][id-tc-007], [TC-030][id-tc-030] | [BL-030][id-bl-030], [BL-032][id-bl-032] | PARTIAL |
| [FR-022][id-fr-022] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-008][id-uc-008] | [AC-008][id-ac-008] → [TC-008][id-tc-008], [TC-027][id-tc-027] | [BL-032][id-bl-032] | PARTIAL |
| [FR-023][id-fr-023] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-008][id-uc-008] | [AC-008][id-ac-008] → [TC-008][id-tc-008], [TC-027][id-tc-027] | [BL-032][id-bl-032] | PARTIAL |
| [FR-024][id-fr-024] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-009][id-uc-009] | [AC-009][id-ac-009] → [TC-009][id-tc-009], [TC-028][id-tc-028] | [BL-033][id-bl-033] | PARTIAL |
| [FR-025][id-fr-025] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-007][id-uc-007] | [AC-010][id-ac-010] → [TC-010][id-tc-010], [TC-024][id-tc-024], [TC-028][id-tc-028] | [BL-034][id-bl-034] | PARTIAL |
| [FR-026][id-fr-026] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-007][id-uc-007] | [AC-010][id-ac-010] → [TC-010][id-tc-010], [TC-024][id-tc-024], [TC-028][id-tc-028] | [BL-034][id-bl-034], [BL-051][id-bl-051] | PARTIAL |
| [FR-027][id-fr-027] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-007][id-uc-007], [UC-008][id-uc-008] | [AC-007][id-ac-007] → [TC-007][id-tc-007], [TC-030][id-tc-030] | [BL-023][id-bl-023], [BL-031][id-bl-031], [BL-032][id-bl-032] | PARTIAL |
| [FR-028][id-fr-028] / S | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | — | При включении S/C | [BL-035][id-bl-035] | OUT_OF_BASE |
| [FR-029][id-fr-029] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-019][id-uc-019], [UC-020][id-uc-020] | [AC-018][id-ac-018] → [TC-018][id-tc-018], [TC-025][id-tc-025], [TC-030][id-tc-030], [TC-031][id-tc-031] | [BL-036][id-bl-036] | NOT_RUN |
| [FR-030][id-fr-030] / M | [runtime][g-runtime]; [Manager](40-agent-execution-and-security.md#runtime) | [UC-007][id-uc-007] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | [BL-040][id-bl-040] | NOT_RUN |
| [FR-031][id-fr-031] / M | [runtime][g-runtime]; [Manager](40-agent-execution-and-security.md#runtime) | [UC-007][id-uc-007] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | [BL-040][id-bl-040], [BL-045][id-bl-045] | NOT_RUN |
| [FR-032][id-fr-032] / M | [runtime][g-runtime]; [Manager](40-agent-execution-and-security.md#runtime) | [UC-007][id-uc-007] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | [BL-042][id-bl-042], [BL-050][id-bl-050] | NOT_RUN |
| [FR-033][id-fr-033] / M | [runtime][g-runtime]; [Manager](40-agent-execution-and-security.md#runtime) | [UC-007][id-uc-007] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | [BL-042][id-bl-042] | NOT_RUN |
| [FR-034][id-fr-034] / M | [runtime][g-runtime]; [Manager](40-agent-execution-and-security.md#runtime) | [UC-007][id-uc-007] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | [BL-041][id-bl-041] | NOT_RUN |
| [FR-035][id-fr-035] / M | [runtime][g-runtime]; [Manager](40-agent-execution-and-security.md#runtime) | [UC-007][id-uc-007], [UC-009][id-uc-009] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | [BL-044][id-bl-044] | NOT_RUN |
| [FR-036][id-fr-036] / M | [runtime][g-runtime]; [Manager](40-agent-execution-and-security.md#runtime) | [UC-007][id-uc-007] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | [BL-070][id-bl-070], [BL-072][id-bl-072] | NOT_RUN |
| [FR-037][id-fr-037] / M | [policy][g-policy]/[approvals][g-approval]; [decision](36-api-events-and-messages.md#approvals) | [UC-007][id-uc-007], [UC-012][id-uc-012] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | [BL-050][id-bl-050] | PARTIAL |
| [FR-038][id-fr-038] / M | [policy][g-policy]/[approvals][g-approval]; [decision](36-api-events-and-messages.md#approvals) | [UC-007][id-uc-007], [UC-012][id-uc-012] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | [BL-051][id-bl-051] | PARTIAL |
| [FR-039][id-fr-039] / M | [policy][g-policy]/[approvals][g-approval]; [decision](36-api-events-and-messages.md#approvals) | [UC-007][id-uc-007], [UC-012][id-uc-012] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | [BL-054][id-bl-054] | PARTIAL |
| [FR-040][id-fr-040] / M | [policy][g-policy]/[approvals][g-approval]; [decision](36-api-events-and-messages.md#approvals) | [UC-012][id-uc-012] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | [BL-052][id-bl-052], [BL-100][id-bl-100] | PARTIAL |
| [FR-041][id-fr-041] / M | [policy][g-policy]/[approvals][g-approval]; [decision](36-api-events-and-messages.md#approvals) | [UC-012][id-uc-012] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | [BL-052][id-bl-052] | PARTIAL |
| [FR-042][id-fr-042] / M | [policy][g-policy]/[approvals][g-approval]; [decision](36-api-events-and-messages.md#approvals) | [UC-012][id-uc-012] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | [BL-052][id-bl-052] | PARTIAL |
| [FR-043][id-fr-043] / M | [policy][g-policy]/[approvals][g-approval]; [decision](36-api-events-and-messages.md#approvals) | [UC-009][id-uc-009], [UC-012][id-uc-012] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | [BL-054][id-bl-054] | NOT_RUN |
| [FR-044][id-fr-044] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-010][id-uc-010] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | [BL-063][id-bl-063] | NOT_RUN |
| [FR-045][id-fr-045] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-011][id-uc-011] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | [BL-060][id-bl-060] | NOT_RUN |
| [FR-046][id-fr-046] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-011][id-uc-011] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | [BL-061][id-bl-061], [BL-066][id-bl-066] | NOT_RUN |
| [FR-047][id-fr-047] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-011][id-uc-011] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | [BL-061][id-bl-061] | NOT_RUN |
| [FR-048][id-fr-048] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-010][id-uc-010], [UC-011][id-uc-011] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | [BL-062][id-bl-062] | NOT_RUN |
| [FR-049][id-fr-049] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-007][id-uc-007], [UC-011][id-uc-011] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | [BL-062][id-bl-062] | NOT_RUN |
| [FR-050][id-fr-050] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-011][id-uc-011] | [AC-014][id-ac-014] → [TC-014][id-tc-014], [TC-025][id-tc-025] | [BL-069][id-bl-069] | NOT_RUN |
| [FR-051][id-fr-051] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-011][id-uc-011] | [AC-010][id-ac-010] → [TC-010][id-tc-010], [TC-024][id-tc-024], [TC-028][id-tc-028] | [BL-064][id-bl-064] | NOT_RUN |
| [FR-052][id-fr-052] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-011][id-uc-011] | [AC-010][id-ac-010] → [TC-010][id-tc-010], [TC-024][id-tc-024], [TC-028][id-tc-028] | [BL-064][id-bl-064] | NOT_RUN |
| [FR-053][id-fr-053] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-011][id-uc-011] | [AC-010][id-ac-010] → [TC-010][id-tc-010], [TC-024][id-tc-024], [TC-028][id-tc-028] | [BL-064][id-bl-064] | PARTIAL |
| [FR-054][id-fr-054] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-009][id-uc-009] | [AC-009][id-ac-009] → [TC-009][id-tc-009], [TC-028][id-tc-028] | [BL-065][id-bl-065] | PARTIAL |
| [FR-055][id-fr-055] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-008][id-uc-008], [UC-011][id-uc-011] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | [BL-067][id-bl-067] | NOT_RUN |
| [FR-056][id-fr-056] / S | [GitHub][g-github] S; [условное ADR](20-architecture-and-decisions.md#adr-10) | — | При включении S/C | [BL-070][id-bl-070] | OUT_OF_BASE |
| [FR-057][id-fr-057] / S | [GitHub][g-github] S; [условное ADR](20-architecture-and-decisions.md#adr-10) | — | При включении S/C | [BL-071][id-bl-071] | OUT_OF_BASE |
| [FR-058][id-fr-058] / S | [GitHub][g-github] S; [условное ADR](20-architecture-and-decisions.md#adr-10) | — | При включении S/C | [BL-071][id-bl-071] | OUT_OF_BASE |
| [FR-059][id-fr-059] / M/S | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-011][id-uc-011] | [AC-014][id-ac-014] → [TC-014][id-tc-014], [TC-025][id-tc-025] | [BL-071][id-bl-071], [BL-118][id-bl-118] | PARTIAL |
| [FR-060][id-fr-060] / M | packages; [ZIP/mapping](38-agent-package-format.md#pipeline) | [UC-015][id-uc-015] | [AC-015][id-ac-015] → [TC-015][id-tc-015], [TC-026][id-tc-026], [TC-032][id-tc-032] | [BL-080][id-bl-080] | NOT_RUN |
| [FR-061][id-fr-061] / M | packages; [ZIP/mapping](38-agent-package-format.md#pipeline) | [UC-015][id-uc-015] | [AC-015][id-ac-015] → [TC-015][id-tc-015], [TC-026][id-tc-026], [TC-032][id-tc-032] | [BL-081][id-bl-081] | NOT_RUN |
| [FR-062][id-fr-062] / M | packages; [ZIP/mapping](38-agent-package-format.md#pipeline) | [UC-016][id-uc-016] | [AC-015][id-ac-015] → [TC-015][id-tc-015], [TC-026][id-tc-026], [TC-032][id-tc-032] | [BL-083][id-bl-083] | NOT_RUN |
| [FR-063][id-fr-063] / M | packages; [ZIP/mapping](38-agent-package-format.md#pipeline) | [UC-016][id-uc-016] | [AC-015][id-ac-015] → [TC-015][id-tc-015], [TC-026][id-tc-026], [TC-032][id-tc-032] | [BL-082][id-bl-082] | NOT_RUN |
| [FR-064][id-fr-064] / M | packages; [ZIP/mapping](38-agent-package-format.md#pipeline) | [UC-017][id-uc-017] | [AC-016][id-ac-016] → [TC-016][id-tc-016], [TC-026][id-tc-026] | [BL-082][id-bl-082] | NOT_RUN |
| [FR-065][id-fr-065] / M | [Flutter][g-flutter]/identity; [экраны](61-android-screens-flows-and-api.md#screens) | [UC-005][id-uc-005] | [AC-003][id-ac-003] → [TC-003][id-tc-003], [TC-022][id-tc-022], [TC-032][id-tc-032] | [BL-100][id-bl-100], [BL-110][id-bl-110], [BL-121][id-bl-121] | NOT_RUN |
| [FR-066][id-fr-066] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-007][id-uc-007], [UC-009][id-uc-009], [UC-020][id-uc-020] | [AC-007][id-ac-007], [AC-009][id-ac-009] → [TC-007][id-tc-007], [TC-009][id-tc-009], [TC-028][id-tc-028], [TC-030][id-tc-030] | [BL-117][id-bl-117] | NOT_RUN |
| [FR-067][id-fr-067] / M | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-008][id-uc-008] | [AC-008][id-ac-008] → [TC-008][id-tc-008], [TC-027][id-tc-027] | [BL-111][id-bl-111] | NOT_RUN |
| [FR-068][id-fr-068] / M | [policy][g-policy]/[approvals][g-approval]; [decision](36-api-events-and-messages.md#approvals) | [UC-012][id-uc-012] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | [BL-112][id-bl-112] | NOT_RUN |
| [FR-069][id-fr-069] / M | messaging/budgets; [envelope](36-api-events-and-messages.md#messages) | [UC-010][id-uc-010] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | [BL-113][id-bl-113] | NOT_RUN |
| [FR-070][id-fr-070] / M/S | [runs][g-run]; [run/SSE/result](36-api-events-and-messages.md#runs) | [UC-011][id-uc-011] | [AC-014][id-ac-014] → [TC-014][id-tc-014], [TC-025][id-tc-025] | [BL-118][id-bl-118] | NOT_RUN |
| [FR-071][id-fr-071] / S | [Runner][g-runner] S; [отложено](20-architecture-and-decisions.md#adr-08) | — | При включении S/C | [BL-043][id-bl-043] | OUT_OF_BASE |
| [FR-072][id-fr-072] / S | [Runner][g-runner] S; [отложено](20-architecture-and-decisions.md#adr-08) | — | При включении S/C | [BL-043][id-bl-043] | OUT_OF_BASE |
| [FR-073][id-fr-073] / C | schedule C; [границы](10-project-scope-and-constraints.md#scope) | — | При включении S/C | [BL-090][id-bl-090], [BL-115][id-bl-115] | OUT_OF_BASE |
| [FR-074][id-fr-074] / C | schedule C; [границы](10-project-scope-and-constraints.md#scope) | — | При включении S/C | [BL-091][id-bl-091] | OUT_OF_BASE |
| [FR-075][id-fr-075] / M | [push][g-push]/identity; [lifecycle](61-android-screens-flows-and-api.md#push) | [UC-013][id-uc-013] | [AC-012][id-ac-012] → [TC-012][id-tc-012], [TC-032][id-tc-032] | [BL-053][id-bl-053], [BL-125][id-bl-125] | NOT_RUN |
| [FR-076][id-fr-076] / M | [push][g-push]/identity; [lifecycle](61-android-screens-flows-and-api.md#push) | [UC-013][id-uc-013] | [AC-012][id-ac-012] → [TC-012][id-tc-012], [TC-032][id-tc-032] | [BL-116][id-bl-116] | NOT_RUN |
| [FR-077][id-fr-077] / M | packages; [ZIP/mapping](38-agent-package-format.md#pipeline) | [UC-015][id-uc-015], [UC-016][id-uc-016], [UC-017][id-uc-017] | [AC-015][id-ac-015] → [TC-015][id-tc-015], [TC-026][id-tc-026], [TC-032][id-tc-032] | [BL-114][id-bl-114] | NOT_RUN |
| [FR-078][id-fr-078] / S | gateway/providers; [provider](36-api-events-and-messages.md#agents) | — | При включении S/C | [BL-024][id-bl-024] | OUT_OF_BASE |
| [FR-079][id-fr-079] / S | audit/metrics S/C; [границы](10-project-scope-and-constraints.md#scope) | — | При включении S/C | [BL-101][id-bl-101] | OUT_OF_BASE |
| [FR-080][id-fr-080] / C | audit/metrics S/C; [границы](10-project-scope-and-constraints.md#scope) | — | При включении S/C | [BL-102][id-bl-102] | OUT_OF_BASE |
| [FR-081][id-fr-081] / M | identity/audit; [auth](36-api-events-and-messages.md#auth) | [UC-001][id-uc-001] | [AC-002][id-ac-002] → [TC-002][id-tc-002], [TC-022][id-tc-022], [TC-029][id-tc-029] | [BL-121][id-bl-121] | PARTIAL |
| [FR-082][id-fr-082] / M | [Flutter][g-flutter]/identity; [экраны](61-android-screens-flows-and-api.md#screens) | [UC-002][id-uc-002], [UC-003][id-uc-003] | [AC-003][id-ac-003] → [TC-003][id-tc-003], [TC-022][id-tc-022], [TC-032][id-tc-032] | [BL-120][id-bl-120] | NOT_RUN |
| [FR-083][id-fr-083] / M | [Flutter][g-flutter]/identity; [экраны](61-android-screens-flows-and-api.md#screens) | [UC-003][id-uc-003] | [AC-003][id-ac-003] → [TC-003][id-tc-003], [TC-022][id-tc-022], [TC-032][id-tc-032] | [BL-122][id-bl-122] | NOT_RUN |
| [FR-084][id-fr-084] / M | [Flutter][g-flutter]/identity; [экраны](61-android-screens-flows-and-api.md#screens) | [UC-003][id-uc-003] | [AC-003][id-ac-003] → [TC-003][id-tc-003], [TC-022][id-tc-022], [TC-032][id-tc-032] | [BL-123][id-bl-123] | PARTIAL |
| [FR-085][id-fr-085] / M | identity/audit; [auth](36-api-events-and-messages.md#auth) | [UC-004][id-uc-004] | [AC-002][id-ac-002] → [TC-002][id-tc-002], [TC-022][id-tc-022], [TC-029][id-tc-029] | [BL-124][id-bl-124] | PARTIAL |
| [FR-086][id-fr-086] / M | [push][g-push]/identity; [lifecycle](61-android-screens-flows-and-api.md#push) | [UC-014][id-uc-014] | [AC-012][id-ac-012] → [TC-012][id-tc-012], [TC-032][id-tc-032] | [BL-125][id-bl-125] | NOT_RUN |
| [FR-087][id-fr-087] / M | [agents][g-agent]; [versions/roles](36-api-events-and-messages.md#agents) | [UC-005][id-uc-005] | [AC-004][id-ac-004] → [TC-004][id-tc-004], [TC-021][id-tc-021] | [BL-014][id-bl-014] | PARTIAL |
| [FR-088][id-fr-088] / M | packages; [ZIP/mapping](38-agent-package-format.md#pipeline) | [UC-017][id-uc-017] | [AC-016][id-ac-016] → [TC-016][id-tc-016], [TC-026][id-tc-026] | [BL-084][id-bl-084] | NOT_RUN |
| [FR-089][id-fr-089] / M | audit/status; [read API](36-api-events-and-messages.md#operations) | [UC-018][id-uc-018] | [AC-017][id-ac-017] → [TC-017][id-tc-017], [TC-022][id-tc-022], [TC-029][id-tc-029], [TC-031][id-tc-031] | [BL-104][id-bl-104] | PARTIAL |
| [FR-090][id-fr-090] / M | [Flutter][g-flutter]/identity; [экраны](61-android-screens-flows-and-api.md#screens) | [UC-002][id-uc-002], [UC-003][id-uc-003] | [AC-003][id-ac-003] → [TC-003][id-tc-003], [TC-022][id-tc-022], [TC-032][id-tc-032] | [BL-008][id-bl-008], [BL-009][id-bl-009] | NOT_RUN |
| [NFR-001][id-nfr-001] / M | identity/[policy][g-policy]/[runtime][g-runtime]/packages; [защиты](40-agent-execution-and-security.md#threats) | [QS-005][id-qs-005], [UC-006][id-uc-006] | [AC-017][id-ac-017] → [TC-017][id-tc-017], [TC-022][id-tc-022], [TC-029][id-tc-029], [TC-031][id-tc-031] | [BL-025][id-bl-025] | PARTIAL |
| [NFR-002][id-nfr-002] / M | identity/[policy][g-policy]/[runtime][g-runtime]/packages; [защиты](40-agent-execution-and-security.md#threats) | [QS-005][id-qs-005], [QS-019][id-qs-019], [UC-005][id-uc-005], [UC-006][id-uc-006], [UC-014][id-uc-014], [UC-015][id-uc-015], [UC-016][id-uc-016], [UC-018][id-uc-018] | [AC-006][id-ac-006], [AC-015][id-ac-015], [AC-017][id-ac-017] → [TC-006][id-tc-006], [TC-015][id-tc-015], [TC-017][id-tc-017], [TC-022][id-tc-022], [TC-023][id-tc-023], [TC-026][id-tc-026], [TC-029][id-tc-029], [TC-031][id-tc-031], [TC-032][id-tc-032] | [BL-025][id-bl-025], [BL-081][id-bl-081] | NOT_RUN |
| [NFR-003][id-nfr-003] / M | identity/[policy][g-policy]/[runtime][g-runtime]/packages; [защиты](40-agent-execution-and-security.md#threats) | [QS-004][id-qs-004], [UC-007][id-uc-007] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | [BL-040][id-bl-040], [BL-041][id-bl-041], [BL-045][id-bl-045] | NOT_RUN |
| [NFR-004][id-nfr-004] / M | identity/[policy][g-policy]/[runtime][g-runtime]/packages; [защиты](40-agent-execution-and-security.md#threats) | [QS-003][id-qs-003], [UC-005][id-uc-005], [UC-007][id-uc-007], [UC-010][id-uc-010], [UC-011][id-uc-011], [UC-012][id-uc-012], [UC-016][id-uc-016], [UC-020][id-uc-020] | [AC-004][id-ac-004], [AC-006][id-ac-006], [AC-011][id-ac-011], [AC-013][id-ac-013] → [TC-004][id-tc-004], [TC-006][id-tc-006], [TC-011][id-tc-011], [TC-013][id-tc-013], [TC-021][id-tc-021], [TC-023][id-tc-023], [TC-024][id-tc-024], [TC-030][id-tc-030] | [BL-050][id-bl-050], [BL-066][id-bl-066] | PARTIAL |
| [NFR-005][id-nfr-005] / M | identity/[policy][g-policy]/[runtime][g-runtime]/packages; [защиты](40-agent-execution-and-security.md#threats) | [QS-001][id-qs-001], [QS-002][id-qs-002], [UC-001][id-uc-001], [UC-002][id-uc-002], [UC-003][id-uc-003], [UC-004][id-uc-004], [UC-013][id-uc-013], [UC-014][id-uc-014] | [AC-002][id-ac-002] → [TC-002][id-tc-002], [TC-022][id-tc-022], [TC-029][id-tc-029] | [BL-043][id-bl-043], [BL-100][id-bl-100] | PARTIAL |
| [NFR-006][id-nfr-006] / M | storage/[outbox][g-outbox]/[recovery][g-recovery]; [транзакции](29-data-and-state-model.md#indexes) | [QS-008][id-qs-008], [QS-009][id-qs-009], [UC-001][id-uc-001], [UC-005][id-uc-005], [UC-008][id-uc-008], [UC-020][id-uc-020] | [AC-004][id-ac-004], [AC-008][id-ac-008], [AC-014][id-ac-014], [AC-018][id-ac-018] → [TC-004][id-tc-004], [TC-008][id-tc-008], [TC-014][id-tc-014], [TC-018][id-tc-018], [TC-021][id-tc-021], [TC-025][id-tc-025], [TC-027][id-tc-027], [TC-030][id-tc-030], [TC-031][id-tc-031] | [BL-012][id-bl-012], [BL-032][id-bl-032], [BL-052][id-bl-052] | PARTIAL |
| [NFR-007][id-nfr-007] / M | storage/[outbox][g-outbox]/[recovery][g-recovery]; [транзакции](29-data-and-state-model.md#indexes) | [QS-003][id-qs-003], [QS-007][id-qs-007], [UC-009][id-uc-009], [UC-010][id-uc-010], [UC-011][id-uc-011], [UC-012][id-uc-012] | [AC-007][id-ac-007], [AC-011][id-ac-011], [AC-013][id-ac-013], [AC-014][id-ac-014] → [TC-007][id-tc-007], [TC-011][id-tc-011], [TC-013][id-tc-013], [TC-014][id-tc-014], [TC-021][id-tc-021], [TC-023][id-tc-023], [TC-024][id-tc-024], [TC-025][id-tc-025], [TC-030][id-tc-030] | [BL-054][id-bl-054], [BL-061][id-bl-061], [BL-062][id-bl-062] | PARTIAL |
| [NFR-008][id-nfr-008] / M | [events][g-event]/[stop][g-cancelled]/ops; [метод](#measurements) | [QS-010][id-qs-010], [UC-008][id-uc-008] | [AC-008][id-ac-008], [AC-019][id-ac-019] → [TC-008][id-tc-008], [TC-019][id-tc-019], [TC-027][id-tc-027], [TC-028][id-tc-028] | [BL-032][id-bl-032], [BL-111][id-bl-111] | NOT_RUN |
| [NFR-009][id-nfr-009] / M | [events][g-event]/[stop][g-cancelled]/ops; [метод](#measurements) | [QS-011][id-qs-011], [UC-009][id-uc-009] | [AC-009][id-ac-009], [AC-019][id-ac-019] → [TC-009][id-tc-009], [TC-019][id-tc-019], [TC-027][id-tc-027], [TC-028][id-tc-028] | [BL-033][id-bl-033], [BL-065][id-bl-065] | NOT_RUN |
| [NFR-010][id-nfr-010] / M | identity/[policy][g-policy]/[runtime][g-runtime]/packages; [защиты](40-agent-execution-and-security.md#threats) | [QS-012][id-qs-012], [UC-006][id-uc-006], [UC-007][id-uc-007], [UC-011][id-uc-011], [UC-012][id-uc-012] | [AC-005][id-ac-005], [AC-010][id-ac-010] → [TC-005][id-tc-005], [TC-010][id-tc-010], [TC-024][id-tc-024], [TC-028][id-tc-028] | [BL-023][id-bl-023], [BL-034][id-bl-034], [BL-064][id-bl-064] | PARTIAL |
| [NFR-011][id-nfr-011] / M | audit/status; [контракт](36-api-events-and-messages.md#operations) | [QS-013][id-qs-013], [QS-014][id-qs-014], [UC-004][id-uc-004], [UC-007][id-uc-007], [UC-008][id-uc-008], [UC-013][id-uc-013] | [AC-007][id-ac-007], [AC-009][id-ac-009], [AC-012][id-ac-012] → [TC-007][id-tc-007], [TC-009][id-tc-009], [TC-012][id-tc-012], [TC-028][id-tc-028], [TC-030][id-tc-030], [TC-032][id-tc-032] | [BL-030][id-bl-030], [BL-051][id-bl-051], [BL-053][id-bl-053], [BL-111][id-bl-111], [BL-116][id-bl-116] | NOT_RUN |
| [NFR-012][id-nfr-012] / M | modules/[CI][g-ci-cd]; [архитектура](20-architecture-and-decisions.md#uml-components) | [QS-018][id-qs-018], [UC-019][id-uc-019] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | [BL-003][id-bl-003] | NOT_RUN |
| [NFR-013][id-nfr-013] / M | modules/[CI][g-ci-cd]; [архитектура](20-architecture-and-decisions.md#uml-components) | [QS-018][id-qs-018], [UC-019][id-uc-019] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | [BL-006][id-bl-006] | PARTIAL |
| [NFR-014][id-nfr-014] / M | modules/[CI][g-ci-cd]; [архитектура](20-architecture-and-decisions.md#uml-components) | [QS-018][id-qs-018], [UC-019][id-uc-019] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | [BL-004][id-bl-004] | PARTIAL |
| [NFR-015][id-nfr-015] / M | modules/[CI][g-ci-cd]; [архитектура](20-architecture-and-decisions.md#uml-components) | [QS-021][id-qs-021], [UC-007][id-uc-007] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | [BL-005][id-bl-005], [BL-007][id-bl-007], [BL-119][id-bl-119] | PARTIAL |
| [NFR-016][id-nfr-016] / M | modules/[CI][g-ci-cd]; [архитектура](20-architecture-and-decisions.md#uml-components) | [QS-016][id-qs-016], [UC-006][id-uc-006], [UC-017][id-uc-017] | [AC-001][id-ac-001], [AC-005][id-ac-005] → [TC-001][id-tc-001], [TC-005][id-tc-005], [TC-031][id-tc-031] | [BL-005][id-bl-005], [BL-023][id-bl-023], [BL-031][id-bl-031], [BL-042][id-bl-042] | PARTIAL |
| [NFR-017][id-nfr-017] / M | packages/bindings; [перенос](38-agent-package-format.md#portability) | [QS-015][id-qs-015], [UC-011][id-uc-011], [UC-015][id-uc-015], [UC-016][id-uc-016], [UC-017][id-uc-017] | [AC-016][id-ac-016] → [TC-016][id-tc-016], [TC-026][id-tc-026] | [BL-080][id-bl-080], [BL-081][id-bl-081], [BL-082][id-bl-082], [BL-084][id-bl-084] | NOT_RUN |
| [NFR-018][id-nfr-018] / M | audit/status; [контракт](36-api-events-and-messages.md#operations) | [QS-019][id-qs-019], [UC-004][id-uc-004], [UC-007][id-uc-007], [UC-008][id-uc-008], [UC-009][id-uc-009], [UC-011][id-uc-011], [UC-013][id-uc-013], [UC-017][id-uc-017], [UC-018][id-uc-018], [UC-020][id-uc-020] | [AC-007][id-ac-007], [AC-017][id-ac-017] → [TC-007][id-tc-007], [TC-017][id-tc-017], [TC-022][id-tc-022], [TC-029][id-tc-029], [TC-030][id-tc-030], [TC-031][id-tc-031] | [BL-032][id-bl-032], [BL-067][id-bl-067], [BL-103][id-bl-103] | PARTIAL |
| [NFR-019][id-nfr-019] / M | [Flutter][g-flutter]; [states/a11y](61-android-screens-flows-and-api.md#states) | [QS-002][id-qs-002], [QS-014][id-qs-014], [QS-020][id-qs-020], [UC-001][id-uc-001], [UC-002][id-uc-002], [UC-003][id-uc-003], [UC-005][id-uc-005], [UC-008][id-uc-008], [UC-010][id-uc-010], [UC-012][id-uc-012], [UC-013][id-uc-013], [UC-014][id-uc-014], [UC-018][id-uc-018] | [AC-002][id-ac-002], [AC-003][id-ac-003], [AC-011][id-ac-011], [AC-012][id-ac-012] → [TC-002][id-tc-002], [TC-003][id-tc-003], [TC-011][id-tc-011], [TC-012][id-tc-012], [TC-021][id-tc-021], [TC-022][id-tc-022], [TC-023][id-tc-023], [TC-029][id-tc-029], [TC-032][id-tc-032] | [BL-110][id-bl-110], [BL-111][id-bl-111], [BL-112][id-bl-112], [BL-120][id-bl-120], [BL-122][id-bl-122] | NOT_RUN |
| [NFR-020][id-nfr-020] / M | DTO/schema; [API](36-api-events-and-messages.md#conventions) | [QS-017][id-qs-017], [UC-006][id-uc-006], [UC-015][id-uc-015], [UC-016][id-uc-016] | [AC-001][id-ac-001], [AC-005][id-ac-005], [AC-013][id-ac-013], [AC-015][id-ac-015] → [TC-001][id-tc-001], [TC-005][id-tc-005], [TC-013][id-tc-013], [TC-015][id-tc-015], [TC-024][id-tc-024], [TC-026][id-tc-026], [TC-030][id-tc-030], [TC-031][id-tc-031], [TC-032][id-tc-032] | [BL-023][id-bl-023], [BL-060][id-bl-060], [BL-080][id-bl-080], [BL-082][id-bl-082] | PARTIAL |
| [NFR-021][id-nfr-021] / M | identity/[policy][g-policy]/[runtime][g-runtime]/packages; [защиты](40-agent-execution-and-security.md#threats) | [QS-006][id-qs-006], [UC-016][id-uc-016] | [AC-015][id-ac-015] → [TC-015][id-tc-015], [TC-026][id-tc-026], [TC-032][id-tc-032] | [BL-083][id-bl-083] | NOT_RUN |
| [NFR-022][id-nfr-022] / M | [events][g-event]/[stop][g-cancelled]/ops; [метод](#measurements) | [QS-018][id-qs-018], [UC-019][id-uc-019] | [AC-001][id-ac-001], [AC-019][id-ac-019] → [TC-001][id-tc-001], [TC-019][id-tc-019], [TC-027][id-tc-027], [TC-028][id-tc-028], [TC-031][id-tc-031] | [BL-003][id-bl-003] | NOT_RUN |
| [NFR-023][id-nfr-023] / M | modules/[CI][g-ci-cd]; [архитектура](20-architecture-and-decisions.md#uml-components) | [QS-021][id-qs-021] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | [BL-001][id-bl-001], [BL-002][id-bl-002] | PARTIAL |
| [NFR-024][id-nfr-024] / M | storage/[outbox][g-outbox]/[recovery][g-recovery]; [транзакции](29-data-and-state-model.md#indexes) | [QS-009][id-qs-009], [UC-019][id-uc-019], [UC-020][id-uc-020] | [AC-018][id-ac-018] → [TC-018][id-tc-018], [TC-025][id-tc-025], [TC-030][id-tc-030], [TC-031][id-tc-031] | [BL-036][id-bl-036] | PARTIAL |

<a id="work-coverage"></a>
## 6. Работы, инженерная готовность и курс

[FR][g-requirement-ids]/[NFR][g-requirement-ids]→[BL][g-work-ids] в [RTM][g-traceability] сохраняют локальную связь [SRS][g-requirement-ids]. Инженерные [BL-001][id-bl-001]–007 дополнительно покрываются [TC-001][id-tc-001]/031, независимо от наличия [UI][g-ui]. Для проверки completeness ниже приведена обратная связь каждой обязательной [BL][g-work-ids]/[CW][g-work-ids] с [AC][g-acceptance]/[TC][g-acceptance]; закрытие работы требует её собственной формулировки [backlog][g-work-ids], не только наличия строки.

| Работа | [AC][g-acceptance] → [TC][g-acceptance] | Статус |
|---|---|---|
| [BL-001][id-bl-001] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | PARTIAL |
| [BL-002][id-bl-002] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | PARTIAL |
| [BL-003][id-bl-003] | [AC-001][id-ac-001], [AC-019][id-ac-019] → [TC-001][id-tc-001], [TC-019][id-tc-019], [TC-027][id-tc-027], [TC-028][id-tc-028], [TC-031][id-tc-031] | PARTIAL |
| [BL-004][id-bl-004] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | PARTIAL |
| [BL-005][id-bl-005] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | PARTIAL |
| [BL-006][id-bl-006] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | PARTIAL |
| [BL-007][id-bl-007] | [AC-001][id-ac-001] → [TC-001][id-tc-001], [TC-031][id-tc-031] | PARTIAL |
| [BL-008][id-bl-008] | [AC-003][id-ac-003], [AC-020][id-ac-020] → [TC-003][id-tc-003], [TC-020][id-tc-020], [TC-022][id-tc-022], [TC-032][id-tc-032] | NOT_RUN |
| [BL-009][id-bl-009] | [AC-003][id-ac-003] → [TC-003][id-tc-003], [TC-022][id-tc-022], [TC-032][id-tc-032] | PARTIAL |
| [BL-119][id-bl-119] | [AC-020][id-ac-020] → [TC-020][id-tc-020] | NOT_RUN |
| [BL-010][id-bl-010] | [AC-004][id-ac-004] → [TC-004][id-tc-004], [TC-021][id-tc-021] | PARTIAL |
| [BL-011][id-bl-011] | [AC-004][id-ac-004] → [TC-004][id-tc-004], [TC-021][id-tc-021] | PARTIAL |
| [BL-012][id-bl-012] | [AC-004][id-ac-004] → [TC-004][id-tc-004], [TC-021][id-tc-021] | PARTIAL |
| [BL-014][id-bl-014] | [AC-004][id-ac-004] → [TC-004][id-tc-004], [TC-021][id-tc-021] | PARTIAL |
| [BL-020][id-bl-020] | [AC-005][id-ac-005] → [TC-005][id-tc-005] | PARTIAL |
| [BL-021][id-bl-021] | [AC-005][id-ac-005] → [TC-005][id-tc-005] | NOT_RUN |
| [BL-023][id-bl-023] | [AC-005][id-ac-005] → [TC-005][id-tc-005] | PARTIAL |
| [BL-025][id-bl-025] | [AC-005][id-ac-005], [AC-017][id-ac-017] → [TC-005][id-tc-005], [TC-017][id-tc-017], [TC-022][id-tc-022], [TC-029][id-tc-029], [TC-031][id-tc-031] | PARTIAL |
| [BL-030][id-bl-030] | [AC-007][id-ac-007] → [TC-007][id-tc-007], [TC-030][id-tc-030] | PARTIAL |
| [BL-031][id-bl-031] | [AC-007][id-ac-007] → [TC-007][id-tc-007], [TC-030][id-tc-030] | PARTIAL |
| [BL-032][id-bl-032] | [AC-008][id-ac-008], [AC-019][id-ac-019] → [TC-008][id-tc-008], [TC-019][id-tc-019], [TC-027][id-tc-027], [TC-028][id-tc-028] | PARTIAL |
| [BL-033][id-bl-033] | [AC-009][id-ac-009], [AC-019][id-ac-019] → [TC-009][id-tc-009], [TC-019][id-tc-019], [TC-027][id-tc-027], [TC-028][id-tc-028] | PARTIAL |
| [BL-034][id-bl-034] | [AC-010][id-ac-010] → [TC-010][id-tc-010], [TC-024][id-tc-024], [TC-028][id-tc-028] | PARTIAL |
| [BL-036][id-bl-036] | [AC-018][id-ac-018] → [TC-018][id-tc-018], [TC-025][id-tc-025], [TC-030][id-tc-030], [TC-031][id-tc-031] | PARTIAL |
| [BL-040][id-bl-040] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | NOT_RUN |
| [BL-041][id-bl-041] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | NOT_RUN |
| [BL-042][id-bl-042] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | NOT_RUN |
| [BL-044][id-bl-044] | [AC-006][id-ac-006], [AC-018][id-ac-018] → [TC-006][id-tc-006], [TC-018][id-tc-018], [TC-023][id-tc-023], [TC-025][id-tc-025], [TC-030][id-tc-030], [TC-031][id-tc-031] | NOT_RUN |
| [BL-045][id-bl-045] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | NOT_RUN |
| [BL-050][id-bl-050] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | PARTIAL |
| [BL-051][id-bl-051] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | PARTIAL |
| [BL-052][id-bl-052] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | PARTIAL |
| [BL-053][id-bl-053] | [AC-012][id-ac-012] → [TC-012][id-tc-012], [TC-032][id-tc-032] | NOT_RUN |
| [BL-054][id-bl-054] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | PARTIAL |
| [BL-060][id-bl-060] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | NOT_RUN |
| [BL-061][id-bl-061] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | PARTIAL |
| [BL-062][id-bl-062] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | NOT_RUN |
| [BL-063][id-bl-063] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | NOT_RUN |
| [BL-064][id-bl-064] | [AC-010][id-ac-010] → [TC-010][id-tc-010], [TC-024][id-tc-024], [TC-028][id-tc-028] | PARTIAL |
| [BL-065][id-bl-065] | [AC-009][id-ac-009], [AC-019][id-ac-019] → [TC-009][id-tc-009], [TC-019][id-tc-019], [TC-027][id-tc-027], [TC-028][id-tc-028] | PARTIAL |
| [BL-066][id-bl-066] | [AC-011][id-ac-011], [AC-013][id-ac-013] → [TC-011][id-tc-011], [TC-013][id-tc-013], [TC-021][id-tc-021], [TC-023][id-tc-023], [TC-024][id-tc-024], [TC-030][id-tc-030] | NOT_RUN |
| [BL-067][id-bl-067] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | NOT_RUN |
| [BL-069][id-bl-069] | [AC-014][id-ac-014] → [TC-014][id-tc-014], [TC-025][id-tc-025] | NOT_RUN |
| [BL-072][id-bl-072] | [AC-006][id-ac-006] → [TC-006][id-tc-006], [TC-023][id-tc-023] | NOT_RUN |
| [BL-080][id-bl-080] | [AC-015][id-ac-015] → [TC-015][id-tc-015], [TC-026][id-tc-026], [TC-032][id-tc-032] | NOT_RUN |
| [BL-081][id-bl-081] | [AC-015][id-ac-015] → [TC-015][id-tc-015], [TC-026][id-tc-026], [TC-032][id-tc-032] | NOT_RUN |
| [BL-082][id-bl-082] | [AC-016][id-ac-016] → [TC-016][id-tc-016], [TC-026][id-tc-026] | NOT_RUN |
| [BL-083][id-bl-083] | [AC-015][id-ac-015] → [TC-015][id-tc-015], [TC-026][id-tc-026], [TC-032][id-tc-032] | NOT_RUN |
| [BL-084][id-bl-084] | [AC-016][id-ac-016] → [TC-016][id-tc-016], [TC-026][id-tc-026] | NOT_RUN |
| [BL-100][id-bl-100] | [AC-002][id-ac-002] → [TC-002][id-tc-002], [TC-022][id-tc-022], [TC-029][id-tc-029] | PARTIAL |
| [BL-103][id-bl-103] | [AC-017][id-ac-017] → [TC-017][id-tc-017], [TC-022][id-tc-022], [TC-029][id-tc-029], [TC-031][id-tc-031] | PARTIAL |
| [BL-104][id-bl-104] | [AC-017][id-ac-017] → [TC-017][id-tc-017], [TC-022][id-tc-022], [TC-029][id-tc-029], [TC-031][id-tc-031] | PARTIAL |
| [BL-110][id-bl-110] | [AC-003][id-ac-003], [AC-004][id-ac-004] → [TC-003][id-tc-003], [TC-004][id-tc-004], [TC-021][id-tc-021], [TC-022][id-tc-022], [TC-032][id-tc-032] | NOT_RUN |
| [BL-111][id-bl-111] | [AC-008][id-ac-008], [AC-019][id-ac-019] → [TC-008][id-tc-008], [TC-019][id-tc-019], [TC-027][id-tc-027], [TC-028][id-tc-028] | NOT_RUN |
| [BL-112][id-bl-112] | [AC-011][id-ac-011] → [TC-011][id-tc-011], [TC-021][id-tc-021], [TC-023][id-tc-023] | NOT_RUN |
| [BL-113][id-bl-113] | [AC-013][id-ac-013] → [TC-013][id-tc-013], [TC-024][id-tc-024], [TC-030][id-tc-030] | NOT_RUN |
| [BL-114][id-bl-114] | [AC-015][id-ac-015] → [TC-015][id-tc-015], [TC-026][id-tc-026], [TC-032][id-tc-032] | NOT_RUN |
| [BL-116][id-bl-116] | [AC-012][id-ac-012] → [TC-012][id-tc-012], [TC-032][id-tc-032] | NOT_RUN |
| [BL-117][id-bl-117] | [AC-007][id-ac-007], [AC-009][id-ac-009] → [TC-007][id-tc-007], [TC-009][id-tc-009], [TC-028][id-tc-028], [TC-030][id-tc-030] | NOT_RUN |
| [BL-118][id-bl-118] | [AC-007][id-ac-007], [AC-014][id-ac-014] → [TC-007][id-tc-007], [TC-014][id-tc-014], [TC-025][id-tc-025], [TC-030][id-tc-030] | NOT_RUN |
| [BL-120][id-bl-120] | [AC-003][id-ac-003] → [TC-003][id-tc-003], [TC-022][id-tc-022], [TC-032][id-tc-032] | NOT_RUN |
| [BL-121][id-bl-121] | [AC-002][id-ac-002] → [TC-002][id-tc-002], [TC-022][id-tc-022], [TC-029][id-tc-029] | PARTIAL |
| [BL-122][id-bl-122] | [AC-003][id-ac-003] → [TC-003][id-tc-003], [TC-022][id-tc-022], [TC-032][id-tc-032] | NOT_RUN |
| [BL-123][id-bl-123] | [AC-003][id-ac-003] → [TC-003][id-tc-003], [TC-022][id-tc-022], [TC-032][id-tc-032] | PARTIAL |
| [BL-124][id-bl-124] | [AC-002][id-ac-002] → [TC-002][id-tc-002], [TC-022][id-tc-022], [TC-029][id-tc-029] | PARTIAL |
| [BL-125][id-bl-125] | [AC-012][id-ac-012] → [TC-012][id-tc-012], [TC-032][id-tc-032] | NOT_RUN |
| [CW-001][id-cw-001] | [AC-020][id-ac-020] → [TC-020][id-tc-020] | NOT_RUN |
| [CW-002][id-cw-002] | [AC-020][id-ac-020] → [TC-020][id-tc-020] | NOT_RUN |
| [CW-003][id-cw-003] | [AC-020][id-ac-020] → [TC-020][id-tc-020] | NOT_RUN |
| [CW-004][id-cw-004] | [AC-020][id-ac-020] → [TC-020][id-tc-020] | NOT_RUN |

[CW-001][id-cw-001] введение, [CW-002][id-cw-002] теория, [CW-003][id-cw-003] библиография, [CW-004][id-cw-004] демо/защита проверяются [TC-020][id-tc-020] по реальным материалам. Ни этот тест-план, ни [ТЗ][g-requirement-ids]/[SRS][g-requirement-ids], ни анализ аналогов не являются уже написанной теоретической главой. КМ07-01–14 сверяются с канонической таблицей [SRS][g-requirement-ids] §7.2.

<a id="protocol"></a>
## 7. Форма фактического протокола и решение о готовности

| Поле протокола | Что заполнить после [запуска][g-run] |
|---|---|
| Candidate / окружение | Revision, [APK][g-apk]/hash/signing identity, image/DB/config versions, A/B/[model][g-model] IDs без secrets |
| [TC][g-acceptance] / уровень / [AC][g-acceptance] / FR-NFR | Точные IDs, [fake][g-fixture]/real/device различаются |
| Начало/конец / часы | [UTC][g-identifier-formats], offset/error, server/phone lifecycle, сеть/нагрузка |
| Предусловия / [шаги][g-step] | Фактические [fixture][g-fixture] IDs, source revision, [run][g-run]/[chain][g-chain]/[approval][g-approval]/message/dispatch IDs |
| Expected / actual | Измерения, assertions, endpoint/status/reason, unknown outcomes |
| [Evidence][g-evidence] | Относительный безопасный путь/URL, [SHA-256][g-config-hash], сырой отчёт/видео/[лог][g-audit] без [секретов][g-secret] |
| Outcome / defect | PASS / FAIL / BLOCKED / INCONCLUSIVE / NOT_RUN, дефект/причина |
| Исполнитель / решение | Кто запускал и кто подтвердил итог; принятие [владельцем][g-owner] при реальной демонстрации |

Текущий итог: ни один из 32 полных TC и 20 AC не принят целиком. Инженерные подмножества проверены и включены в evidence index выше; PARTIAL не снимает обязательные реальные проверки 77 FR и 24 NFR. Фиктивные PASS/видео/отчёты не создаются. После FAIL сохраняются исходный отчёт и defect, исправление повторяет затронутые [TC][g-acceptance] и полный candidate-прогон; новая влияющая revision/config требует актуальных [evidence][g-evidence]. При BLOCKED фиксируется конкретная зависимость, без снижения M и без маскировки real-проверки [fake][g-fixture].

Отдельный файл протокола не обязателен: фактические результаты дополняют этот раздел/таблицу и ссылки на test-generated artifacts. Обязательный остаток после проектирования — реализация [BL][g-work-ids], внешние доступы/стенд B, реальные прогоны, фактические [CW][g-work-ids] и репетиция. Новые формальные документы по исходным71 ID не требуются.

### Локальная проверка Flutter, 13.10.2026

[Протокол первого клиента](evidence/flutter-client-2026-10-13.md) содержит команды, тесты auth/refresh/idempotency/logout/events, widget Dark/Light/200% и hash debug APK. Это частичные свидетельства FR-001/065/081–084/090 и BL-008/110/120–123; TC-003/022/032 и полная Android-приёмка остаются NOT_RUN. HTTP fixtures не доказывают телефонный путь через реальный bridge; backend real-Codex проверяется отдельным протоколом.

## История

| Версия | Дата | Изменение |
|---|---|---|
| v0.1 | 10.10.2026 | Созданы32 [TC][g-acceptance],114 строк [FR][g-requirement-ids]/[NFR][g-requirement-ids] [RTM][g-traceability],70 обязательных [BL][g-work-ids]/[CW][g-work-ids] связей, [ПМИ][g-traceability] и форма фактического протокола |
| v0.2 | 12.10.2026 | Сверены фактические backend-срезы, D-22/Funnel и статусы; добавлен [план IP-01–08](implementation-plan.md). Обязательные требования/приоритеты сохранены; полная приёмка ещё не пройдена |
| v0.5 | 13.10.2026 | Локальный Flutter evidence без подмены Android acceptance |

[g-acceptance]: 07-glossary.md#term-acceptance

[g-accessibility]: 07-glossary.md#term-accessibility

[g-agent]: 07-glossary.md#term-agent

[g-api]: 07-glossary.md#term-api

[g-apk]: 07-glossary.md#term-apk

[g-app-lifecycle]: 07-glossary.md#term-app-lifecycle

[g-approval]: 07-glossary.md#term-approval

[g-audit]: 07-glossary.md#term-audit

[g-auth]: 07-glossary.md#term-auth

[g-budget]: 07-glossary.md#term-budget

[g-cancelled]: 07-glossary.md#term-cancelled

[g-chain]: 07-glossary.md#term-chain

[g-ci-cd]: 07-glossary.md#term-ci-cd

[g-config-hash]: 07-glossary.md#term-config-hash

[g-device-session]: 07-glossary.md#term-device-session

[g-event]: 07-glossary.md#term-event

[g-evidence]: 07-glossary.md#term-evidence

[g-export-import]: 07-glossary.md#term-export-import

[g-fencing]: 07-glossary.md#term-fencing

[g-fixture]: 07-glossary.md#term-fixture

[g-flutter]: 07-glossary.md#term-flutter

[g-github]: 07-glossary.md#term-github

[g-harness]: 07-glossary.md#term-harness

[g-hook]: 07-glossary.md#term-hook

[g-https]: 07-glossary.md#term-https

[g-identifier-formats]: 07-glossary.md#term-identifier-formats

[g-jwt-jwks]: 07-glossary.md#term-jwt-jwks

[g-main-roles]: 07-glossary.md#term-main-roles

[g-mcp]: 07-glossary.md#term-mcp

[g-message]: 07-glossary.md#term-message

[g-model]: 07-glossary.md#term-model

[g-model-protocol]: 07-glossary.md#term-model-protocol

[g-observability-tools]: 07-glossary.md#term-observability-tools

[g-outbox]: 07-glossary.md#term-outbox

[g-owner]: 07-glossary.md#term-owner

[g-package]: 07-glossary.md#term-package

[g-policy]: 07-glossary.md#term-policy

[g-priorities]: 07-glossary.md#term-priorities

[g-provider]: 07-glossary.md#term-provider

[g-push]: 07-glossary.md#term-push

[g-quality-scenario]: 07-glossary.md#term-quality-scenario

[g-readiness]: 07-glossary.md#term-readiness

[g-recovery]: 07-glossary.md#term-recovery

[g-repository]: 07-glossary.md#term-repository

[g-requirement-ids]: 07-glossary.md#term-requirement-ids

[g-result]: 07-glossary.md#term-result

[g-retry]: 07-glossary.md#term-retry

[g-review]: 07-glossary.md#term-review

[g-role]: 07-glossary.md#term-role

[g-run]: 07-glossary.md#term-run

[g-runner]: 07-glossary.md#term-runner

[g-runtime]: 07-glossary.md#term-runtime

[g-sandbox]: 07-glossary.md#term-sandbox

[g-secret]: 07-glossary.md#term-secret

[g-smart]: 07-glossary.md#term-smart

[g-sse]: 07-glossary.md#term-sse

[g-staging]: 07-glossary.md#term-staging

[g-step]: 07-glossary.md#term-step

[g-supabase]: 07-glossary.md#term-supabase

[g-task]: 07-glossary.md#term-task

[g-timeout]: 07-glossary.md#term-timeout

[g-tool]: 07-glossary.md#term-tool

[g-traceability]: 07-glossary.md#term-traceability

[g-ui]: 07-glossary.md#term-ui

[g-uml]: 07-glossary.md#term-uml

[g-usage]: 07-glossary.md#term-usage

[g-use-case]: 07-glossary.md#term-use-case

[g-work-ids]: 07-glossary.md#term-work-ids

[g-workspace]: 07-glossary.md#term-workspace

[id-ac-001]: 14-demo-acceptance-criteria.md#ac-001

[id-ac-002]: 14-demo-acceptance-criteria.md#ac-002

[id-ac-003]: 14-demo-acceptance-criteria.md#ac-003

[id-ac-004]: 14-demo-acceptance-criteria.md#ac-004

[id-ac-005]: 14-demo-acceptance-criteria.md#ac-005

[id-ac-006]: 14-demo-acceptance-criteria.md#ac-006

[id-ac-007]: 14-demo-acceptance-criteria.md#ac-007

[id-ac-008]: 14-demo-acceptance-criteria.md#ac-008

[id-ac-009]: 14-demo-acceptance-criteria.md#ac-009

[id-ac-010]: 14-demo-acceptance-criteria.md#ac-010

[id-ac-011]: 14-demo-acceptance-criteria.md#ac-011

[id-ac-012]: 14-demo-acceptance-criteria.md#ac-012

[id-ac-013]: 14-demo-acceptance-criteria.md#ac-013

[id-ac-014]: 14-demo-acceptance-criteria.md#ac-014

[id-ac-015]: 14-demo-acceptance-criteria.md#ac-015

[id-ac-016]: 14-demo-acceptance-criteria.md#ac-016

[id-ac-017]: 14-demo-acceptance-criteria.md#ac-017

[id-ac-018]: 14-demo-acceptance-criteria.md#ac-018

[id-ac-019]: 14-demo-acceptance-criteria.md#ac-019

[id-ac-020]: 14-demo-acceptance-criteria.md#ac-020

[id-bl-001]: 01-product-backlog.md#bl-001

[id-bl-002]: 01-product-backlog.md#bl-002

[id-bl-003]: 01-product-backlog.md#bl-003

[id-bl-004]: 01-product-backlog.md#bl-004

[id-bl-005]: 01-product-backlog.md#bl-005

[id-bl-006]: 01-product-backlog.md#bl-006

[id-bl-007]: 01-product-backlog.md#bl-007

[id-bl-008]: 01-product-backlog.md#bl-008

[id-bl-009]: 01-product-backlog.md#bl-009

[id-bl-010]: 01-product-backlog.md#bl-010

[id-bl-011]: 01-product-backlog.md#bl-011

[id-bl-012]: 01-product-backlog.md#bl-012

[id-bl-013]: 01-product-backlog.md#bl-013

[id-bl-014]: 01-product-backlog.md#bl-014

[id-bl-020]: 01-product-backlog.md#bl-020

[id-bl-021]: 01-product-backlog.md#bl-021

[id-bl-022]: 01-product-backlog.md#bl-022

[id-bl-023]: 01-product-backlog.md#bl-023

[id-bl-024]: 01-product-backlog.md#bl-024

[id-bl-025]: 01-product-backlog.md#bl-025

[id-bl-030]: 01-product-backlog.md#bl-030

[id-bl-031]: 01-product-backlog.md#bl-031

[id-bl-032]: 01-product-backlog.md#bl-032

[id-bl-033]: 01-product-backlog.md#bl-033

[id-bl-034]: 01-product-backlog.md#bl-034

[id-bl-035]: 01-product-backlog.md#bl-035

[id-bl-036]: 01-product-backlog.md#bl-036

[id-bl-040]: 01-product-backlog.md#bl-040

[id-bl-041]: 01-product-backlog.md#bl-041

[id-bl-042]: 01-product-backlog.md#bl-042

[id-bl-043]: 01-product-backlog.md#bl-043

[id-bl-044]: 01-product-backlog.md#bl-044

[id-bl-045]: 01-product-backlog.md#bl-045

[id-bl-050]: 01-product-backlog.md#bl-050

[id-bl-051]: 01-product-backlog.md#bl-051

[id-bl-052]: 01-product-backlog.md#bl-052

[id-bl-053]: 01-product-backlog.md#bl-053

[id-bl-054]: 01-product-backlog.md#bl-054

[id-bl-060]: 01-product-backlog.md#bl-060

[id-bl-061]: 01-product-backlog.md#bl-061

[id-bl-062]: 01-product-backlog.md#bl-062

[id-bl-063]: 01-product-backlog.md#bl-063

[id-bl-064]: 01-product-backlog.md#bl-064

[id-bl-065]: 01-product-backlog.md#bl-065

[id-bl-066]: 01-product-backlog.md#bl-066

[id-bl-067]: 01-product-backlog.md#bl-067

[id-bl-069]: 01-product-backlog.md#bl-069

[id-bl-070]: 01-product-backlog.md#bl-070

[id-bl-071]: 01-product-backlog.md#bl-071

[id-bl-072]: 01-product-backlog.md#bl-072

[id-bl-080]: 01-product-backlog.md#bl-080

[id-bl-081]: 01-product-backlog.md#bl-081

[id-bl-082]: 01-product-backlog.md#bl-082

[id-bl-083]: 01-product-backlog.md#bl-083

[id-bl-084]: 01-product-backlog.md#bl-084

[id-bl-090]: 01-product-backlog.md#bl-090

[id-bl-091]: 01-product-backlog.md#bl-091

[id-bl-100]: 01-product-backlog.md#bl-100

[id-bl-101]: 01-product-backlog.md#bl-101

[id-bl-102]: 01-product-backlog.md#bl-102

[id-bl-103]: 01-product-backlog.md#bl-103

[id-bl-104]: 01-product-backlog.md#bl-104

[id-bl-110]: 01-product-backlog.md#bl-110

[id-bl-111]: 01-product-backlog.md#bl-111

[id-bl-112]: 01-product-backlog.md#bl-112

[id-bl-113]: 01-product-backlog.md#bl-113

[id-bl-114]: 01-product-backlog.md#bl-114

[id-bl-115]: 01-product-backlog.md#bl-115

[id-bl-116]: 01-product-backlog.md#bl-116

[id-bl-117]: 01-product-backlog.md#bl-117

[id-bl-118]: 01-product-backlog.md#bl-118

[id-bl-119]: 01-product-backlog.md#bl-119

[id-bl-120]: 01-product-backlog.md#bl-120

[id-bl-121]: 01-product-backlog.md#bl-121

[id-bl-122]: 01-product-backlog.md#bl-122

[id-bl-123]: 01-product-backlog.md#bl-123

[id-bl-124]: 01-product-backlog.md#bl-124

[id-bl-125]: 01-product-backlog.md#bl-125

[id-cw-001]: 01-product-backlog.md#cw-001

[id-cw-002]: 01-product-backlog.md#cw-002

[id-cw-003]: 01-product-backlog.md#cw-003

[id-cw-004]: 01-product-backlog.md#cw-004

[id-doc-014]: 14-demo-acceptance-criteria.md

[id-doc-057]: 57-verification-and-traceability.md

[id-fr-001]: 03-software-requirements-specification.md#fr-001

[id-fr-002]: 03-software-requirements-specification.md#fr-002

[id-fr-003]: 03-software-requirements-specification.md#fr-003

[id-fr-004]: 03-software-requirements-specification.md#fr-004

[id-fr-005]: 03-software-requirements-specification.md#fr-005

[id-fr-006]: 03-software-requirements-specification.md#fr-006

[id-fr-007]: 03-software-requirements-specification.md#fr-007

[id-fr-008]: 03-software-requirements-specification.md#fr-008

[id-fr-009]: 03-software-requirements-specification.md#fr-009

[id-fr-010]: 03-software-requirements-specification.md#fr-010

[id-fr-011]: 03-software-requirements-specification.md#fr-011

[id-fr-012]: 03-software-requirements-specification.md#fr-012

[id-fr-013]: 03-software-requirements-specification.md#fr-013

[id-fr-014]: 03-software-requirements-specification.md#fr-014

[id-fr-015]: 03-software-requirements-specification.md#fr-015

[id-fr-016]: 03-software-requirements-specification.md#fr-016

[id-fr-017]: 03-software-requirements-specification.md#fr-017

[id-fr-018]: 03-software-requirements-specification.md#fr-018

[id-fr-019]: 03-software-requirements-specification.md#fr-019

[id-fr-020]: 03-software-requirements-specification.md#fr-020

[id-fr-021]: 03-software-requirements-specification.md#fr-021

[id-fr-022]: 03-software-requirements-specification.md#fr-022

[id-fr-023]: 03-software-requirements-specification.md#fr-023

[id-fr-024]: 03-software-requirements-specification.md#fr-024

[id-fr-025]: 03-software-requirements-specification.md#fr-025

[id-fr-026]: 03-software-requirements-specification.md#fr-026

[id-fr-027]: 03-software-requirements-specification.md#fr-027

[id-fr-028]: 03-software-requirements-specification.md#fr-028

[id-fr-029]: 03-software-requirements-specification.md#fr-029

[id-fr-030]: 03-software-requirements-specification.md#fr-030

[id-fr-031]: 03-software-requirements-specification.md#fr-031

[id-fr-032]: 03-software-requirements-specification.md#fr-032

[id-fr-033]: 03-software-requirements-specification.md#fr-033

[id-fr-034]: 03-software-requirements-specification.md#fr-034

[id-fr-035]: 03-software-requirements-specification.md#fr-035

[id-fr-036]: 03-software-requirements-specification.md#fr-036

[id-fr-037]: 03-software-requirements-specification.md#fr-037

[id-fr-038]: 03-software-requirements-specification.md#fr-038

[id-fr-039]: 03-software-requirements-specification.md#fr-039

[id-fr-040]: 03-software-requirements-specification.md#fr-040

[id-fr-041]: 03-software-requirements-specification.md#fr-041

[id-fr-042]: 03-software-requirements-specification.md#fr-042

[id-fr-043]: 03-software-requirements-specification.md#fr-043

[id-fr-044]: 03-software-requirements-specification.md#fr-044

[id-fr-045]: 03-software-requirements-specification.md#fr-045

[id-fr-046]: 03-software-requirements-specification.md#fr-046

[id-fr-047]: 03-software-requirements-specification.md#fr-047

[id-fr-048]: 03-software-requirements-specification.md#fr-048

[id-fr-049]: 03-software-requirements-specification.md#fr-049

[id-fr-050]: 03-software-requirements-specification.md#fr-050

[id-fr-051]: 03-software-requirements-specification.md#fr-051

[id-fr-052]: 03-software-requirements-specification.md#fr-052

[id-fr-053]: 03-software-requirements-specification.md#fr-053

[id-fr-054]: 03-software-requirements-specification.md#fr-054

[id-fr-055]: 03-software-requirements-specification.md#fr-055

[id-fr-056]: 03-software-requirements-specification.md#fr-056

[id-fr-057]: 03-software-requirements-specification.md#fr-057

[id-fr-058]: 03-software-requirements-specification.md#fr-058

[id-fr-059]: 03-software-requirements-specification.md#fr-059

[id-fr-060]: 03-software-requirements-specification.md#fr-060

[id-fr-061]: 03-software-requirements-specification.md#fr-061

[id-fr-062]: 03-software-requirements-specification.md#fr-062

[id-fr-063]: 03-software-requirements-specification.md#fr-063

[id-fr-064]: 03-software-requirements-specification.md#fr-064

[id-fr-065]: 03-software-requirements-specification.md#fr-065

[id-fr-066]: 03-software-requirements-specification.md#fr-066

[id-fr-067]: 03-software-requirements-specification.md#fr-067

[id-fr-068]: 03-software-requirements-specification.md#fr-068

[id-fr-069]: 03-software-requirements-specification.md#fr-069

[id-fr-070]: 03-software-requirements-specification.md#fr-070

[id-fr-071]: 03-software-requirements-specification.md#fr-071

[id-fr-072]: 03-software-requirements-specification.md#fr-072

[id-fr-073]: 03-software-requirements-specification.md#fr-073

[id-fr-074]: 03-software-requirements-specification.md#fr-074

[id-fr-075]: 03-software-requirements-specification.md#fr-075

[id-fr-076]: 03-software-requirements-specification.md#fr-076

[id-fr-077]: 03-software-requirements-specification.md#fr-077

[id-fr-078]: 03-software-requirements-specification.md#fr-078

[id-fr-079]: 03-software-requirements-specification.md#fr-079

[id-fr-080]: 03-software-requirements-specification.md#fr-080

[id-fr-081]: 03-software-requirements-specification.md#fr-081

[id-fr-082]: 03-software-requirements-specification.md#fr-082

[id-fr-083]: 03-software-requirements-specification.md#fr-083

[id-fr-084]: 03-software-requirements-specification.md#fr-084

[id-fr-085]: 03-software-requirements-specification.md#fr-085

[id-fr-086]: 03-software-requirements-specification.md#fr-086

[id-fr-087]: 03-software-requirements-specification.md#fr-087

[id-fr-088]: 03-software-requirements-specification.md#fr-088

[id-fr-089]: 03-software-requirements-specification.md#fr-089

[id-fr-090]: 03-software-requirements-specification.md#fr-090

[id-mx-01]: 04-smart-goals-and-metrics.md#mx-01

[id-nfr-001]: 03-software-requirements-specification.md#nfr-001

[id-nfr-002]: 03-software-requirements-specification.md#nfr-002

[id-nfr-003]: 03-software-requirements-specification.md#nfr-003

[id-nfr-004]: 03-software-requirements-specification.md#nfr-004

[id-nfr-005]: 03-software-requirements-specification.md#nfr-005

[id-nfr-006]: 03-software-requirements-specification.md#nfr-006

[id-nfr-007]: 03-software-requirements-specification.md#nfr-007

[id-nfr-008]: 03-software-requirements-specification.md#nfr-008

[id-nfr-009]: 03-software-requirements-specification.md#nfr-009

[id-nfr-010]: 03-software-requirements-specification.md#nfr-010

[id-nfr-011]: 03-software-requirements-specification.md#nfr-011

[id-nfr-012]: 03-software-requirements-specification.md#nfr-012

[id-nfr-013]: 03-software-requirements-specification.md#nfr-013

[id-nfr-014]: 03-software-requirements-specification.md#nfr-014

[id-nfr-015]: 03-software-requirements-specification.md#nfr-015

[id-nfr-016]: 03-software-requirements-specification.md#nfr-016

[id-nfr-017]: 03-software-requirements-specification.md#nfr-017

[id-nfr-018]: 03-software-requirements-specification.md#nfr-018

[id-nfr-019]: 03-software-requirements-specification.md#nfr-019

[id-nfr-020]: 03-software-requirements-specification.md#nfr-020

[id-nfr-021]: 03-software-requirements-specification.md#nfr-021

[id-nfr-022]: 03-software-requirements-specification.md#nfr-022

[id-nfr-023]: 03-software-requirements-specification.md#nfr-023

[id-nfr-024]: 03-software-requirements-specification.md#nfr-024

[id-qs-001]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-001

[id-qs-002]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-002

[id-qs-003]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-003

[id-qs-004]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-004

[id-qs-005]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-005

[id-qs-006]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-006

[id-qs-007]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-007

[id-qs-008]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-008

[id-qs-009]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-009

[id-qs-010]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-010

[id-qs-011]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-011

[id-qs-012]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-012

[id-qs-013]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-013

[id-qs-014]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-014

[id-qs-015]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-015

[id-qs-016]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-016

[id-qs-017]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-017

[id-qs-018]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-018

[id-qs-019]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-019

[id-qs-020]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-020

[id-qs-021]: 13-quality-attribute-scenarios-and-utility-tree.md#qs-021

[id-tc-001]: 57-verification-and-traceability.md#tc-001

[id-tc-002]: 57-verification-and-traceability.md#tc-002

[id-tc-003]: 57-verification-and-traceability.md#tc-003

[id-tc-004]: 57-verification-and-traceability.md#tc-004

[id-tc-005]: 57-verification-and-traceability.md#tc-005

[id-tc-006]: 57-verification-and-traceability.md#tc-006

[id-tc-007]: 57-verification-and-traceability.md#tc-007

[id-tc-008]: 57-verification-and-traceability.md#tc-008

[id-tc-009]: 57-verification-and-traceability.md#tc-009

[id-tc-010]: 57-verification-and-traceability.md#tc-010

[id-tc-011]: 57-verification-and-traceability.md#tc-011

[id-tc-012]: 57-verification-and-traceability.md#tc-012

[id-tc-013]: 57-verification-and-traceability.md#tc-013

[id-tc-014]: 57-verification-and-traceability.md#tc-014

[id-tc-015]: 57-verification-and-traceability.md#tc-015

[id-tc-016]: 57-verification-and-traceability.md#tc-016

[id-tc-017]: 57-verification-and-traceability.md#tc-017

[id-tc-018]: 57-verification-and-traceability.md#tc-018

[id-tc-019]: 57-verification-and-traceability.md#tc-019

[id-tc-020]: 57-verification-and-traceability.md#tc-020

[id-tc-021]: 57-verification-and-traceability.md#tc-021

[id-tc-022]: 57-verification-and-traceability.md#tc-022

[id-tc-023]: 57-verification-and-traceability.md#tc-023

[id-tc-024]: 57-verification-and-traceability.md#tc-024

[id-tc-025]: 57-verification-and-traceability.md#tc-025

[id-tc-026]: 57-verification-and-traceability.md#tc-026

[id-tc-027]: 57-verification-and-traceability.md#tc-027

[id-tc-028]: 57-verification-and-traceability.md#tc-028

[id-tc-029]: 57-verification-and-traceability.md#tc-029

[id-tc-030]: 57-verification-and-traceability.md#tc-030

[id-tc-031]: 57-verification-and-traceability.md#tc-031

[id-tc-032]: 57-verification-and-traceability.md#tc-032

[id-uc-001]: 12-use-case-diagrams-and-specifications.md#uc-001

[id-uc-002]: 12-use-case-diagrams-and-specifications.md#uc-002

[id-uc-003]: 12-use-case-diagrams-and-specifications.md#uc-003

[id-uc-004]: 12-use-case-diagrams-and-specifications.md#uc-004

[id-uc-005]: 12-use-case-diagrams-and-specifications.md#uc-005

[id-uc-006]: 12-use-case-diagrams-and-specifications.md#uc-006

[id-uc-007]: 12-use-case-diagrams-and-specifications.md#uc-007

[id-uc-008]: 12-use-case-diagrams-and-specifications.md#uc-008

[id-uc-009]: 12-use-case-diagrams-and-specifications.md#uc-009

[id-uc-010]: 12-use-case-diagrams-and-specifications.md#uc-010

[id-uc-011]: 12-use-case-diagrams-and-specifications.md#uc-011

[id-uc-012]: 12-use-case-diagrams-and-specifications.md#uc-012

[id-uc-013]: 12-use-case-diagrams-and-specifications.md#uc-013

[id-uc-014]: 12-use-case-diagrams-and-specifications.md#uc-014

[id-uc-015]: 12-use-case-diagrams-and-specifications.md#uc-015

[id-uc-016]: 12-use-case-diagrams-and-specifications.md#uc-016

[id-uc-017]: 12-use-case-diagrams-and-specifications.md#uc-017

[id-uc-018]: 12-use-case-diagrams-and-specifications.md#uc-018

[id-uc-019]: 12-use-case-diagrams-and-specifications.md#uc-019

[id-uc-020]: 12-use-case-diagrams-and-specifications.md#uc-020
