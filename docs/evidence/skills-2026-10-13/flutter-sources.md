# Источники и фактические границы проверки

Дата: 13.10.2026. Корень источников: `/home/mushi/Workspace/Portway`. Применён навык `skills/portway-flutter-conventions/SKILL.md` и его `references/client-contracts.md` как исполнитель двух задач будущего клиента. Оценка качества/переписывание навыка не выполнялись.

## Результаты

- `ui12-approval-controller.md`: explicit controller phases/transitions; отдельные server approval/tool outcome; lost-POST recovery с original key/body; resume/time offset; три значения изменившегося deadline; план A01–A15.
- `ui04-session-navigation.md`: login exception; isolated installations и epoch; teardown/revoke/tombstone; auth→GET latest→navigation; mismatched old push; план S01–S18.
- `contract-evidence.json`: результаты локального сопоставления OpenAPI/code по выбранным контрактным границам. Это evidence чтения/проверки границ, не выполненные тесты клиента/серверной интеграции.

## Прочитанные источники

| Источник | Прочитанная область / роль |
|---|---|
| `skills/portway-flutter-conventions/SKILL.md` | Целиком; последовательность источников, слои/неопределённость/auth exception, тестовые и фактические границы |
| `skills/portway-flutter-conventions/references/client-contracts.md` | Целиком; профильная карта auth/approval/push/архитектуры |
| `README.md` | Обзор реализации, identity, auth callback, доступные API и ограничения |
| `docs/implementation-status.md` | Целиком; частичная реализация identity/approvals, отсутствие Android/FCM/Runtime |
| `docs/implementation-plan.md` | Целиком; IP-06/приёмка, fixtures не равны production |
| `docs/61-android-screens-flows-and-api.md` | §1–8; UI-02/04/11/12/17, flow, states, background/push, доступность |
| `docs/20-architecture-and-decisions.md` | §8 и найденные связанные ADR/сервисные границы; один coordinator и явные controllers |
| `docs/36-api-events-and-messages.md` | §1–6; HTTP/idempotency/auth exception, login/refresh/logout, approvals, SSE |
| `docs/25-approval-workflow.md` | Целиком; реализованный CAS/TTL/stop/replay и пределы Runtime/FCM |
| `docs/29-data-and-state-model.md` | Installation scope, atomic rules; §5–6 run/approval/tool states и серверное время |
| `docs/40-agent-execution-and-security.md` | Identity/secrets §8, threats §9, recovery §10 и связанные найденные approval/deadline границы |
| `docs/03-software-requirements-specification.md` | FR-001–003,038–043,065–070,075/076,081–086/090; defaults §4.2 и мобильная приёмка |
| `docs/57-verification-and-traceability.md` | Таблица TC, особенно 002/003/011/012/021/022/029/032; измерения §4 и статусы трассировки |
| `api/openapi.json` | Локальный JSON parse; выбранные client-config/auth/me/devices/approval routes, schemas Approval/Decision/LoginResult/Session/Meta/Error/security |
| `internal/execution/approvals.go` | Целиком; immutable proposal +180s, ResolveApproval, реальные codes/details, expiry/cancel |
| `internal/execution/approval_store.go` | Целиком; GET expiry reconcile, transactional decision, replay возвращает current tool state |
| `internal/execution/store.go` | begin/replay/remember и listing: DB clock, installation lock/scope, key validation |
| `internal/execution/http.go` | Approvals handlers и envelope приватного сервиса |
| `internal/transport/execution.go` | Целиком; public DTO, key validation, bounded proxy и неопределённый dependency outcome |
| `internal/transport/identity.go` | Целиком; login/refresh/logout/client-config/me/devices/revoke |
| `internal/transport/server.go` | Регистрация маршрутов и respond envelope; push-token routes отсутствуют, meta.server_time |
| `internal/transport/json.go` | Делегирование Decode строгому JSON слою |
| `internal/identity/service.go` | Типы LoginResult/Profile и Login/device/Authenticate/Refresh/Logout/Revoke |
| `internal/identity/supabase.go` | Session DTO и adapter, поиск login/refresh/logout |
| `internal/execution/approvals_test.go` | Прочитанные domain tests: граница expiry, conflicting args/version, replay/stop, budget/deadline/restart |
| `tests/integration/approvals_test.go` | Целиком; существующие race/replay/stop/422 persistence/foreign owner проверки; не запускались |
| `tests/integration/backend_test.go` | Auth fixtures/части регистрации→callback→login→refresh→me/revoke/logout и safe-log marker tests; не запускались |

Предковый/локальный применимый AGENTS.md в корне Portway/родительских каталогах не найден. Файлы `.env` и credentials не читались. Никакие реальные адреса API из README/status не вызывались. Репозиторий оставался read-only; артефакты записаны исключительно в `/tmp/portway-skill-eval-flutter`.

## Принятые решения и расхождения контрактов

1. Для клиента используются фактические wire enum/code из OpenAPI+handlers. Conceptual `invalidated` — UI смысл для `cancelled`+reason; generic `APPROVAL_RESOLVED` не подменяет реальные `APPROVAL_ALREADY_RESOLVED/ARGUMENTS_CHANGED/INVALIDATED`.
2. Неизвестный POST решения сохраняет immutable operation; GET pending не доказывает rollback. Только исходный key/body после revalidation; новый key/противоположное решение под unknown запрещены.
3. Срок апрува не стартует заново при resume. Для same-ID expiry extension текущего протокола нет; это mismatch, а не новая клиентская функция. Clock offset, approval expiry и run/chain deadline не смешиваются.
4. Login/refresh не повторяют credential response через request keys. Потерянный полный login без credentials может оставить серверное устройство, которое клиент не умеет отозвать без proof; это отражено как ограничение.
5. Один coordinator с immutable context/epoch отбрасывает late callbacks A. Старый A push при активной B не выбирает URL, не отправляет credentials и не восстанавливает маршрут A.
6. Push-token endpoint и FCM lifecycle пока проектные: отсутствуют в OpenAPI/router/code storage. Logout подтверждает device revoke, не доказанный push unlink.

## Что проверено / не проверено

Выполнены чтение актуальных локальных контрактов, локальный JSON parse OpenAPI и сопоставление выбранных DTO/routes/security/error branches. Результаты наблюдений записаны в `contract-evidence.json`; это не отдельный автоматический тестовый прогон. Существующие backend tests прочитаны как evidence контрактов, заново не запускались.

Flutter binary найден: `/home/mushi/.flutter/bin/flutter`. В Portway не найдены pubspec.yaml или Dart-файлы; владелец подтвердил отсутствие клиента. SDK version/doctor не запускались, package format/analyze/unit/widget/golden/APK/device tests недоступны без клиентского пакета. Установка/сборка/выбор SDK/API/новых библиотек в эту задачу не входили, внешняя документация и сеть не вызывались. Figma также не открывалась.

Схемы и планы проверок не являются проверенной сборкой, backend integration PASS или приёмкой TC/AC. Docs/status/plan/backlog не обновлялись по прямому ограничению этой независимой задачи; необходимые будущие изменения и evidence должны быть зафиксированы при реальной реализации клиента.
