# UI-12: решение, timeout, возврат из фона и серверный срок

Дата: 13.10.2026. Результат — схема будущего controller и сценарии проверки, без реализации Flutter/APK. Клиент ещё не реализован; по уточнению владельца существует дизайн в Figma, который в этой задаче не инспектировался. Все нижеописанные тесты — план, НЕ выполненные тесты клиента.

## Основания и границы

- DOC-061 §4: server-time offset; GET после фона/сомнения; во время POST обе кнопки заблокированы; потеря ответа → GET latest и при необходимости исходный key; server ACK решения отделён от выполнения инструмента.
- DOC-036 §1/5, OpenAPI: POST `/v1/approvals/{id}/decision`, заголовки `Authorization`, `X-Device-Session`, `Idempotency-Key` (8–200 символов); body ровно `{decision, expected_version, args_hash}`. Ключ относится к installation+owner+operation, окно 7 суток; фактическое operation в execution — `approval-decision/{id}`. `If-Match` для этой операции не предусмотрен: её CAS — `expected_version` и `args_hash`.
- DOC-025 и `internal/execution/approvals.go`: создание закрепляет argv/working_dir/timeout/hash/policy/epochs; expiry = server creation + 180 секунд. При `now == expires_at` уже поздно. Approval, tool outcome и Runtime grant — разные сущности.
- DOC-020 §8: widget → controller → repository → API; один session coordinator. Здесь проектируются библиотечно независимые состояния и интерфейсы; выбор пакетов состояния/навигации/DI и изменение ADR не выполняются.

## Серверная модель, которую отображает controller

`Approval.state` по фактическому OpenAPI: `pending | approved | rejected | expired | cancelled`. Conceptual `invalidated` в DOC-029/DOC-061 отображается из wire `cancelled` и `reason`, а не отправляется/ожидается как шестой enum.

`tool_state` — отдельное поле: `awaiting_approval | ready | denied | cancelled | dispatched | completed | failed | unknown | timeout`. UI никогда не выводит «Инструмент выполнен» из одного `approved`/`ready`. Неизвестный результат POST решения и `tool_state=unknown` — разные оси состояния.

| Серверный переход | Что принимается как итог UI-12 |
|---|---|
| pending → approved | «Разрешено» после ответа сервера; текущий tool_state показывается отдельно |
| pending → rejected | «Отклонено», tool=denied, кнопок старого решения нет |
| pending → expired | «Срок подтверждения истёк», tool=denied, причина/контекст задачи сохранены |
| pending → cancelled | «Запрос больше недействителен», безопасная причина stop/restart/deadline/policy/budget |
| approved + последующий stop | Approval остаётся approved; ещё не выполненный ready tool становится cancelled; replay не оживляет tool |

GET detail сам сверяет expiry. Истечение локального таймера не является серверным переходом. Уже approved не становится expired только потому, что часы перешли `expires_at`.

## Состояние будущего ApprovalController

Контекст экземпляра: `{baseUrl, installationId, ownerId, sessionEpoch, approvalId}`. Каждый async result проверяет этот контекст и номер запроса. Session epoch меняется при logout/revoke/смене установки; ответы старого контекста отбрасываются.

Данные: последний серверный `Approval`, `checkedAt`, `serverClockSample`, признак revalidation после resume, текущий `operation?`. Operation создаётся один раз на явное решение пользователя и содержит исходные `{key, approvalId, decision, expected_version, args_hash}` и scope установки/owner. Эти значения при retry не переписываются свежей версией или hash. Ни key, ни cached DTO не являются полномочием.

| Controller phase | Допустимые действия / текст |
|---|---|
| loading | Начальный GET; skeleton, не ложное empty |
| ready | Свежий pending; обе кнопки доступны только через единый guard ниже |
| revalidating | GET перед решением / после resume / после timeout; обе кнопки выключены |
| submitting | Один POST исходной operation; «Отправляем решение»; обе кнопки выключены |
| outcomeUnknown | Ответ POST потерян или транспорт/прокси не подтвердил исход; «Решение отправлено, результат пока не подтверждён»; нельзя выбирать противоположное/новое решение |
| terminal | Подтверждённый approved/rejected/expired/cancelled; причина, tool outcome и переход к задаче; старых кнопок нет |
| unavailable | GET не удался; last checked/cached помечены; повтор только проверки; operation сохраняется, если исход неизвестен |
| authRequired / forbidden / notFound | Доступ отозван/нужен вход/403/404; старые controls выключены; auth передан session coordinator |
| contractMismatch | Неподдерживаемая схема, противоречащие immutable данные; доступен GET/контекст, решения заблокированы |
| disposed | UI/repository подписки закрыты; поздний ответ ничего не меняет |

`lifecycle = foreground | background` — отдельная ось, не бизнес-состояние. В фоне не вводится новая mutation/retry. Уход в фон не означает rollback уже отправленного POST. Operation переживает обычный background в памяти coordinator/controller; никакой постоянной offline-очереди решений не создаётся. После смерти процесса — auth + GET latest; неизвестное старое решение не восстанавливается как автоматически исполняемая команда.

Guard обеих кнопок: foreground; актуальная сессия той же установки/owner; законченный GET текущего поколения после resume; валидный pending DTO; нет submitting/unknown operation; серверный срок не истёк и его оценка не сомнительна; нет известной блокировки run/chain. Сервер повторяет свои auth/policy/epoch/deadline/budget проверки независимо от guard.

## Переходы и действия

| Событие | Из → в | Действие и запрет |
|---|---|---|
| open / resume | любое живое → revalidating | Сразу закрыть controls; coordinator восстанавливает доступ; GET latest текущего approval |
| GET pending, operation отсутствует | revalidating → ready | Обновить детали/clock sample; только затем новые кнопки |
| явный tap approve/reject | ready → revalidating | Preflight GET. Если те же pending/hash/version/условия — создать key и immutable operation, перейти submitting. Если детали изменились — показать свежие детали и требовать нового явного выбора; старый tap не становится согласием на новые условия |
| duplicate tap | revalidating/submitting/unknown → то же | Второго POST/operation нет |
| POST 200 | submitting → terminal | Применить текущий Approval, а не локально выбранный статус; удалить operation после подтверждённого исхода |
| timeout/обрыв/неопределённый 503 после отправки | submitting → outcomeUnknown → revalidating | Сохранить исходный key/body; выполнить GET latest, не предполагать «не отправилось» |
| GET terminal после timeout | revalidating → terminal | Это подтверждение текущего серверного исхода; повтор не нужен. Если actor другой — не утверждать, что именно этот POST был победителем |
| GET pending после timeout | revalidating → outcomeUnknown | Pending не доказывает, что старый запрос больше не может commit. При необходимости допустим только повтор той же operation, тем же key/body, после проверки доступа/срока/идентичности; вернуться submitting. Нельзя создать новый key или противоположное решение |
| GET ошибся после timeout | revalidating → unavailable(outcomeUnknown) | Сохранить неопределённость и operation; «Проверить состояние», без кнопок решений |
| локальное remaining ≤0 | ready/unknown → revalidating | Controls выключить; запросить сервер. До ACK текст о сроке по последним данным, не окончательный expired |
| SSE approval.resolved / изменённый run | любое → revalidating | Dedupe event ID/seq, GET latest; событие не запускает approve и не заменяет восстановление после lost response |
| revoke/logout/смена установки | любое → authRequired/disposed | Coordinator закрывает потоки и действие; operation A никогда не переходит в B |
| поздний ответ старого GET/POST | любое → то же | Отбросить неверный epoch/request generation; меньшая version не возвращает terminal к pending |

Повтор исходной operation — восстановление уже выраженного намерения, а не новая команда исполнения инструмента. Он допустим при необходимости после GET, не запускается постоянной фоновой очередью. Если нет уверенности, сохраняется unknown и доступен read-only retry. Даже при истечении 7 суток нельзя «обойти» неопределённость новым key.

Ответы 409/422 применяются по фактическому `error.code` и `error.details.approval` при наличии; после неоднозначности перечитывается detail. Реальные коды: `APPROVAL_ARGUMENTS_CHANGED`, `VERSION_CONFLICT`, `APPROVAL_ALREADY_RESOLVED`, `APPROVAL_INVALIDATED`, `IDEMPOTENCY_CONFLICT`; 422 `APPROVAL_EXPIRED`, `BUDGET_EXHAUSTED`. DOC-036 обобщает часть конфликтов как `APPROVAL_RESOLVED`; клиент не должен ожидать исключительно это имя. 422 expiry сохраняется серверной транзакцией, не откатывается вместе с error. 401 передаётся coordinator; refresh не разрешает слепой повтор mutation с новым key. 403/404 не трактуются как успех.

## Возврат из фона и изменившийся deadline

1. Сначала revalidating с выключенными кнопками. Старый ticker останавливается/теряет авторитет. Доступ проверяет coordinator, затем GET detail.
2. Detail содержит `expires_at`, время берётся из публичного `meta.server_time`; `data.server_time` предусмотрен дополнительно в list, в detail его нет. `server_time` относится к envelope control API; это не доказательство идеальной синхронизации часов DB/execution/control.
3. Для отображения фиксируются server time и monotonic момент получения. Оценка offset по времени request/response может учитывать RTT; uncertainty должна включать неизвестную сетевую/серверную погрешность. Время после anchor считается monotonic, ручной перевод часов телефона не продлевает доступ. При сомнении/около границы — GET и disabled controls, без выдуманного миллисекундного допуска.
4. `remaining = max(0, expires_at - estimatedServerNow)`. После GET применяется новая server-time оценка, а не оставшиеся секунды до background. Например, было показано 90 секунд, fresh server_time даёт 20 — показываются 20; 180 секунд заново не запускаются.
5. Если под изменением deadline понимается run/chain deadline, это отдельная сущность. Detail Approval не содержит этих сроков; контекст берётся из GET run/chain. Известный истёкший run/chain запрещает controls, а сервер вернёт invalidation. Сроки run/chain не изменяют approval TTL и не возобновляют terminal работу.
6. Если сервер прислал **другой `expires_at` того же approval ID**, текущий контракт не описывает продление: создание фиксирует +180 секунд, endpoint renewal отсутствует. Latest срок показывается как серверное значение, но pending решение переводится в contractMismatch/revalidation, автоматический replay/новый key запрещены. Подтверждённый terminal ответ всё равно показывается. Это выявленная граница контракта, не повод клиенту изобрести «продлить апрув». Новый approval ID — другой запрос, требующий нового чтения и явного решения; unknown operation старого ID к нему не переносится.

## План проверок

Будущие unit-тесты controller/coordinator — на fake clock и детерминированном repository; widget/golden/integration только после появления Flutter-пакета. Fixtures явно помечаются, не выдаются за live API.

| ID | Вход/последовательность | Обязательный oracle |
|---|---|---|
| A01 | pending v1; tap approve дважды | Один key/один immutable body; обе кнопки disabled уже при preflight и во время POST |
| A02 | POST committed approved, ответ потерян, background, resume, GET approved/tool=ready | GET после resume; terminal «Разрешено», нет второго POST и нет «Выполнен» |
| A03 | POST не завершён, GET pending, необходим replay | GET pending не объявляет failure; повтор побайтно/канонически того же body и исходного key; один логический исход |
| A04 | A03, старый POST commit происходит после GET pending | Same-key replay возвращает current outcome; нет второго решения, нового key или смены actor в уже сохранённом решении |
| A05 | POST lost, GET недоступен/503 | unknown сохраняется; повтор проверки доступен, approve/reject недоступны; last checked виден |
| A06 | resume: expires_at прежний, offset уменьшил остаток 90→20 | Новый countdown 20, не 90/180; до GET действия закрыты; перевод часов телефона не продлевает срок |
| A07 | clock на `expires_at−ε`, затем ровно expires_at и после | До границы сервер может принять; на/после границы 422 + error.details.approval.state=expired, обе кнопки закрыты; local zero сам по себе не ACK |
| A08 | POST approved до expiry, ACK lost; GET после expiry | Показывается approved, не expired; поздний таймер не переписывает решение |
| A09 | approved, затем stop/restart, replay исходного approve | State остаётся approved, current tool=cancelled; никаких ready/dispatch из cached response |
| A10 | Другое устройство выигрывает reject; наш approve timeout | GET rejected/actor; нет нового approve, корректная причина/переход к задаче |
| A11 | 409 args/version/idempotency; 422 budget; cancelled/invalidated | Применяются реальные codes/details; нельзя исправить старый body и повторить его старым key; запрещённый инструмент не выполняется |
| A12 | resume с новым expires_at того же ID / другим ID / новым run deadline | Первый случай — mismatch без mutation; второй — новый экран/явное решение; третий — актуальный run context, без продления TTL |
| A13 | revoke/expired JWT во время POST; смена A→B; поздний ACK A | Coordinator закрывает доступ; old epoch не меняет B; нет mutation до auth+GET; operation A не отправляется B |
| A14 | SSE duplicate/out-of-order; старый GET pending приходит после terminal v2 | Dedupe/resync по DOC-036; terminal не возвращается к pending; контроллер перечитывает latest |
| A15 | UI light/dark, 200% font, TalkBack, длинные argv/path, portrait/landscape | Touch target ≥48dp, meaningful args/дедлайн и обе допустимые CTA достижимы, контраст/фокус/семантика/reduce motion по DOC-061 §8; golden будущего UI-12 в обеих темах |

Трассировка: FR-038–043/068; TC-011/021/022/032, общий мобильный путь TC-003. Полный TC-021 требует реального Runtime/dispatch и marker oracle; controller fixture не доказывает «инструмент исполнился ровно один раз». Формат/analyze/widget/golden/APK не запускались: Flutter-пакета нет.
