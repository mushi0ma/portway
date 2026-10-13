# Ручной протокол FCM для владельца Portway

Статус: **протокол подготовлен; реальные TC-012 / TC-032 NOT_RUN**. Телефон и APK недоступны в испытании. Владелец приёмку не подтверждал. Дата подготовки: 13.10.2026, Asia/Almaty (UTC+05:00).

## Текущая готовность и граница сценария

Локально прочитаны README, status/plan, DOC-052, TC-012/032 и ПМИ DOC-057, AC-012 DOC-014, DOC-061 §§4/7/8 и контракт DOC-036. Текущий статус: Android/FCM ещё не реализованы (IP-06 TODO). Уточнение владельца: есть Figma-дизайн, Flutter-клиента ещё нет. В OpenAPI и `internal/transport/server.go` есть login/refresh/device/approval read/decision; **нет push-token routes и delivery worker-контракта в реализованном API**. Description OpenAPI сохраняет FCM unimplemented. Публичный run admission закрыт, поэтому создание pending через обычный пользовательский run сейчас не готово.

Это проверка кода/контракта, **не проверка доступности публичного endpoint**: сеть, FCM-проект, backend, установленный APK и Firebase client config не опрашивались. Существование URL в status не доказывает готовность FCM.

Уровень: A/R. Связи: TC-012, выбранные push-ветви TC-032; AC-012; FR-075/076/086 и logout/device FR-003/085; NFR-011/019; BL-053/116/125. Этот протокол не закрывает весь TC-032, полный Android-путь, SSE-p95 или исполнение tool.

## Условия начала — заполнить до первого trial

| Gate | Требование | Сейчас |
|---|---|---|
| Candidate | Backend и Flutter revision, dirty/config fingerprint, APK version/SHA-256/signing identity, OpenAPI/DB migration/image versions закреплены | APK отсутствует; candidate не закреплён |
| Телефон | Физический Android **16**; модель, Android/API level, build, Google Play Services version, сеть, battery/background settings записаны | Телефон недоступен |
| Установка | HTTPS Go API достижим с телефона; installation_id/name совпадают с APK; owner confirmed, device active; auth/refresh работает | На телефоне не проверено |
| Push | Реальный FCM project и client config соответствуют APK; permission granted; authenticated device token registered; worker умеет отправлять | FCM и routes ещё отсутствуют |
| Pending fixture | Документированный безопасный способ на тестовом backend породить отдельный pending approval с server TTL 180 с и реальный FCM; точные fixture revision/операция записаны | Такой способ в текущем публичном API не готов |
| Lifecycle | Способ получить и независимо подтвердить background/normal terminated; безопасная диагностика не раскрывает секреты | Не проверено |
| Время | Server/phone UTC, offset/погрешность, способ получения receipt/tap timestamps, запись экрана и безопасные диагностические trace | Не подготовлено |

Если gate отсутствует, оставить trial `NOT_RUN`, указать blocker. Не создавать локальное уведомление вместо FCM, не править production DB/часы, не придумывать public endpoint генерации approval. Тестовый backend заранее документирует fixture-операцию; она создаёт pending, а не самовольно исполняет команду. Никаких новых подключений/действий этим документом не выполнено. ADB не используется автоматически.

## Роли и подготовка владельца

Владелец выполняет действия на телефоне. Оператор тестового backend создаёт только согласованные fixture approvals и собирает безопасные server traces. Исполнитель фиксирует наблюдения/evidence. Это роли; их может выполнять один владелец, второй человек не требуется.

1. Зафиксировать candidate и среду, открыть UI-02 с installation name, войти подтверждённым owner. Сохранить безопасный trace auth/device без пароля, JWT, device proof или FCM registration token.
2. На UI-17 проверить объяснение назначения уведомлений и разрешить системный permission. Backend подтверждает binding к текущим owner/device; evidence содержит только факт/время/псевдоним устройства, не token value.
3. Подготовить отдельный тестовый run/approval fixture с известными неизменяемыми безопасными args. Для этого FCM-теста достаточно открыть детали и **отклонить** после проверки; разрешение произвольной shell-команды из задачи не следует.
4. Backend записывает `created_at`, `sent_at`, `expires_at`, состояние pending и safe approval/run IDs. Deadline для основных trials — TTL180 с; FCM message TTL не больше остающегося server deadline. Payload — notification+data только с типом/installation_id/approval_id, без args, secrets и прямой approve-кнопки.

## Основная выборка — 3 background + 3 normal terminated

Каждый trial использует новый approval. Между trials подтвердить его терминальный исход/очистку и вернуть приложение в исходное состояние. Foreground и force-stop в эти шесть строк не входят.

| Шаг | Владелец | Тестовый backend | Наблюдает/записывает исполнитель |
|---|---|---|---|
| 1. Lifecycle | BG: обычный Home. TERM: согласованный способ обычного завершения процесса, **без force-stop через настройки** | Пока не отправляет | Способ и timestamp, доказательство состояния до send |
| 2. Создание | Оставляет APK в указанном состоянии | Создаёт новый pending, фиксирует deadline, отправляет через настоящий FCM | IDs, server send/accept/error, FCM TTL; accepted отдельно от delivery |
| 3. Получение | Наблюдает системное уведомление | Не подменяет его вторым каналом | Receipt timestamp и источник, notification ID/запись экрана; receipt до server deadline |
| 4. Tap | Нажимает до expiry | Обрабатывает обычные защищённые запросы | Tap timestamp; coordinator: destination → auth/refresh при необходимости → GET latest → UI-12 |
| 5. Детали | Читает актуальные args/state/deadline | Возвращает текущую version/args_hash/state | Проверка installation/approval, содержимое из GET; stale payload не используется как authority |
| 6. Завершение | По необходимости «Отклонить» на актуальном pending | CAS decision, один ACK, без tool dispatch | Server ACK отдельно от tool outcome; повторный tap не даёт второго решения |

Для normal terminated свайп карточки сам по себе **не доказывает смерть процесса**. Перед отправкой нужно независимое доступное свидетельство отсутствия процесса/нового process start при tap из безопасной диагностики candidate или согласованного наблюдения ОС. Метод и ограничения записываются. Если состояние нельзя подтвердить, строка `INCONCLUSIVE`, а не подтверждённый terminated; нельзя переименовать background в terminated. Если диагностика требует дополнительных прав/ADB, она готовится отдельно с владельцем, здесь не запускается.

Ни одно измерение не заполнять по памяти. Server send, provider accepted, phone receipt, tap и GET являются разными событиями. Отсутствие receipt trace/неопределённое lifecycle не превращается в PASS только по снимку открытого UI-12.

| Trial | Lifecycle / доказательство | Approval ID | Created/send/deadline/FCM TTL | Receipt/tap / источник/offset | Auth + GET latest/state | Outcome |
|---|---|---|---|---|---|---|
| BG-1 | NOT_RUN | — | — | — | — | NOT_RUN: нет APK/телефона/FCM |
| BG-2 | NOT_RUN | — | — | — | — | NOT_RUN: нет APK/телефона/FCM |
| BG-3 | NOT_RUN | — | — | — | — | NOT_RUN: нет APK/телефона/FCM |
| TERM-1 | NOT_RUN | — | — | — | — | NOT_RUN: нет APK/телефона/FCM |
| TERM-2 | NOT_RUN | — | — | — | — | NOT_RUN: нет APK/телефона/FCM |
| TERM-3 | NOT_RUN | — | — | — | — | NOT_RUN: нет APK/телефона/FCM |

Oracle выборки по TC-012/ПМИ: **6/6 реальных доставок до server approval TTL180 с**, с подтверждённым lifecycle и auth→GET latest. Это критерий испытания, не SLA FCM. Время/ошибка часов должны позволять однозначно установить порядок относительно deadline; иначе результат INCONCLUSIVE. Крайние ветви ниже не добавляются в знаменатель шести trials.

## Отдельные края TC-012/032 и AC-012

Все строки ниже сейчас NOT_RUN; каждую провести с отдельной fixture и зафиксировать actual.

| Сценарий | Действия владельца / backend | Expected |
|---|---|---|
| Foreground | Оставить APK открытым, backend создаёт pending+FCM | Актуальная лента/состояние обновлены, нет дублированного уведомления |
| Expired tap | Получить notification, дождаться server expiry, затем tap | GET latest показывает expiry/исход, старого рабочего approve нет; marker исполнения отсутствует |
| Repeat tap | Открыть одно уведомление повторно после решения/expiry | Одно решение; latest state; нет второго dispatch |
| Expired auth | Истёкшая access session, pending destination; tap | Refresh либо явный вход до GET; при отсутствии действующего доступа кнопки disabled |
| Revoked device | С другого допустимого owner device отозвать тестовое; tap старого notification | Отказ/объяснение отзыва, нет действия; binding push inactive/unlinked |
| Другая installation | Допустимая тестовая pending destination от A при активном контексте B | Нельзя принять A за B или применить args из payload; явная смена установки и auth по DOC-061 либо безопасный отказ |
| Permission deny → grant | На чистом допустимом состоянии отказать; затем включить в системных настройках | Отказ не блокирует обычный вход; объяснено отсутствие фоновых alerts; после grant binding актуален |
| Token rotation | Поддерживаемым способом fixture/candidate вызвать реальную rotation | Сервер обновлён, старая запись inactive, notification приходит на текущую; raw tokens не публикуются |
| Logout unlink | Выполнить logout и проверить durable backend outcome | Local logout и server revoke различимы; confirmed logout отвязывает push; offline показывает неподтверждённый отзыв |
| Ошибка отправки | Backend допускает контролируемую ошибку FCM без повреждения рабочей установки | Ошибка наблюдаема в status/audit с причиной; provider accepted не выдаётся за delivered |
| Force-stop | Только отдельный край, способ явно «Настройки Android → Остановить» | По DOC-061 гарантии доставки нет до ручного открытия; не смешивать с normal terminated и не имитировать notification |

## Evidence, итог и повтор

Форма соответствует DOC-057 §7 и `references/evidence-template.md`. Заполнить actual после устройства:

```text
protocol_id / TC-subset / AC / FR-NFR / BL:
started_at_utc / ended_at_utc / local_zone / executor / owner_confirmation:
backend_revision / flutter_revision / dirty+safe-diff fingerprint:
APK filename / SHA-256 / version / signing identity:
image digests / DB migration / OpenAPI+safe config fingerprint:
installation label / device label / phone model / Android+API / Play Services:
permission / network / battery-background settings / lifecycle proof:
fixture revision / run ID / approval ID / exact fixture operation:
created_at / FCM send / provider accept-or-error / expires_at / FCM TTL:
phone receipt / tap / auth+GET timestamps / clocks offset+uncertainty:
expected / actual / server ACK / separate tool outcome-or-NOT_TESTED:
artifact relative paths / hashes / redactions:
outcome / defect-or-blocker / rerun link / owner decision:
```

Для каждой строки — экранная запись фактического lifecycle/уведомления/tap, безопасный client trace GET latest, server trace deadline/send/state/decision. Скриншоты и logs редактировать без потери корреляции; не сохранять credentials, auth bodies, callback token_hash, device proof, FCM token и чувствительные payload. Будущие filenames/URLs не обозначать существующими evidence. Исходные timestamps сохраняются, не только сводный счётчик.

PASS основной выборки возможен только при 6/6. Полный TC/AC не повышается по выборке без обязательных краёв и достаточных свидетельств. FAIL сохраняется с defect; повтор имеет новые IDs и связан с исходным. После значимой revision/config повторяются затронутые проверки и candidate-прогон по DOC-057/014. Решение владельца заполняется только после его фактического подтверждения.

**Фактически выполнено здесь:** чтение/сопоставление текущих TC/AC, lifecycle, deadline и payload-контракта; проверка наличия маршрутов в OpenAPI/коде; составление шести строк и edge-протокола. **Не выполнено:** ни один FCM send/receipt/tap, API request, server fixture, установка APK, устройство или owner acceptance. Workspace только читался.

Источники (корень `/home/mushi/Workspace/Portway`): `skills/portway-android-acceptance/SKILL.md`; `references/device-scenarios.md`, `references/evidence-template.md`; `README.md`; `docs/implementation-status.md`; `docs/implementation-plan.md`; `docs/52-build-deploy-and-operations.md` §§1/4/6/7; `docs/57-verification-and-traceability.md` TC-012/032, §§2/4/7; `docs/14-demo-acceptance-criteria.md` §§2/5 и AC-012; `docs/61-android-screens-flows-and-api.md` §§4/7/8; `docs/36-api-events-and-messages.md` §§2/5; `api/openapi.json`; `internal/transport/server.go`, `internal/transport/execution.go`, `internal/execution/http.go`. Внешние ссылки Firebase из DOC-061 не открывались; актуальные SDK-особенности не заявляются проверенными.
