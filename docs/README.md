# Portway — навигатор проектной документации

> **D-22, 11.10:** [микросервисы и Kafka](21-microservices-kafka-amendment.md) приняты владельцем. Этот amendment имеет приоритет над прежней топологией; control API, execution и audit/relay уже разделены; остальная целевая топология ещё не завершена.

> Реализация начата 11.10.2026: [фактический статус Go-бэкенда](implementation-status.md), [запуск и Docker](../README.md), [OpenAPI реализованного среза](../api/openapi.json). Ниже сохранён статус проектного комплекта на 10.10; он не заменяет статус кода.

**Актуально на 13.10.2026, Asia/Almaty.** [Free cloud connections/private Gateway](28-free-model-connections.md) добавлены; env-ключ настроен, реальные ограниченные probes выполнены без объявления полного accepted model. Verified tokenizer/Runtime/admission ещё нужны. Основной вход для разработки — [план реализации](implementation-plan.md), для фактов — [implementation status](implementation-status.md), для обязательств — ТЗ/SRS/Backlog/MoSCoW. Backend имеет выполненные срезы и протоколы; Runtime, реальные модели, Android/FCM и перенос ещё не приняты. Ниже история подготовки 22 проектных артефактов отделена от текущей реализации. Дата защиты 12.10 остаётся исходным решением; кандидат/репетиция 11.10 не заявляются состоявшимися.

<a id="existing"></a>
## Что из существующего нужно поддерживать

| Материалы | Роль в работе |
|---|---|
| [Backlog][g-work-ids], [SRS][g-requirement-ids], [MoSCoW][g-priorities], глоссарий, [критерии приёмки][g-acceptance] | Пять основных точек входа: что делать, как работает система, что входит в [MVP][g-mvp], как называются понятия и как проверить готовность |
| [ТЗ][g-requirement-ids], [SMART][g-smart]-методики, [риски][g-risk], [Use Case][g-use-case] | Нужные рабочие основания: учебное задание/решения, точные измерения, блокеры и сценарии. Дополняются при изменении соответствующей области |
| [Vision][g-vision], [акторы][g-actors], аналоги, пояснение границ, [Utility Tree][g-quality-scenario] | Пять готовых справочных материалов. Используются в архитектуре и курсовой; отдельные новые версии ради формальности не нужны. Канонические требования/приоритеты находятся в [ТЗ][g-requirement-ids]/[SRS][g-requirement-ids]/[MoSCoW][g-priorities] |

Из первых 14 материалов — 9 рабочих и 5 справочных; вместе с восемью новыми контрактами — 17 рабочих и 5 справочных. Справочный статус не означает удаление полезных источников или утрату связей. [ТЗ][g-requirement-ids] сохраняется для курса, [SRS][g-requirement-ids] — для разработки; одинаковые правила связываются ссылкой, а не размножаются в новых документах. Навигатор и прежний указатель 07–11 не считаются отдельными проектными артефактами.

<a id="prepared"></a>
## Подготовленные артефакты и источники истины

Прямые решения [владельца][g-owner] имеют приоритет. Для изменений сначала сверяется решение и соответствующее требование, затем обновляются зависимые представления. При расхождении документ исправляется; копии численных значений не создают самостоятельных новых правил.

| Артефакт | Каноническая ответственность | Зависимые материалы |
|---|---|---|
| [DOC-001 Backlog](01-product-backlog.md) | [BL][g-work-ids]/[CW][g-work-ids]/[SP][g-work-ids], зависимости работ, план 8 сводных документов и сопоставление исходных DOC-ID | [ТЗ][g-requirement-ids], [SMART][g-smart], [MoSCoW][g-priorities], [RTM][g-traceability] и приёмка |
| [DOC-002 ТЗ](02-technical-specification.md) | Назначение, [SYS][g-requirement-ids]/[QLT][g-requirement-ids], решения [D-01][id-d-01]–22, вопросы Q | [SRS][g-requirement-ids], [Vision][g-vision], границы и архитектура |
| [DOC-003 SRS](03-software-requirements-specification.md) | [FR/NFR][g-requirement-ids], технические defaults § 4.2, интерфейсные ограничения, КМ07 | [UC][g-use-case], [QS][g-quality-scenario], [AC][g-acceptance], контракты [DOC-036][id-doc-036]/038 и тесты [DOC-057][id-doc-057] |
| [DOC-004 SMART](04-smart-goals-and-metrics.md) | SM-цели, MX-методики, контрольные сроки, отделение цели от факта | Приёмка, план реализации, протокол |
| [DOC-005 MoSCoW](05-moscow-prioritization.md) | 66 M / 8 S / 4 C / 1 W по 79 [BL][g-work-ids]; 4 [CW][g-work-ids] отдельно; допуск расширений | Границы, [backlog][g-work-ids], [UC][g-use-case]/[QS][g-quality-scenario]/[AC][g-acceptance] |
| [DOC-006 Vision](06-product-vision.md) | Пользователь, ценность и основной сценарий | Аналоги, границы, архитектурные драйверы |
| [DOC-007 Глоссарий](07-glossary.md) | Термины и устойчивые якоря; не источник новых функций/лимитов | Все документы |
| [DOC-008 Акторы и стейкхолдеры](08-actors-and-stakeholders.md) | [ACT][g-actors]/[STK][g-actors], участники, ответственность и граница системы | [UC][g-use-case], [BPMN][g-architecture-notations], [C4][g-architecture-notations], угрозы |
| [DOC-009 Анализ аналогов](09-analogue-analysis.md) | Обзор первичных источников от 08.10, выводы для проекта | [Vision][g-vision], теория [CW-002][id-cw-002]/003 |
| [DOC-010 Границы и ограничения](10-project-scope-and-constraints.md) | Подробные включения/исключения выбранного [MoSCoW][g-priorities] | [UC][g-use-case], архитектура, приёмка |
| [DOC-011 Первоначальные риски](11-initial-risk-register.md) | [RSK-01][id-rsk-01]–16, сигналы и реакции; все открыты | [QS][g-quality-scenario], план, угрозы [DOC-040][id-doc-040]; обновляется на месте |
| [DOC-012 Use Case](12-use-case-diagrams-and-specifications.md) | 20 обязательных [UC][g-use-case] и [UML-диаграммы][g-uml] | [QS][g-quality-scenario], [AC][g-acceptance], RTM/API/TC [DOC-036][id-doc-036]/057 |
| [DOC-013 Качество / Utility Tree](13-quality-attribute-scenarios-and-utility-tree.md) | 21 шестичастный [QS][g-quality-scenario] для 24 [NFR][g-requirement-ids], оценки I/D | [AC][g-acceptance], архитектурные драйверы, тесты |
| [DOC-014 Приёмка демо](14-demo-acceptance-criteria.md) | 20 [AC][g-acceptance], условия/[свидетельства][g-evidence]/отклонения; всё [NOT_RUN][g-acceptance] | [RTM][g-traceability], [TC][g-acceptance], [ПМИ][g-traceability], фактический протокол |
| [DOC-020 Архитектура и решения](20-architecture-and-decisions.md) | [C4][g-architecture-notations]/[UML][g-uml], модули, зависимости, 16 [ADR][g-traceability]; проектный baseline | Связанные контракты и [DOC-057][id-doc-057] |
| [DOC-029 Данные и состояния](29-data-and-state-model.md) | [ER][g-architecture-notations], таблицы/ограничения, [транзакции][g-atomicity], состояния, retention; проектный baseline | Связанные контракты и [DOC-057][id-doc-057] |
| [DOC-036 API, события и сообщения](36-api-events-and-messages.md) | HTTP DTO/ошибки, [auth][g-auth]/device, [SSE][g-sse], envelope и routing; проектный baseline | Связанные контракты и [DOC-057][id-doc-057] |
| [DOC-040 Исполнение агентов и безопасность](40-agent-execution-and-security.md) | [Harness][g-harness], [policy][g-policy]/[approval][g-approval]/budget, [Runtime][g-runtime], угрозы и [recovery][g-recovery]; проектный baseline | Связанные контракты и [DOC-057][id-doc-057] |
| [DOC-038 Формат переносимого пакета](38-agent-package-format.md) | [ZIP][g-package]/manifest/[MCP][g-mcp], [staging][g-staging]/[mapping][g-staging]/[activation][g-staging], перенос; проектный baseline | Связанные контракты и [DOC-057][id-doc-057] |
| [DOC-061 Android: экраны, переходы и API](61-android-screens-flows-and-api.md) | 18 экранов, user flows, состояния, [push][g-push]/[ZIP][g-package], доступность; проектный baseline | Связанные контракты и [DOC-057][id-doc-057] |
| [DOC-057 Проверки и трассировка](57-verification-and-traceability.md) | 32 [TC][g-acceptance], 114 [FR][g-requirement-ids]/[NFR][g-requirement-ids] [RTM][g-traceability], 70 обязательных [BL][g-work-ids]/[CW][g-work-ids] связей, [ПМИ][g-traceability]/протокол; проектный baseline | Связанные контракты и [DOC-057][id-doc-057] |
| [DOC-052 Сборка, развёртывание и эксплуатация](52-build-deploy-and-operations.md) | Config/[CI][g-ci-cd]/сборка, [deployment][g-ci-cd], инструкции, [runbook][g-runbook], сохранность; проектный baseline | Связанные контракты и [DOC-057][id-doc-057] |

[DOC-007][id-doc-007]–011 выделены в отдельные файлы для прямых ссылок и независимого обновления. [Прежний объединённый путь](07-11-glossary-actors-analogues-scope-risks.md) содержит только переходы с сохранёнными якорями; он не второй источник содержания. Этот README — навигатор. Инструкции из прежнего [DOC-071][id-doc-071] описаны в [DOC-052][id-doc-052], а ПМИ — в [DOC-057][id-doc-057]; отдельная книга не требуется. Команды и фактические протоколы добавляются при реализации.

<a id="reduction-2026-10-10"></a>
## Решение о сокращении комплекта

10.10 [владелец][g-owner] попросил оставить действительно важное. Принято: 8 сводных документов вместо продолжения всех 57 позиций. Старые DOC-ID сохранены как карта разделов/условных расширений, без самостоятельного обязательства. Требования к приложению, безопасность и реальные проверки сохранены. Это заменяет прежнюю очередь «следующий [DOC-015][id-doc-015], затем весь реестр»; ниже результат предшествующей сверки сохранён как запись выполненной проверки.

При сокращении проверены локальные ссылки и устойчивые якоря, совпадение очереди и ролей документов. Канонические требования, решения, 79 [BL][g-work-ids] и 4 [CW][g-work-ids] сохранены; приоритеты совпадают с [MoSCoW][g-priorities]. Содержание 20 [AC][g-acceptance] и их покрытие 77 обязательных [FR][g-requirement-ids], 24 [NFR][g-requirement-ids], 20 [UC][g-use-case] и 21 [QS][g-quality-scenario] не изменено. Исходники диаграмм также сохранены. Это проверка документации; результатов испытаний приложения по-прежнему нет.

## Сверка после подготовки восьми документов

Созданы [C4][g-architecture-notations] Context/Container/Component и отдельный [UML Component][g-uml], [UML][g-uml] [Deployment][g-ci-cd]/[Sequence][g-uml]/[State][g-uml], [ER][g-architecture-notations], физический словарь, HTTP/[SSE][g-sse]/envelope/ZIP-контракты, 18 Android-экранов, 16 [ADR][g-traceability], 32 [TC][g-acceptance] и [RTM][g-traceability] всех 114 [FR][g-requirement-ids]/[NFR][g-requirement-ids]. Обратная связь есть у всех 66 обязательных [BL][g-work-ids] и 4 [CW][g-work-ids]. Проверены 24 Markdown-файла и 119 словарных записей: локальные ссылки/якоря и реестры согласованы, шесть JSON-примеров синтаксически корректны, исходные нормативные требования и приоритеты сохранены. Для диаграмм проверены исходники и структура блоков; полное визуальное отображение ещё не проверено. Реализация и испытания остаются NOT_RUN.

<a id="audit"></a>
## Предшествующая сверка 10.10.2026 — до сводных контрактов

Сопоставлены [backlog][g-work-ids], [ТЗ][g-requirement-ids], [SRS][g-requirement-ids], [SMART][g-smart], [MoSCoW][g-priorities], [Vision][g-vision], [DOC-007][id-doc-007]–013 и новая приёмка. Проверены идентичность/приоритеты работ, состав требований, [владельцы][g-owner] решений, сроки, [роли][g-role], границы доверия, лимиты, реальные интеграции и перенос.

| Проверено | Принятое состояние / исправление |
|---|---|
| Состав продукта | Один [owner][g-owner], [Flutter][g-flutter] [Android][g-flutter] [APK][g-apk] → [HTTPS][g-https] → Go-монолит/локальный [Runtime][g-runtime]; внешние [Supabase Auth][g-supabase]/[PostgreSQL][g-supabase], [FCM][g-push], [модели][g-model] и [MCP][g-mcp]. Общая система совпадает во всех представлениях |
| [Базовая поставка][g-mvp] | [coder][g-main-roles]→отдельный [reviewer][g-main-roles], [политика][g-policy]/[апрув][g-approval] 180 с, [безопасная альтернатива][g-safe-alternative], [stop][g-cancelled]/бюджеты, [события][g-event], [аудит][g-audit]/состояние, мобильный [пакет][g-package] и [A/M1→A/M2→B/M2][g-portability] обязательны |
| Расширения | [GitHub App][g-github]/[PR][g-pr] и [Runner][g-runner] — S, расписания/[Prometheus/Grafana][g-observability-tools] — C; новая S/C-работа с 11.10 не начинается. Схема [SRS][g-requirement-ids] уточнена: положительный [review][g-review] сначала даёт результат, публикация условна |
| Чтение и архив | [BL-013][id-bl-013] уточнён как расширенный просмотр/архив S; обязательные список/чтение/выбор версии остаются в [BL-010][id-bl-010]/012/110 и [FR-006][id-fr-006]/008/009 |
| Доступ | Регистрация с setup и email confirmation/callback; [owner][g-owner]/device проверяются сервером. [Offline][g-self-hosted] logout не выдается за подтверждённый серверный отзыв |
| Сроки и ресурс | Защита 12.10, кандидат 11.10. Окна 08–09.10 прошли без подтверждений; добавлен текущий статус 10.10. 12–16 ч относятся ко всему 08–11.10, фактический остаток неизвестен; M молча не сокращены |
| Численные правила | [SRS][g-requirement-ids] § 4.2 — единый источник defaults; [SMART][g-smart] — методик. Сверены 2 с / 5 с / 10 мин, [PERF-01][g-smart], 180 с, [run][g-run]/[chain-бюджеты][g-chain], 8 [hops][g-hops]/2 [depth][g-depth], 7/30 суток, ZIP 10/25 [MiB][g-host-resources] и 1000 файлов |
| Реальные [свидетельства][g-evidence] | [CI][g-ci-cd]/[Fake][g-fixture] не подменяют [Android][g-flutter]/[FCM][g-push], [Runtime][g-runtime], локальную [модель][g-model], [MCP][g-mcp] и [хост][g-host] B. Удалена формулировка [BL-116][id-bl-116], выглядевшая как уже выполненный [Android-прогон][g-flutter] |
| Версии и связи | Обновлены версии/статусы реестра, устаревшие подписи ссылок [SRS][g-requirement-ids] и очередь после [DOC-014][id-doc-014]; добавлены устойчивые ID-якоря, прямые ссылки и единый глоссарий |

Проверка выполнена 10.10: 16 Markdown-файлов (14 канонических артефактов, указатель и этот навигатор), 110 словарных записей; ошибок локальных ссылок, якорей, ссылок на ID и расхождений BL-приоритетов не обнаружено. Критерии [AC][g-acceptance] охватывают все 77 обязательных [FR][g-requirement-ids] с учётом M-частей двух смешанных требований, 24 [NFR][g-requirement-ids], 20 [UC][g-use-case] и 21 [QS][g-quality-scenario]. Исходники трёх [UML][g-uml] Use Case-диаграмм и [Utility Tree][g-quality-scenario] сохранены; у каждого [QS][g-quality-scenario] остались все шесть полей. Это проверка документации, испытания приложения не проводились.

Структурная проверка проверяет локальные Markdown-ссылки/якоря, ссылки терминов/ID, уникальность реестров, совпадение BL-приоритетов и покрытие [UC][g-use-case]/[QS][g-quality-scenario]/[NFR][g-requirement-ids] обязательной приёмкой. Содержательная сверка относится к текущим проектным правилам; факт реализации/работоспособности ею не устанавливается. Аналоги сохраняют дату обзора 08.10; актуальность внешних возможностей повторно проверяется перед интеграцией/итоговой сдачей.

## Правила сопровождения

1. Изменение обязательства: решение [владельца][g-owner] → [ТЗ][g-requirement-ids]/[SRS][g-requirement-ids] → [backlog][g-work-ids]/[SMART][g-smart]/[MoSCoW][g-priorities]/границы → [UC][g-use-case]/[QS][g-quality-scenario]/[AC][g-acceptance] → [RTM][g-traceability] [DOC-057][id-doc-057]/контракты/тесты. Обычная техническая детализация выполняется [агентом][g-agent] в разрешённых границах.
2. Численные defaults редактируются в [SRS][g-requirement-ids], метод измерения — в [SMART][g-smart]; зависимые копии сверяются в той же правке. Цель и фактический результат хранятся отдельно.
3. У каждого проектного термина есть якорь в глоссарии. В тексте/таблицах/заголовках используются ссылки на него; код, машинные поля, исходники диаграмм, URL и названия внешних источников сохраняют точный синтаксис. Для них определения доступны через глоссарий и окружающий текст.
4. [BL][g-work-ids]/[CW][g-work-ids]/[FR/NFR][g-requirement-ids]/[SYS][g-requirement-ids]/[QLT][g-requirement-ids]/D/Q/[TBD][g-decision-ids]/[SM][g-smart]/[MX][g-smart]/[ACT][g-actors]/[STK][g-actors]/[RSK][g-risk]/[UC][g-use-case]/[QS][g-quality-scenario]/[AC][g-acceptance] ведутся в своих канонических документах. Ссылки на ещё не подготовленные [DOC][g-work-ids] ведут к строке реестра [backlog][g-work-ids] и не изображают существующий файл.
5. При новом артефакте обновляются реестр/навигатор, ссылки источников и обратные связи зависимых документов. История сохраняет состояние на свою дату; прежние числа в истории не являются текущим статусом.

<a id="remaining"></a>
## Сокращённый комплект подготовлен

Все восемь документов ниже созданы. Вместо исходных 57 следующих позиций используется этот комплект; [71 старый ID](01-product-backlog.md#legacy-registry) сохранён только как карта разделов и условных расширений. Контракты — проектные, результаты исполнения — NOT_RUN.

| [DOC][g-work-ids] | Подготовленный документ | Содержание | Статус |
|---|---|---|---|
| [DOC-020](20-architecture-and-decisions.md) | [Архитектура и решения](20-architecture-and-decisions.md) | [C4][g-architecture-notations]/[UML][g-uml], модули, зависимости, 16 [ADR][g-traceability] | v0.1; проектный baseline, NOT_RUN |
| [DOC-029](29-data-and-state-model.md) | [Данные и состояния](29-data-and-state-model.md) | [ER][g-architecture-notations], таблицы/ограничения, [транзакции][g-atomicity], состояния, retention | v0.1; проектный baseline, NOT_RUN |
| [DOC-036](36-api-events-and-messages.md) | [API, события и сообщения](36-api-events-and-messages.md) | HTTP DTO/ошибки, [auth][g-auth]/device, [SSE][g-sse], envelope и routing | v0.1; проектный baseline, NOT_RUN |
| [DOC-040](40-agent-execution-and-security.md) | [Исполнение агентов и безопасность](40-agent-execution-and-security.md) | [Harness][g-harness], [policy][g-policy]/[approval][g-approval]/budget, [Runtime][g-runtime], угрозы и [recovery][g-recovery] | v0.1; проектный baseline, NOT_RUN |
| [DOC-038](38-agent-package-format.md) | [Формат переносимого пакета](38-agent-package-format.md) | [ZIP][g-package]/manifest/[MCP][g-mcp], [staging][g-staging]/[mapping][g-staging]/[activation][g-staging], перенос | v0.1; проектный baseline, NOT_RUN |
| [DOC-061](61-android-screens-flows-and-api.md) | [Android: экраны, переходы и API](61-android-screens-flows-and-api.md) | 18 экранов, user flows, состояния, [push][g-push]/[ZIP][g-package], доступность | v0.1; проектный baseline, NOT_RUN |
| [DOC-057](57-verification-and-traceability.md) | [Проверки и трассировка](57-verification-and-traceability.md) | 32 [TC][g-acceptance], 114 [FR][g-requirement-ids]/[NFR][g-requirement-ids] [RTM][g-traceability], 70 обязательных [BL][g-work-ids]/[CW][g-work-ids] связей, [ПМИ][g-traceability]/протокол | v0.1; проектный baseline, NOT_RUN |
| [DOC-052](52-build-deploy-and-operations.md) | [Сборка, развёртывание и эксплуатация](52-build-deploy-and-operations.md) | Config/[CI][g-ci-cd]/сборка, [deployment][g-ci-cd], инструкции, [runbook][g-runbook], сохранность | v0.1; проектный baseline, NOT_RUN |

Дополнительные отдельные [RTM][g-traceability]/[ADR][g-traceability]/тест-планы/руководства не требуются: они входят в эти файлы. При реализации уточняются существующие контракты и появляются кодовые артефакты [OpenAPI][g-api]/JSON schemas/[миграций][g-environment]/[CI][g-ci-cd], а не новая формальная очередь проектных документов. [Runner][g-runner], расписания, [GitHub][g-github] PR и дополнительный мониторинг остаются условными по [MoSCoW][g-priorities].

Следующий обязательный этап: [очередь IP-01–08](implementation-plan.md) для 66 M-BL → реальные [Runtime][g-runtime]/две [модели][g-model]/[MCP][g-mcp]/[Android][g-flutter]/[FCM][g-push]/[хост][g-host] B → [TC][g-acceptance]/[AC][g-acceptance] и количественные измерения → candidate и [CW-001][id-cw-001]–004 с репетицией. Фактические результаты дополняют [DOC-057][id-doc-057]; введение, теория и источники курсовой ещё не написаны этими документами. Документация не закрывает работы и не доказывает достижимость оставшегося срока.

<!-- glossary-and-id-links: определения ссылок для текста выше -->
[g-acceptance]: 07-glossary.md#term-acceptance
[g-agent]: 07-glossary.md#term-agent
[g-api]: 07-glossary.md#term-api
[g-apk]: 07-glossary.md#term-apk
[g-approval]: 07-glossary.md#term-approval
[g-architecture-doc]: 07-glossary.md#term-architecture-doc
[g-architecture-notations]: 07-glossary.md#term-architecture-notations
[g-audit]: 07-glossary.md#term-audit
[g-cancelled]: 07-glossary.md#term-cancelled
[g-ci-cd]: 07-glossary.md#term-ci-cd
[g-control-plane]: 07-glossary.md#term-control-plane
[g-decision-ids]: 07-glossary.md#term-decision-ids
[g-depth]: 07-glossary.md#term-depth
[g-environment]: 07-glossary.md#term-environment
[g-event]: 07-glossary.md#term-event
[g-evidence]: 07-glossary.md#term-evidence
[g-fixture]: 07-glossary.md#term-fixture
[g-flutter]: 07-glossary.md#term-flutter
[g-gateway]: 07-glossary.md#term-gateway
[g-github]: 07-glossary.md#term-github
[g-harness]: 07-glossary.md#term-harness
[g-hops]: 07-glossary.md#term-hops
[g-host]: 07-glossary.md#term-host
[g-host-resources]: 07-glossary.md#term-host-resources
[g-https]: 07-glossary.md#term-https
[g-main-roles]: 07-glossary.md#term-main-roles
[g-mcp]: 07-glossary.md#term-mcp
[g-message]: 07-glossary.md#term-message
[g-model]: 07-glossary.md#term-model
[g-mvp]: 07-glossary.md#term-mvp
[g-observability-tools]: 07-glossary.md#term-observability-tools
[g-owner]: 07-glossary.md#term-owner
[g-package]: 07-glossary.md#term-package
[g-planning]: 07-glossary.md#term-planning
[g-policy]: 07-glossary.md#term-policy
[g-portability]: 07-glossary.md#term-portability
[g-pr]: 07-glossary.md#term-pr
[g-priorities]: 07-glossary.md#term-priorities
[g-push]: 07-glossary.md#term-push
[g-quality-scenario]: 07-glossary.md#term-quality-scenario
[g-requirement-ids]: 07-glossary.md#term-requirement-ids
[g-review]: 07-glossary.md#term-review
[g-role]: 07-glossary.md#term-role
[g-run]: 07-glossary.md#term-run
[g-runbook]: 07-glossary.md#term-runbook
[g-runner]: 07-glossary.md#term-runner
[g-runtime]: 07-glossary.md#term-runtime
[g-safe-alternative]: 07-glossary.md#term-safe-alternative
[g-sandbox]: 07-glossary.md#term-sandbox
[g-secret]: 07-glossary.md#term-secret
[g-self-hosted]: 07-glossary.md#term-self-hosted
[g-smart]: 07-glossary.md#term-smart
[g-stride]: 07-glossary.md#term-stride
[g-supabase]: 07-glossary.md#term-supabase
[g-task]: 07-glossary.md#term-task
[g-traceability]: 07-glossary.md#term-traceability
[g-ui]: 07-glossary.md#term-ui
[g-uml]: 07-glossary.md#term-uml
[g-use-case]: 07-glossary.md#term-use-case
[g-vision]: 07-glossary.md#term-vision
[g-work-ids]: 07-glossary.md#term-work-ids
[id-bl-010]: 01-product-backlog.md#bl-010
[id-bl-013]: 01-product-backlog.md#bl-013
[id-bl-116]: 01-product-backlog.md#bl-116
[id-cw-001]: 01-product-backlog.md#cw-001
[id-cw-002]: 01-product-backlog.md#cw-002
[id-d-01]: 02-technical-specification.md#d-01
[id-doc-001]: 01-product-backlog.md
[id-doc-007]: 07-glossary.md
[id-doc-014]: 14-demo-acceptance-criteria.md
[id-doc-015]: 01-product-backlog.md#doc-015
[id-doc-037]: 01-product-backlog.md#doc-037
[id-doc-067]: 01-product-backlog.md#doc-067
[id-doc-069]: 01-product-backlog.md#doc-069
[id-doc-071]: 01-product-backlog.md#doc-071
[id-fr-006]: 03-software-requirements-specification.md#fr-006
[id-rsk-01]: 11-initial-risk-register.md#rsk-01
[g-actors]: 07-glossary.md#term-actors
[g-risk]: 07-glossary.md#term-risk
[g-chain]: 07-glossary.md#term-chain
[g-sse]: 07-glossary.md#term-sse

[g-auth]: 07-glossary.md#term-auth

[g-budget]: 07-glossary.md#term-budget

[g-devcontainer]: 07-glossary.md#term-devcontainer

[g-export-import]: 07-glossary.md#term-export-import

[g-outbox]: 07-glossary.md#term-outbox

[g-recovery]: 07-glossary.md#term-recovery

[g-secret-reference]: 07-glossary.md#term-secret-reference

[g-skill]: 07-glossary.md#term-skill

[g-tool]: 07-glossary.md#term-tool

[id-doc-002]: 02-technical-specification.md

[id-doc-011]: 11-initial-risk-register.md

[id-doc-020]: 20-architecture-and-decisions.md

[id-doc-040]: 40-agent-execution-and-security.md

[id-doc-052]: 52-build-deploy-and-operations.md

[id-doc-057]: 57-verification-and-traceability.md

[g-go]: 07-glossary.md#term-go

[g-atomicity]: 07-glossary.md#term-atomicity

[g-staging]: 07-glossary.md#term-staging

[id-doc-036]: 36-api-events-and-messages.md

- [DOC-022: первый срез Kafka, доставка аудита и эксплуатационные ограничения](22-kafka-first-slice.md).

- [Execution foundation: фактическая граница и ограничения](23-execution-foundation.md).

- [Harness: модельные шаги, согласования и durable results](26-harness-turns-and-results.md).

- [Потоковый модельный адаптер — IP-01](27-model-gateway.md).
