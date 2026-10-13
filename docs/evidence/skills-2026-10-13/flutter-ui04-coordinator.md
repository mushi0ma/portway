# UI-04: потеря login response, переход A→B и старый push

Дата: 13.10.2026. Это алгоритм будущего SessionCoordinator и набор проверок; Flutter-клиент/APK ещё не реализованы. Дизайн Figma существует по уточнению владельца, но не исследовался в рамках задачи. Реальные API, email, FCM и внешние сервисы не вызывались.

## Контрактные основания

- DOC-036 §1/2: login/refresh — исключение из replay credential-ответов. Они не кешируют токены в request_keys. Потерянный ответ завершается свежей проверкой сессии либо новым входом. Logout идемпотентен по device session.
- Фактический login: JSON `{email, password, device_label?}`, публичный POST `/v1/auth/login`; результат `{session, device_proof, device_id, owner}`. Каждый успешный `Identity.Login` создаёт новый device proof и device row. Header `Idempotency-Key` не входит в протокол login и не гарантирует повтор прежней сессии.
- Refresh: POST `/v1/auth/refresh` с `{refresh_token}` и `X-Device-Session`; не требует действующего access JWT, но proof должен быть активен, токены — того же subject/auth session. Refresh возвращает Session, не новый device proof.
- Защищённые чтения: Bearer JWT **и** `X-Device-Session`. Один unexpired JWT не доказывает действующий доступ: серверный device guard обязателен.
- DOC-020 §8, DOC-061 §2/5/7: один coordinator; смена API URL очищает старый контекст, требует подтверждения перехода и нового входа; pending destination только `{installation_id, approval_id}`; auth→GET latest→экран. Push не выбирает API URL и не подтверждает действие.
- DOC-040 §8: offline logout немедленно убирает local access, remote revoke отдельно; tombstone допускает только ограниченный защищённый proof для отзыва и не даёт входа.

## Модель coordinator

`InstallationContext = {trustedBaseUrl, installationId, name, apiVersion, epoch}`. URL задаётся пользователем в UI-02, только HTTPS с обычной проверкой сертификата. installation ID берётся из публичного GET `/v1/client-config`, не из уведомления. Совпадение host string не заменяет ID; если по прежнему URL обнаружена другая installation, тоже нужен новый контекст/вход.

`SessionState = unconfigured | checkingInstallation | signedOut(reason) | signingIn | sessionUncertain | validatingSession | refreshing | authenticated | switchingInstallation | revocationPending`. `revocationPending` — отдельный статус старой установки, не разрешение локального входа; текущая B может при этом оставаться signedOut/authenticated. Состояние навигации отдельно: `idle | awaitingAuth(destination) | fetchingDestination | showingDestination | destinationRejected`.

Один coordinator владеет secure storage, обновлением токенов single-flight, auth gates, lifecycle и закрытием SSE. Feature controllers не refresh-ят токены независимо и не хранят глобальный изменяемый client с заголовками старой сессии. Repositories получают immutable connection/session snapshot. Navigation adapter получает только разрешённые destinations. Выбор конкретных Flutter-библиотек этой спецификацией не утверждается.

Каждый запрос/refresh/login/navigation effect захватывает `{epoch, trustedBaseUrl, installationId, ownerId?, deviceId?}`. Сохранение токенов, cache write и переход выполняются только если snapshot ещё совпадает с текущим. Switch/logout/revoke повышает epoch до возможности старых callbacks что-либо записать. Отмена HTTP не доказывает отмену серверного эффекта.

## Алгоритм для поставленного сценария

### 1. Потерян login response установки A

1. `signingIn(A, epochA)` блокирует повторные taps. Пароль живёт только во временном поле формы/параметре запроса, не попадает в restoration/logs/analytics; очистка при завершении попытки, уходе/dispose/смене установки.
2. Network timeout после отправки → `sessionUncertain(A)`, не authenticated. Нельзя угадать JWT/device proof, восстановить их из push либо повторить login как идемпотентную mutation с тем же/new key.
3. Если до попытки действительно существовала **полная** защищённая локальная session A, свежая проверка GET `/v1/me` с ней может подтвердить только эту существующую session. Истёкший access обрабатывает один coordinator refresh с соответствующим proof; при потере refresh response — снова проверка имеющейся session либо новый вход, без гарантии replay/rotation recovery. Неполная/непроверенная session не открывает protected navigation.
4. В типичном первом входе ответ потерян целиком: session/proof отсутствуют. Доступ не восстанавливается через GET me без них; пользователю нужен новый явный вход. Его успех создаёт новую device session, а не «тот же login».
5. Сервер мог создать неизвестный клиенту device A. Без полученного proof/device ID клиент не может подтвердить его отзыв. Это ограничение показывается честно; позднее устройство можно проверить/отозвать через другой авторизованный device A/профиль A либо доверенную операцию оператора. Ни автоматической очистки такого device, ни endpoint «получить токены по request key» в контракте нет.

### 2. Пользователь выбирает другую установку B

1. UI-02 проверяет candidate URL B публичным GET client-config **без** Bearer/proof A; показывает name/ID B. Ошибочный сертификат/HTTP/недоступность не обходятся. Кандидат пока не становится активным auth endpoint.
2. Требуемое DOC-061 подтверждение перехода — часть будущего UI. До него контекст A сохраняется; после подтверждения начинается serial `switchingInstallation`, закрывается общий action gate и повышается epoch. Login/refresh/GET/navigation callbacks A уже не могут принять сессию или открыть экран.
3. Закрыть A SSE/подписки/timers, отменить запросы по возможности, убрать A feature controllers/caches/cursors/forms/pending destination/back stack. Секретные формы очищаются; передача того же password в B не происходит автоматически.
4. При наличии **известного** proof A возможна ограниченная попытка POST logout через отдельный immutable A connection. Logout принимает proof и опциональный ещё действующий JWT. ACK `{device_revoked:true, upstream_outcome: revoked|unavailable|not_attempted}` подтверждает durable revoke Portway отдельно от Supabase signout. Он не подтверждает реализованный push unlink, которого в текущем срезе нет.
5. Отсутствующий ACK → «Отзыв на сервере не подтверждён». Local access A удаляется независимо от сети. Если отложенный revoke проектируется, хранится только защищённая tombstone `{A endpoint, installationId, ограниченный deviceProof, revoke status}`; она доступна исключительно пути logout/revoke A, не пути login/refresh B. После ACK удаляется. Если proof потерян вместе с login response, tombstone не создаётся из выдуманных данных.
6. Commit B: trusted URL + проверенный installation ID B; создать новые repositories/cache namespace; `signedOut(B)`. Ни токены/owner/proof A, ни старый refresh flight, ни pending destination A не переносятся. Даже одинаковый email/Supabase project не делает installations общим контекстом.
7. Новый явный login B, проверка результата/контекста и безопасное сохранение session+proof+device ID+owner как согласованного набора. До завершения записи/проверки нет authenticated route. Если storage не принял полный набор, локальный вход не считается завершённым. Опоздавший 200 login A уничтожается как устаревший результат, не сохраняется в B; его remote effect остаётся отдельно неподтверждённым.

### 3. После switch пользователь открывает старое уведомление A

1. Общий обработчик background tap и normal terminated launch валидирует разрешённый тип и UUID-пару. Payload — только сигнал; произвольные URL/route/args/решение/credentials не используются. В pending хранится только `{installation_id, approval_id}`, внутренний generation хранится coordinator отдельно.
2. Сопоставить installation с **активной, уже проверенной B** до auth/navigation/GET approval. Для пары A≠B → `destinationRejected`: «Уведомление относится к другой установке». Сохранить текущий экран/сессию B; не делать GET A approval через B, не отправлять B credentials A, не переключать URL автоматически. Отбросить чужой destination, не заменить им валидный pending B.
3. Возможный следующий пользовательский путь — вручную открыть UI-02, выбрать известный A endpoint, подтвердить переход и войти A. Сам payload не содержит trusted URL. Без этого старый push не открывает protected A route и не запускает никакой mutation.
4. Для совпадающей installation: сохранить/coalesce pending pair; восстановить/проверить auth текущего контекста; при необходимости UI-04; после успешного auth GET `/v1/approvals/{id}` с актуальными JWT+proof; только затем открыть UI-12 с **серверным** состоянием. Pending появляется с controls только после свежего GET; expired/rejected/cancelled — причины/контекст без старых кнопок; approved — отдельно tool outcome. 404/403 — безопасная ошибка; 401 — coordinator; offline — явная невозможность проверить, не cached actionable screen.
5. Перед navigation commit ещё раз проверить epoch/installation и актуальность pending pair. Если во время login/GET пользователь переключился с B на C, старый destination не открывается. Повторные taps одного pair объединяются в один auth/GET/navigation flight; уже открытый экран получает revalidation. Не вводить вечный dedupe по одному approval ID, который мешал бы видеть его новый outcome.
6. Logout/revoke очищает protected back stack, доступ и feature data. Для текущей установки допустим pending pair через требуемый reauth, но Back/state restoration не восстанавливают protected экран без auth+GET. Смена установки всегда очищает старый pending pair.

## Push и реализация: обязательная граница

DOC-036 описывает PUT/DELETE `/v1/devices/current/push-token`, registration/rotation/unlink; эти маршруты **отсутствуют** в текущем OpenAPI и `Server.Handler`. Фактические revoke/logout сейчас меняют device row и audit, push storage/FCM отсутствуют. Нельзя объявить unlink успешным по `device_revoked=true` или mock success.

Будущий PushRepository имеет заменяемые real/fake adapters; fixture-режим видим. Не реализованный endpoint не превращается в рабочую кнопку «уведомления подключены». По DOC-061 permission запрашивается после объяснения, отказ не блокирует login; token только с owner/device, rotation обновляется, payload не содержит args/secrets, прямого approve нет, FCM TTL ≤ остатка approval deadline. Foreground обновляет состояние без двойного уведомления. Реальная доставка требует 3 background + 3 normal terminated trials до TTL; force-stop исключён из гарантии до повторного открытия. Ничего из этого не проверено данным проектным результатом.

## План проверок

Будущие unit/fake-repository проверки выполняются с управляемыми completion order и epoch; будущие widget/golden/Android проверки пока NOT_RUN.

| ID | Сценарий | Обязательный oracle |
|---|---|---|
| S01 | Первый login A committed, ответ потерян целиком | sessionUncertain/signedOut; нет локальных tokens/proof; GET me без credentials не используется как восстановление; новый вход не называется replay |
| S02 | Повторный tap login во время in-flight; новый явный login после unknown | Один текущий flight; второй завершённый login может иметь новый device ID; Idempotency-Key не обещает прежние credentials |
| S03 | Имеется полная session A, ответ другой login/refresh потерян | Только свежий GET me/допустимый refresh подтверждает доступ; failure/offline не даёт authenticated route |
| S04 | Два экрана и tap push одновременно требуют refresh | Один refresh flight; нет конкурентной token rotation; revoked proof → очистка доступа/причина, не бесконечный refresh loop |
| S05 | Candidate B возвращает другой installation ID | До подтверждения не посылаются A auth headers B; после подтверждения epoch меняется и нужен новый login B |
| S06 | Старые A login 200 / refresh 200 / GET approval завершаются после switch | Ни storage/cache, ни routes B не меняются; пароль A не переотправляется B; A outcome не объявляется отозванным |
| S07 | A logout ACK; upstream unavailable | device_revoked отмечен отдельно от upstream; неизвестный FCM unlink не объявлен успешным |
| S08 | A logout timeout/offline / lost login без proof | Local A access/back stack очищены; remote revoke unconfirmed; tombstone только для известного proof, не даёт входа; отсутствие proof не скрывается |
| S09 | Active B, старый push `{A, approvalA}`; затем повтор tap | Ни GET approvalA на B, ни запрос с B proof к A, ни авто-switch; понятная другая установка; B session/pending сохраняются |
| S10 | B push при signedOut; auth success; GET latest expired/rejected/cancelled/approved | Порядок auth→GET→route; никакого approve из payload; UI показывает actual state/tool, не уведомление |
| S11 | Смена B→C во время auth/GET destination B | Старый pending очищен, async commit отвергнут по epoch; Back не открывает B |
| S12 | Payload с URL/credential/command/invalid UUID/неизвестным типом | Нет произвольного endpoint/route/action; сигнал отвергнут/лишние поля не используются; safe logs не содержат секреты |
| S13 | 401 revoked / чужой subject / session ID mismatch / 403 / 404 / offline | Серверный отказ не заменён локальным JWT check; безопасный auth/error route; нет protected controls |
| S14 | Confirmation callback `portway://auth/confirmed`, повтор/посторонний next | Callback не выдаёт session и не навигирует как authenticated; обычный login; токены не принимаются из deep link |
| S15 | Restart/state restoration/logout/dispose | Нет password/setup/token/proof в restoration/crash/analytics; secure session сверяется до protected route; Back после logout требует auth |
| S16 | Permission deny/разрешение/rotation; current API push routes отсутствуют | Login работает при deny; limitation видима; fake явно обозначен; нет выдуманного real register/unlink успеха |
| S17 | Background и normal terminated tap, foreground duplicate, force-stop | Общий coordinator и dedupe; 3+3 реальные FCM trials отдельно после реализации; force-stop не включён в delivery guarantee |
| S18 | Light/dark, keyboard, 200% font/TalkBack, длинная ошибка/URL | Login поля/ошибки/смена URL/причина отзыва достижимы; ≥48dp targets, порядок фокуса и контраст по DOC-061 §8 |

Трассировка: FR-001–003/065/076/081–086/090, TC-002/003/012/022/029/032. Fake coordinator integration не закрывает real FCM, реальные signup/email/Supabase session rotation или телефонные испытания. Политика password/rate limits должна быть проверена в реальном проекте до приёмки; здесь она не выводится из фикстуры backend.
