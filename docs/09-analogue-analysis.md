# Portway — Анализ аналогов

| Поле | Значение |
|---|---|
| Артефакт | [DOC-009][id-doc-009] |
| Версия и дата | v0.5 · 10.10.2026 |
| Роль документа | Справочный материал; использовать по необходимости, без самостоятельной очереди работ; актуальный состав — [навигатор](README.md#existing) |
| Статус | Актуализирован проектный документ; реализация и испытания не подтверждены |
| Основания | [Навигатор](README.md), [backlog](01-product-backlog.md), [ТЗ](02-technical-specification.md), [SRS](03-software-requirements-specification.md), [SMART](04-smart-goals-and-metrics.md), [MoSCoW](05-moscow-prioritization.md) |
| Связанные артефакты | [Глоссарий](07-glossary.md#glossary), [Акторы и заинтересованные стороны](08-actors-and-stakeholders.md#actors), [Границы проекта и ограничения](10-project-scope-and-constraints.md#scope), [Первоначальный реестр рисков](11-initial-risk-register.md#risks), [Use Case](12-use-case-diagrams-and-specifications.md), [сценарии качества](13-quality-attribute-scenarios-and-utility-tree.md), [приёмка](14-demo-acceptance-criteria.md) |

<a id="documentation-plan"></a>
## Состав документации

Подготовлены все восемь сводных контрактов сокращённого плана; всего 22 проектных артефакта. [Состав и актуальный остаток](README.md#remaining), [сопоставление прежних DOC-ID](01-product-backlog.md#legacy-registry). Новых обязательных проектных документов в очереди нет. Следующая работа — реализация и проверки; [BL][g-work-ids]/[CW][g-work-ids] и все [TC][g-acceptance]/[AC][g-acceptance] остаются неподтверждёнными.

<a id="analogues"></a>
## 1. Метод и ограничения

Кабинетный обзор официальных страниц, проверенных **08.10.2026**. Выбраны три близких по разным [задачам][g-task] продукта: Cline — [инструменты][g-tool] разработчика и контроль действий, OpenHands [Agent][g-agent] Canvas — управление [агентами][g-agent] и окружением, Dify Workflow Studio — построение и перенос процессов. Это целевая выборка, не полный обзор рынка. [Установки][g-host], сравнения скорости/качества [моделей][g-model] и стоимости здесь не проводились.

«Указано» означает заявление официального источника, а не наш результат испытания. «Не подтверждено в изученных страницах» не означает отсутствие функции. Возможности Portway ниже — требования к будущему [MVP][g-mvp], не доказанное превосходство над работающим продуктом.

## 2. Проверяемое сравнение

| Аналог | Что подтверждает официальный источник | Связь с [задачей][g-task] Portway / что проверить отдельно |
|---|---|---|
| Cline | Работа из IDE, терминала и desktop; изменения файлов, команды, [подтверждения действий][g-approval], выбор [моделей][g-model]/[совместимого endpoint][g-model-endpoint], [MCP][g-mcp], команды [агентов][g-agent] и расписания. [Официальный обзор](https://cline.bot/) | Близок к управляемому [агенту][g-agent] разработки. Нельзя описывать его как только автодополнение или только одного [агента][g-agent]. Наш [Android][g-flutter] [APK][g-apk] + [FCM][g-push] и [пакет][g-package] [A/M1→A/M2→B/M2][g-portability] этой страницей для Cline не подтверждены. |
| Cline: [задачи][g-task] | [Задача][g-task] содержит контекст; документация описывает продолжение сессий и Git-based checkpoints. [Tasks](https://docs.cline.bot/core-workflows/task-management) | Полезный ориентир для видимости изменений. Portway пока ограничивается новым ручным [run][g-run] после рестарта; перенос активного контекста не входит в [пакет][g-package]. |
| OpenHands [Agent][g-agent] Canvas | Визуальная работа с несколькими [агентами][g-agent] в отдельных worktree; подключение [harness][g-harness] через ACP, переключение local/VM/cloud, [MCP][g-mcp]/skills; Docker указан как необязательный. [Agent Canvas](https://www.openhands.dev/product/canvas) | Прямое пересечение с [оркестрацией][g-orchestration] и сменой [хоста][g-host]. Worktree не следует приравнивать к контейнерной изоляции. Специфический сценарий [Android][g-flutter]/[FCM][g-push]/ZIP Portway этой страницей не подтверждён. |
| Dify Workflow Studio | Визуальные процессы с [моделью][g-model], [инструментами][g-tool], ветвлениями и human [review][g-review]; [API][g-api]/web-публикация, история версий, [логи][g-audit] и перенос через DSL. [Workflow Studio](https://www.dify.ai/workflows) | Переносимое описание процесса уже существует у аналога; это не уникальное свойство Portway. Отдельно требуется проверить соответствие сценарию изолированной работы с [репозиторием][g-repository] и нашему мобильному [апруву][g-approval]. |

## 3. Вывод для проектирования

Следующие положения — **выводы [агента][g-development-agent] из сравнения**, а не заявления производителей:

1. [Оркестрация][g-orchestration], модельный выбор и перенос описания работы не дают оснований заявлять уникальность Portway. Учебная ценность — самостоятельная связная реализация выбранного сценария и проверяемые инженерные решения.
2. Фокус Portway: один [владелец][g-owner] управляет работой по коду из [Flutter][g-flutter] [APK][g-apk], получает FCM-запрос на подтверждение, видит отдельный [review][g-review] и переносит конфигурацию между проверенными [моделями][g-model]/[хостами][g-host]. Отличие пока формулируется как набор требований, который нужно подтвердить [SM-01][id-sm-01]–11.
3. Полезные ориентиры аналогов: показывать актуальный результат/изменения, отделять запрос действия от разрешения, явно отображать выбранное окружение, экспортировать конфигурацию с проверкой привязок. Это согласуется с уже выбранными [BL-032][id-bl-032]/052/082/084/118, не добавляет новый scope.
4. Не расширяем [MVP][g-mvp] до общего workflow-конструктора, ACP-хаба или автоматической команды [агентов][g-agent] только ради паритета функций. Добавление таких интеграций — отдельное будущее решение.
5. Для теоретической главы можно использовать эти источники и выводы с датой обращения. Перед итоговой сдачей проверить актуальность страниц; не заявлять отсутствие Android-поддержки у всех аналогов без отдельного исследования.


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
[g-api]: 07-glossary.md#term-api
[g-apk]: 07-glossary.md#term-apk
[g-approval]: 07-glossary.md#term-approval
[g-audit]: 07-glossary.md#term-audit
[g-flutter]: 07-glossary.md#term-flutter
[g-harness]: 07-glossary.md#term-harness
[g-host]: 07-glossary.md#term-host
[g-mcp]: 07-glossary.md#term-mcp
[g-model]: 07-glossary.md#term-model
[g-model-endpoint]: 07-glossary.md#term-model-endpoint
[g-mvp]: 07-glossary.md#term-mvp
[g-orchestration]: 07-glossary.md#term-orchestration
[g-owner]: 07-glossary.md#term-owner
[g-package]: 07-glossary.md#term-package
[g-portability]: 07-glossary.md#term-portability
[g-push]: 07-glossary.md#term-push
[g-repository]: 07-glossary.md#term-repository
[g-review]: 07-glossary.md#term-review
[g-run]: 07-glossary.md#term-run
[g-task]: 07-glossary.md#term-task
[g-tool]: 07-glossary.md#term-tool
[id-bl-032]: 01-product-backlog.md#bl-032
[id-doc-007]: 07-glossary.md
[id-doc-009]: 09-analogue-analysis.md
[id-sm-01]: 04-smart-goals-and-metrics.md#sm-01
[g-development-agent]: 07-glossary.md#term-development-agent

[g-work-ids]: 07-glossary.md#term-work-ids

[g-acceptance]: 07-glossary.md#term-acceptance

[g-traceability]: 07-glossary.md#term-traceability
