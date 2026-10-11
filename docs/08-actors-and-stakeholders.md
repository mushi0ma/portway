# Portway — [Акторы][g-actors] и заинтересованные стороны

| Поле | Значение |
|---|---|
| Артефакт | [DOC-008][id-doc-008] |
| Версия и дата | v0.5 · 10.10.2026 |
| Роль документа | Справочный материал; использовать по необходимости, без самостоятельной очереди работ; актуальный состав — [навигатор](README.md#existing) |
| Статус | Актуализирован проектный документ; реализация и испытания не подтверждены |
| Основания | [Навигатор](README.md), [backlog](01-product-backlog.md), [ТЗ](02-technical-specification.md), [SRS](03-software-requirements-specification.md), [SMART](04-smart-goals-and-metrics.md), [MoSCoW](05-moscow-prioritization.md) |
| Связанные артефакты | [Глоссарий](07-glossary.md#glossary), [Анализ аналогов](09-analogue-analysis.md#analogues), [Границы проекта и ограничения](10-project-scope-and-constraints.md#scope), [Первоначальный реестр рисков](11-initial-risk-register.md#risks), [Use Case](12-use-case-diagrams-and-specifications.md), [сценарии качества](13-quality-attribute-scenarios-and-utility-tree.md), [приёмка](14-demo-acceptance-criteria.md) |

<a id="documentation-plan"></a>
## Состав документации

Подготовлены все восемь сводных контрактов сокращённого плана; всего 22 проектных артефакта. [Состав и актуальный остаток](README.md#remaining), [сопоставление прежних DOC-ID](01-product-backlog.md#legacy-registry). Новых обязательных проектных документов в очереди нет. Следующая работа — реализация и проверки; [BL][g-work-ids]/[CW][g-work-ids] и все [TC][g-acceptance]/[AC][g-acceptance] остаются неподтверждёнными.

<a id="actors"></a>
## 1. Граница системы

Для действующей [Use Case-модели](12-use-case-diagrams-and-specifications.md) Portway включает [Android][g-flutter] [APK][g-apk], [Go][g-go] [control plane][g-control-plane], локальный [Runtime][g-runtime] и внутренний сервис [аудита][g-audit]/состояния. Модельный сервис, [MCP-сервис][g-mcp], [Supabase][g-supabase] и [FCM][g-push] находятся за этой границей, даже если часть сервисов размещена на том же ПК. [Coder][g-main-roles]/[reviewer][g-main-roles] — внутренние исполнители Portway; в диаграмме вариантов использования всей системы они не подменяют внешнего пользователя. Уточнение границы отдельной подсистемы допускается только с явным названием этой подсистемы.

| ID | [Актор][g-actors] | Цель / взаимодействие | Полномочия и граница доверия | Связи |
|---|---|---|---|---|
| <a id="act-01"></a>[ACT-01][id-act-01] | [Владелец][g-owner] через [APK][g-apk] | Регистрация/вход, конфигурация, [запуск][g-run]/[stop][g-cancelled], [апрув][g-approval], связи, [пакет][g-package] и результат | Единственный допущенный [owner][g-owner]; все операции проверяет сервер | [SYS-02][id-sys-02]/04/06–12/16/18/19 |
| <a id="act-02"></a>[ACT-02][id-act-02] | Оператор [установки][g-host] — тот же [владелец][g-owner] | Подготовить ПК/второй [хост][g-host], доступы, конфигурацию и разобрать сбой | Эксплуатационная [роль][g-role] того же человека, не второй аккаунт продукта; серверные [секреты][g-secret] не вводятся в экспортируемый [пакет][g-package] | [SYS-01][id-sys-01]/13/17; [NFR-012][id-nfr-012]/024 |
| <a id="act-03"></a>[ACT-03][id-act-03] | Совместимый модельный сервис | Получить запрос и вернуть [stream][g-model-protocol]/[tool calls][g-tool]/[usage][g-usage]/ошибку | Внешний ответ — данные; предложение [tool call][g-tool] не разрешает исполнение | [SYS-03][id-sys-03]; [FR-011][id-fr-011]–016 |
| <a id="act-04"></a>[ACT-04][id-act-04] | [MCP-сервис][g-mcp] документации | Ответить на разрешённый запрос [инструмента][g-tool] | Ограниченный адаптер и [политика][g-policy]; содержимое ответа не повышает права | [SYS-05][id-sys-05]; [BL-042][id-bl-042] |
| <a id="act-05"></a>[ACT-05][id-act-05] | [Supabase Auth][g-supabase] | Подтвердить учётную запись/сессию | Успешная [аутентификация][g-auth] ещё требует проверки [owner][g-owner]/device в [Go][g-go] | [SYS-12][id-sys-12]; [FR-001][id-fr-001]–003/081 |
| <a id="act-06"></a>[ACT-06][id-act-06] | [Supabase PostgreSQL][g-supabase] | Хранить состояния, версии, [события][g-event]/[outbox][g-outbox] и [аудит][g-audit] | Доступ серверный; бизнес-таблицы не открываются [APK][g-apk] через anonymous [Data API][g-supabase] | [SYS-01][id-sys-01]/13; [SRS][g-requirement-ids] § 5.1/5.4 |
| <a id="act-07"></a>[ACT-07][id-act-07] | [FCM][g-push] | Доставить уведомление установленному [APK][g-apk] | [Push][g-push] содержит сигнал открыть актуальное состояние, не разрешение [инструмента][g-tool] | [SYS-16][id-sys-16]; [FR-075][id-fr-075]/076/086 |
| <a id="act-08"></a>[ACT-08][id-act-08] | [GitHub][g-github] — условный [актор][g-actors] S | Принять разрешённую публикацию и вернуть [PR][g-pr]/ошибку | Только при допуске расширения; временный ограниченный [credential][g-secret] и [апрув][g-approval] | [BL-070][id-bl-070]/071; [FR-056][id-fr-056]–059 |

[Android][g-flutter]/[Google Play Services][g-flutter] и HTTPS-туннель — инфраструктурные зависимости маршрута, а не люди с продуктовыми полномочиями. Второй [хост][g-host] — целевое окружение переноса, не отдельный пользователь. [ИИ-помощник][g-development-agent], разрабатывающий Portway, относится к команде проекта; не требуется добавлять его как actor работающего [MVP][g-mvp].

## 2. Заинтересованные стороны и ответственность

| ID | Сторона | Интерес / критерий успеха | Участие в решениях |
|---|---|---|---|
| <a id="stk-01"></a>[STK-01][id-stk-01] | [Владелец][g-owner], автор курсовой и первый пользователь | Рабочий [APK][g-apk], управляемые [агенты][g-agent], перенос, защита и понимание собственной работы | Глобальный scope, срок, дополнительные расходы; доступы и физический телефон; 3–4 ч в день 8–11.10 |
| <a id="stk-02"></a>[STK-02][id-stk-02] | [ИИ-агент разработки][g-development-agent] | Согласованные документы, реализация/проверки и объяснимый результат | Обычные технические решения делегированы; не принимает за [владельца][g-owner] глобальное снижение обязательных требований |
| <a id="stk-03"></a>[STK-03][id-stk-03] | Преподаватель / сторона оценивания курса | Демонстрация критериев первого семестра, теория, источники и объяснение | Оценивает по приложенным критериям; доступ к production-аккаунту или административная [роль][g-role] не предполагаются |
| <a id="stk-04"></a>[STK-04][id-stk-04] | Поставщики инфраструктуры и [моделей][g-model] | Корректное использование предоставленных сервисов | Определяют доступность/лимиты своих сервисов; не управляют внутренним scope Portway |

[Владелец][g-owner] подтверждает итоговую учебную демонстрацию; [агент][g-development-agent] готовит технические [свидетельства][g-evidence]. Преподаватель не считается уже согласовавшим архитектуру. Потенциальные команды/платящие заказчики не вводятся как установленные [стейкхолдеры][g-actors]: многопользовательский продукт вне [MVP][g-mvp].


## Связи и сопровождение

[Карта документов и результаты сверки](README.md#audit); [термины](07-glossary.md#glossary); [актуальные критерии приёмки](14-demo-acceptance-criteria.md). Численные правила — [SRS § 4.2](03-software-requirements-specification.md#defaults), методики — [SMART](04-smart-goals-and-metrics.md#methods), состав — [MoSCoW](05-moscow-prioritization.md#scope). Изменение этого артефакта проверяется по связанным требованиям и зависимым документам.

## Источники и история

Прямые решения [владельца][g-owner] 07–10.10.2026; [бриф](</home/mushi/Downloads/portway-brief-v2-design-roadmap.md>); [критерии первого семестра](</home/mushi/Downloads/КМ07 Бағалау критерийлер курсовой і 26-27 ж..docx>), сопоставленные в [SRS](03-software-requirements-specification.md).

| Версия | Дата | Изменение |
|---|---|---|
| v0.1–0.2 | 08.10.2026 | Подготовлен как часть [DOC-007][id-doc-007]–011; содержание сохранено при разделении |
| v0.3 | 10.10.2026 | Выделен в самостоятельный канонический документ; добавлены прямые связи, терминология и актуальный статус |
| v0.4 | 10.10.2026 | По запросу [владельца][g-owner] очередь сокращена с 57 позиций до 8 сводных документов; определена роль существующего материала, ссылки на старые DOC-ID сохранены как сопоставление. Требования и приоритеты [BL][g-work-ids]/[CW][g-work-ids] не изменены |
| v0.5 | 10.10.2026 | Подготовлены все восемь сводных контрактов; обновлены навигация/статусы/ссылки и связь с [RTM][g-traceability]/[TC][g-acceptance]; требования, приоритеты и NOT_RUN сохранены |

<!-- glossary-and-id-links: определения ссылок для текста выше -->
[g-agent]: 07-glossary.md#term-agent
[g-apk]: 07-glossary.md#term-apk
[g-approval]: 07-glossary.md#term-approval
[g-audit]: 07-glossary.md#term-audit
[g-auth]: 07-glossary.md#term-auth
[g-cancelled]: 07-glossary.md#term-cancelled
[g-control-plane]: 07-glossary.md#term-control-plane
[g-development-agent]: 07-glossary.md#term-development-agent
[g-event]: 07-glossary.md#term-event
[g-evidence]: 07-glossary.md#term-evidence
[g-flutter]: 07-glossary.md#term-flutter
[g-github]: 07-glossary.md#term-github
[g-go]: 07-glossary.md#term-go
[g-host]: 07-glossary.md#term-host
[g-main-roles]: 07-glossary.md#term-main-roles
[g-model]: 07-glossary.md#term-model
[g-model-protocol]: 07-glossary.md#term-model-protocol
[g-mvp]: 07-glossary.md#term-mvp
[g-outbox]: 07-glossary.md#term-outbox
[g-owner]: 07-glossary.md#term-owner
[g-package]: 07-glossary.md#term-package
[g-policy]: 07-glossary.md#term-policy
[g-pr]: 07-glossary.md#term-pr
[g-push]: 07-glossary.md#term-push
[g-requirement-ids]: 07-glossary.md#term-requirement-ids
[g-role]: 07-glossary.md#term-role
[g-run]: 07-glossary.md#term-run
[g-runtime]: 07-glossary.md#term-runtime
[g-secret]: 07-glossary.md#term-secret
[g-supabase]: 07-glossary.md#term-supabase
[g-tool]: 07-glossary.md#term-tool
[g-usage]: 07-glossary.md#term-usage
[id-act-01]: 08-actors-and-stakeholders.md#act-01
[id-act-02]: 08-actors-and-stakeholders.md#act-02
[id-act-03]: 08-actors-and-stakeholders.md#act-03
[id-act-04]: 08-actors-and-stakeholders.md#act-04
[id-act-05]: 08-actors-and-stakeholders.md#act-05
[id-act-06]: 08-actors-and-stakeholders.md#act-06
[id-act-07]: 08-actors-and-stakeholders.md#act-07
[id-act-08]: 08-actors-and-stakeholders.md#act-08
[id-bl-042]: 01-product-backlog.md#bl-042
[id-bl-070]: 01-product-backlog.md#bl-070
[id-doc-007]: 07-glossary.md
[id-doc-008]: 08-actors-and-stakeholders.md
[id-fr-001]: 03-software-requirements-specification.md#fr-001
[id-fr-011]: 03-software-requirements-specification.md#fr-011
[id-fr-056]: 03-software-requirements-specification.md#fr-056
[id-fr-075]: 03-software-requirements-specification.md#fr-075
[id-nfr-012]: 03-software-requirements-specification.md#nfr-012
[id-stk-01]: 08-actors-and-stakeholders.md#stk-01
[id-stk-02]: 08-actors-and-stakeholders.md#stk-02
[id-stk-03]: 08-actors-and-stakeholders.md#stk-03
[id-stk-04]: 08-actors-and-stakeholders.md#stk-04
[id-sys-01]: 02-technical-specification.md#sys-01
[id-sys-02]: 02-technical-specification.md#sys-02
[id-sys-03]: 02-technical-specification.md#sys-03
[id-sys-05]: 02-technical-specification.md#sys-05
[id-sys-12]: 02-technical-specification.md#sys-12
[id-sys-16]: 02-technical-specification.md#sys-16
[g-actors]: 07-glossary.md#term-actors
[g-mcp]: 07-glossary.md#term-mcp

[g-work-ids]: 07-glossary.md#term-work-ids

[g-acceptance]: 07-glossary.md#term-acceptance

[g-traceability]: 07-glossary.md#term-traceability
