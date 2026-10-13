# Протокол SAF import / unresolved mapping и UI-12 с большим шрифтом

Статус: **два исполнимых ручных сценария подготовлены; устройство и APK NOT_RUN**. Дата: 13.10.2026. Flutter-клиент ещё не реализован, существует Figma-дизайн; живого Figma в испытании нет. Здесь нет утверждения о принятом UI или проведённом переносе.

Точное покрытие: SAF-части TC-015 / TC-032, missing-mapping подмножество TC-026, AC-015; UI-12 accessibility-подмножество TC-003 / TC-032, AC-003, NFR-019. UI-15/16 относятся к пакету, UI-12 — к решению апрува. Один ZIP round trip не закрывает TC-016/AC-016 и A/M1→A/M2→B/M2.

## Условия начала и текущие блокеры

Общее: физический Android16, модель/Android API/build и фактические настройки записаны; установлен закреплённый APK с SHA-256/version/signing identity; backend revision/dirty/config/image/DB/OpenAPI закреплены; owner confirmed/device active; HTTPS Go API доступен; executor и safe evidence storage определены. Сейчас телефон/APK отсутствуют, следовательно все device trials NOT_RUN. Live endpoints не проверялись.

**Для SAF:** должны существовать реализованные UI-15/16 и server package endpoints, staging/validation/mapping/activation, тестовая установка и разрешённые target refs. README/status/plan (IP-07 TODO), текущий OpenAPI и routes подтверждают, что packages/import ещё отсутствуют. Сценарий нельзя начать сейчас и нельзя заменить API-only прогоном. До начала подготовить F-PACKAGE fixtures ниже, exact paths/hash/expected config; Android должен открывать настоящий системный document picker.

**Для UI-12:** нужны реализованный экран, auth/session coordinator, свежий pending fixture с безопасными длинными args и подтверждённым deadline, доступ к latest GET/decision trace. Approval routes в текущем backend есть, UI/публичное run-creation пока не готовы. Тестовый backend обязан иметь документированный безопасный способ создать pending; произвольный SQL/новый выдуманный endpoint не предлагаются.

Роли: владелец управляет телефоном/решает тестовые действия; тестовый backend создаёт fixture и возвращает authority state; исполнитель записывает факт. Один человек может выполнять все роли. Не требуются новые подключения или автоматический ADB.

## A. SAF: fixtures и ожидаемые состояния

| Fixture | Подготовить до старта | Expected |
|---|---|---|
| F-VALID-U | Валидный ZIP DOC-038 v1: manifest.json, AGENTS.md, mcp.json, допустимые skills/resources, корректные hashes; хотя бы один required alias намеренно без target mapping | validation проходит, mapping_required / список unresolved; нет activation/run |
| F-VALID-R | Тот же portable config, но доступны проверенные target refs для всех required dependencies | После явного mapping server ready; затем явное «Создать агента» |
| F-BAD | Отдельный разрешённый тестовый ZIP с corrupted hash либо неподдержанной schema | Понятная ошибка; нет частичного агента или исполнения |

Alias labels и target refs определяет fixture, они не являются существующими настройками проекта. В F-VALID-U заранее записать ожидаемые unresolved alias/type/required и expected portable hash. Реальные secrets не вкладывать: только references/markers. Размеры валидного файла в пределах ≤10 MiB input / ≤25 MiB распакованного / ≤1000 файлов. Граничные 10/25MiB,1000±1 и hostile entries относятся к полному TC-026; этот ограниченный ручной сценарий не заявляет их проверенными.

### Шаги импорта

| № | Владелец / действие | Backend / expected | Evidence / oracle |
|---|---|---|---|
| A1 | Открыть UI-15 → «Импортировать ZIP», затем отменить **системный** picker | Нет upload/activation | Запись системного picker и возврата; отмена пользователя не server error; agents/versions count без изменений |
| A2 | Повторить, выбрать F-VALID-U через «Открыть документ» | Scoped URI → upload → staging validation | Видны имя/размер; не запрашивается широкое permission всего накопителя; безопасный trace факта URI access без личных путей |
| A3 | Дождаться preview | uploaded→validated→mapping_required, import_id/expires_at/dependencies | Preview показывает роль/права/portable config, конкретные unresolved; сохранены ZIP SHA-256 и server validation state |
| A4 | Оставить required alias без привязки; открыть UI-16 | Activate/start disabled | Видны alias, target/ref и статус/причина; объяснение текстом/иконкой; disabled не только цветом |
| A5 | Уйти назад/закрыть экран, вернуться к последнему import | Pending не превращается в activated | GET того же import, прежние unresolved; нет нового AgentVersion/run/скрытого activation |
| A6 | В контролируемом серверном подмножестве попытаться activate с unresolved через документированный test harness | По DOC-036 expected 422; транзакция не создала AgentVersion | Exact approved harness command/request без secrets, response/status, agent/version count/audit. Сейчас endpoint отсутствует; шаг NOT_RUN, UI-disabled сам по себе не доказывает server rejection |
| A7 | Явно сопоставить все required model/provider/tool/MCP/secret/agent/role dependencies с допустимыми target refs | expected_version checked, права/capabilities/allowlist проверены; unresolved=[] и ready | Список mappings и effective-policy diff; source IDs не принимаются как target ID автоматически; secret value не возвращается |
| A8 | Нажать «Создать агента» один раз | activate атомарна; один AgentVersion + audit | Показаны name/version/hash; agent/version count изменился один раз; import→activated |
| A9 | Повторить tap / восстановить потерянный activation response разрешённой fixture сети | GET import и тот же key возвращают ту же версию | Один logical activation; новый key молча не создаётся; клиент не утверждает успех до server evidence |
| A10 | Выбрать F-BAD на новом import | validation reject | Понятная причина, возможность выбрать другой ZIP; ни частичного агента, ни выполнения ресурсов |

A6 — дополнительное C/I-подмножество для server guard, A1–5/7–10 остаются обязательными device/UI действиями. Не подменять отсутствующий system picker прямой загрузкой через curl.

### Сбой сети, cancel и экспорт

1. На **новом** F-VALID-U import контролируемо оборвать сеть при upload. Записать момент и, если известен, key/import_id. UI показывает фактическое неизвестное/ошибку, не «агент создан». После восстановления использовать GET import, если ID получен, и исходный request key; не загружать слепо с новым key. Проверить отсутствие двух import/двух AgentVersion; если API не даёт достаточной диагностики, outcome INCONCLUSIVE с причиной.
2. Отменить незавершённый import предусмотренным действием (не путать с A1 picker cancel); backend удаляет только неактивный staging по контракту. Отмена/закрытие не активирует его. Export ZIP пользователя не удаляется.
3. Контролируемо потерять ответ activation только после всех разрешённых mappings; восстановить GET+same key, подтвердить один AgentVersion. Исходную неудачу сохранять.
4. Для экспортной части TC-015 выбрать immutable версию → «Экспортировать» → системный «Сохранить документ»: сначала cancel, затем успешное сохранение. Записать имя/размер/SHA-256/config hash, отсутствие secrets/local host paths. F-VALID fixture import выше не заменяет экспорт. Весь TC-015 не PASS без этой и иных обязательных ветвей.

Плановые API из **DOC-036**, сейчас отсутствующие в OpenAPI/routes:

```text
GET /v1/agent-versions/{id}/package
POST /v1/package-imports                  multipart ZIP + original key
GET /v1/package-imports/{id}
PUT /v1/package-imports/{id}/mappings     expected_version + target refs
POST /v1/package-imports/{id}/activate    expected_version + original key
DELETE /v1/package-imports/{id}           non-active staging
```

До выполнения сверить фактические schemas/статусы с candidate OpenAPI. Это контрактные пути, не обещание работоспособности текущего стенда.

## B. UI-12: 200% font, длинные args, действия

Fixtures: свежий pending для каждого trial; безопасная длинная argv, длинные task/tool names, рабочий каталог и существенный изменяемый объект; отдельные loading/network/expired/revoked случаи. Все значения тестовые и заранее известны; содержимое argv из реального чувствительного payload не записывается. Команда из wireframe DOC-061 сама по себе не является разрешением на shell.

Для доступности решений можно использовать **«Отклонить»**: это проверяет достижимость/семантику без исполнения tool. Если проверяется «Разрешить», действие допускается только для отдельно согласованной controlled fixture в тестовой sandbox с независимым marker; server ACK всё равно не означает выполненный tool. Этот сценарий не заявляет TC-011/AC-011 пройденным.

| Trial | Тема | Ориентация | Фактический font scale | Сейчас |
|---|---|---|---|---|
| U1 | Dark | portrait | обычный (записать) | NOT_RUN |
| U2 | Dark | landscape | обычный (записать) | NOT_RUN |
| U3 | Light | portrait | обычный (записать) | NOT_RUN |
| U4 | Light | landscape | обычный (записать) | NOT_RUN |
| U5 | Dark | portrait | 200% (подтвердить) | NOT_RUN |
| U6 | Dark | landscape | 200% (подтвердить) | NOT_RUN |
| U7 | Light | portrait | 200% (подтвердить) | NOT_RUN |
| U8 | Light | landscape | 200% (подтвердить) | NOT_RUN |

Перед U5–U8 владелец выставляет 200% в системных настройках и сохраняет evidence фактического значения/диагностики candidate. Если телефон не позволяет подтвердить ровно 200%, записать реальное значение и limitation; меньший размер не объявлять проектной проверкой 200%. Тему проверить по реально открытому экрану, не только выбранному переключателю.

Для каждой строки U1–U8:

1. Открыть свежий pending на UI-12 через список апрувов; убедиться, что после auth выполнен **GET latest** и отображаются текущие state/version/args/deadline. Часы и offset/погрешность записаны. Если pending успел истечь во время проверки, не считать его valid-state trial: сохранить expiry-результат и подготовить новый.
2. Сверить полные существенные argv/путь/объект с fixture до решения. Длинные детали раскрываются/копируются безопасно; ellipsis без доступа к полному значению не проходит. Горизонтальный scroll только внутри code/diff, не всей формы. Поля решения не превращают immutable args в редактируемую команду.
3. Прокрутить экран до каждой кнопки, повернуть и проверить layout. При узкой ширине/200% «Разрешить» и «Отклонить» вертикальны и достижимы, текст не обрезан/не перекрыт системными inset. Нет preselected approve, swipe-confirm или auto approve. Зафиксировать target ≥48 dp через доступную layout/Semantics диагностику; снимок экрана сам по себе этого не доказывает.
4. Включить TalkBack и пройти фокусом: заголовок/задача → состояние/срок → tool → полные существенные args → действие/объяснение. Прослушать доступные названия кнопок, состояние disabled/loading/expired и причину; прочитать полные args и выполнить «Отклонить» с TalkBack на отдельном pending. Сохранить screen recording с озвучкой и фактическую последовательность; screenshot не подтверждает focus order.
5. Проверить действия и чтение при возврате из background: таймер синхронизирован по серверу, до latest GET старое решение недоступно. Цвет статуса сопровождается текстом/иконкой, уменьшение анимации учитывается.
6. На отдельном pending повторно нажать решение / вызвать допустимую задержку сети. Пока POST идёт, **обе** кнопки disabled и «Отправляем решение»; при потере ответа GET latest+исходный key. Один server outcome, нет двойного approve. «Разрешено» только после server ACK и отдельно от «Инструмент выполнен».
7. На отдельном expired/revoked fixture прочитать причину и проверить отсутствие рабочего stale approve. Таймер истёк по server deadline, не только по локальному обратному отсчёту.

Keyboard edge TC-032: не добавлять в UI-12 выдуманное поле ввода. Открыть клавиатуру на существующей форме UI-04/UI-09 и проверить предусмотренный переход/возврат к апруву: keyboard/insets корректны, действия не перекрыты. Если UI-12 не допускает клавиатуру, это конкретное ограничение подслучая, не повод объявлять всю keyboard-ветвь TC-032 проверенной. Длинная ошибка также проверяется отдельной fixture, а не строкой из production log.

Контраст UI-12 в каждой теме: обычный текст ≥4.5:1, крупный/значимые границы ≥3:1. Нужны реальные resolved цвета и alpha composition; три расчёта из результата 02 не покрывают все пары этого экрана. Figma binding, actual rendered contrast и touch geometry здесь NOT_RUN.

## Evidence: общая форма и две детализации

По DOC-057 §7 / skill evidence template:

```text
protocol_id / TC+подмножество / AC / FR-NFR / BL / UI:
date-time UTC + Asia/Almaty / executor / owner confirmation (пока нет):
source+backend+Flutter revisions / dirty-safe fingerprint:
APK SHA-256+version+signing / images+DB+OpenAPI+safe config fingerprint:
phone model / Android+API / network / installation+device labels:
preconditions / fixture revision+hash / exact actual steps+roles:
expected / actual (до прогона «не выполнено»):
safe correlation IDs / timestamps+offset+error:
artifact relative paths+hashes / redactions:
outcome / defect-blocker / rerun link / owner decision:
```

SAF дополнительно: ZIP filename/bytes/files/SHA-256/portable hash; настоящее picker recording; scoped URI permission fact; import_id/state/version/expires_at; expected и actual unresolved/mappings; safe endpoint/status/key correlation; count/ID AgentVersion до/после; cancel/network/duplicate исходы. Не хранить реальные content URI личных документов и secret values.

UI-12 дополнительно: U1–U8, фактический theme/orientation/font scale; keyboard/TalkBack settings и запись чтения/фокуса; long-args fixture hash; screenshot/recording до решения; measured target dp/метод; state/version/deadline/latest GET; decision ACK и отдельный tool outcome; expected/actual каждого assertion. Секреты, пароль, JWT, device proof и чувствительные argv удаляются из evidence с сохранением корреляции.

Форма короткого итога:

| Scenario/trial | Expected assertion | Actual | Evidence path/hash | Outcome / defect / rerun |
|---|---|---|---|---|
| SAF A1–A10 / network / export | По шагам выше | Не выполнено | Не создано | NOT_RUN: нет APK/телефона/package API |
| UI-12 U1–U8 / states | По шагам выше | Не выполнено | Не создано | NOT_RUN: нет APK/телефона/UI |

## Правило завершения и фактически выполненное

Подготовка протокола не является PASS. PASS только для фактически выполненного подмножества с достаточным evidence; whole TC-003/015/026/032 и AC не повышать по одной ветви. Исходный FAIL сохранять, повтор ссылать на него. Significant revision/config требует нового актуального прогона. Владелец подтверждает принятие отдельной записью после демонстрации. Обновление DOC-057/BL/status/plan сейчас не выполнялось из-за read-only границ испытания.

Фактически выполнены чтение указанных sources, сверка planned package endpoints с текущими OpenAPI/routes, подготовка fixtures requirements, state oracles, ручных шагов и evidence forms. Никакой ZIP не импортировался, picker не открывался, server fixture/API/network action не запускались, font/TalkBack/геометрия/контраст UI-12 на устройстве не проверялись, перенос B не проводился.

Источники (корень `/home/mushi/Workspace/Portway`): `skills/portway-android-acceptance/SKILL.md`; `references/device-scenarios.md`, `references/evidence-template.md`; `README.md`; `docs/implementation-status.md`; `docs/implementation-plan.md`; `docs/52-build-deploy-and-operations.md` §§1/4/6; `docs/57-verification-and-traceability.md` TC-003/015/026/032 и §§2/7; `docs/14-demo-acceptance-criteria.md` AC-003/015/016; `docs/61-android-screens-flows-and-api.md` §§3–6/8; `docs/38-agent-package-format.md` §§2/6/7; `docs/36-api-events-and-messages.md` §§5/8; `api/openapi.json`; `internal/transport/server.go`. Современные Android SDK-подробности по интернету не проверялись; Android16 указан как проектный target из DOC-014/057.
