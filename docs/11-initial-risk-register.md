# Portway — Первоначальный реестр [рисков][g-risk]

| Поле | Значение |
|---|---|
| Артефакт | [DOC-011][id-doc-011] |
| Версия и дата | v0.5 · 10.10.2026 |
| Роль документа | Рабочий документ; обновлять при изменении своей области; актуальный состав — [навигатор](README.md#existing) |
| Статус | Актуализирован проектный документ; реализация и испытания не подтверждены |
| Основания | [Навигатор](README.md), [backlog](01-product-backlog.md), [ТЗ](02-technical-specification.md), [SRS](03-software-requirements-specification.md), [SMART](04-smart-goals-and-metrics.md), [MoSCoW](05-moscow-prioritization.md) |
| Связанные артефакты | [Глоссарий](07-glossary.md#glossary), [Акторы и заинтересованные стороны](08-actors-and-stakeholders.md#actors), [Анализ аналогов](09-analogue-analysis.md#analogues), [Границы проекта и ограничения](10-project-scope-and-constraints.md#scope), [Use Case](12-use-case-diagrams-and-specifications.md), [сценарии качества](13-quality-attribute-scenarios-and-utility-tree.md), [приёмка](14-demo-acceptance-criteria.md) |

<a id="documentation-plan"></a>
## Состав документации

Подготовлены все восемь сводных контрактов сокращённого плана; всего 22 проектных артефакта. [Состав и актуальный остаток](README.md#remaining), [сопоставление прежних DOC-ID](01-product-backlog.md#legacy-registry). Новых обязательных проектных документов в очереди нет. Следующая работа — реализация и проверки; [BL][g-work-ids]/[CW][g-work-ids] и все [TC][g-acceptance]/[AC][g-acceptance] остаются неподтверждёнными.

<a id="risks"></a>
## 1. Метод

Оценки предварительные, экспертные, **не измеренные вероятности**: вероятность В = 1 низкая / 2 средняя / 3 высокая; влияние И = 1 локальное / 2 задержка среза / 3 срыв обязательного результата или безопасности. Балл В×И: 6–9 — высокий, 3–4 — средний, 1–2 — низкий. [Риск][g-risk] с И=3 требует внимания независимо от балла. Все записи сейчас **открыты**; факты реализации/испытаний не установлены.

[Агент][g-development-agent] ведёт техническую реакцию и [свидетельства][g-evidence]; «[владелец][g-owner]» в таблице означает человеческое участие/решение, а не исполнение всей инженерной работы вручную. План реакции не считается выполненной мерой. Остаточный [риск][g-risk] пересчитывается после проверки; сейчас он не оценён повторно.

Состояние сверено 10.10: все [риски][g-risk] открыты; прошлые даты в сигналах — исходные контрольные точки. Отсутствие [свидетельств][g-evidence] не доказывает, что внешнее подключение не существует, но не позволяет считать зависимость готовой. [RSK-01][id-rsk-01] имеет уже наблюдаемый сигнал отставания плана; технические причины остальных [рисков][g-risk] ещё не проверены.

## 2. [Риски][g-risk], сигналы и реакция

| ID | [Риск][g-risk] / В×И | Сигнал или срок проверки | Реакция и допустимый резервный путь | Ответственный / связи |
|---|---|---|---|---|
| <a id="rsk-01"></a>[RSK-01][id-rsk-01] | Обязательный объём не укладывается в четыре дня и 12–16 ч участия; 3×3=9 | На 10.10 подтверждены только документы; точки 08–09.10 прошли без [свидетельств][g-evidence] работающего среза; ресурс 12–16 ч относится ко всему 08–11.10 | Сначала M, заморозить [S/C][g-priorities]; учитывать реальные затраты. При неустранимом конфликте представить [владельцу][g-owner] варианты глобального изменения, не объявлять частичную работу полным [MVP][g-mvp] | [Агент][g-development-agent] ведёт план, [владелец][g-owner] решает scope; все [SM][g-smart] |
| <a id="rsk-02"></a>[RSK-02][id-rsk-02] | Доступы [Supabase][g-supabase]/[FCM][g-push]/[HTTPS][g-https]/[MCP][g-mcp] задерживают интеграцию; 3×3=9 | Ранний [smoke][g-fixture] 08.10 не проходит или нет [credential][g-secret] | Получить конкретный список недостающих доступов; подготовить [дублёры][g-fixture] для [CI][g-ci-cd] параллельно. [Дублёр][g-fixture] не закрывает реальную приёмку | [Агент][g-development-agent] + [владелец][g-owner] для аккаунтов; [SM-01][id-sm-01]/03/05 |
| <a id="rsk-03"></a>[RSK-03][id-rsk-03] | Локальная [модель][g-model] не даёт корректный [tool calling][g-model-protocol] или не помещается/слишком медленна; 3×3=9 | До 09.10 нет реального завершённого [run][g-run] на малом [fixture][g-fixture] | Проверить меньшую совместимую [модель][g-model]/параметры и формат ответа, записать [usage][g-usage]/errors. Удалённый [endpoint][g-model-endpoint] может дополнить, но не снимает обязательную локальную проверку | [Агент][g-development-agent]; [BL-021][id-bl-021]/023; [SM-03][id-sm-03] |
| <a id="rsk-04"></a>[RSK-04][id-rsk-04] | Не готова вторая [модель][g-model] или независимый [хост][g-host] B; 3×3=9 | Определить 08.10, доступность до 10.10; нет среды для матрицы | Подготовить отдельную явно обозначенную ВМ при достаточных ресурсах либо доступный второй [хост][g-host]. Не заменять B каталогом A; без целевого [run][g-run] перенос не подтверждён | [Агент][g-development-agent], [владелец][g-owner] для ресурсов; [SM-07][id-sm-07] |
| <a id="rsk-05"></a>[RSK-05][id-rsk-05] | [FCM][g-push]/[Android][g-flutter] permission/lifecycle не дают [push][g-push] в нужном состоянии; 2×3=6 | [Smoke][g-fixture] 08.10; шесть [background][g-app-lifecycle]/[terminated][g-app-lifecycle] прогонов по [MX-05][id-mx-05] | Проверить проект, токен, разрешение, delivery/error и актуальность [апрува][g-approval]; [SSE][g-sse] при открытом [APK][g-apk] не выдавать за замену [push][g-push] | [Агент][g-development-agent] + [владелец][g-owner] на телефоне; [SM-05][id-sm-05] |
| <a id="rsk-06"></a>[RSK-06][id-rsk-06] | [HTTPS][g-https]/[SSE][g-sse] нестабильны во внешней сети, ПК спит/недоступен; 2×3=6 | Нет [replay][g-sse]/реального маршрута 08–09.10; обрыв на репетиции | Проверить переподключение, питание и адрес; заменить технический [туннель][g-https] при необходимости. Подготовить проверенную сеть демонстрации; локальный localhost не закрывает мобильный сценарий | [Агент][g-development-agent] + [владелец][g-owner]; [SM-01][id-sm-01]/09 |
| <a id="rsk-07"></a>[RSK-07][id-rsk-07] | Escape/неограниченный shell/доступ к host или [секретам][g-secret]; 2×3=6 | До первого реального [инструмента][g-tool]; отрицательный Runtime-тест не проходит | Сначала [policy][g-policy]/[allowlist][g-policy]/[non-root][g-host-resources]/ресурсные и сетевые ограничения, затем [инструмент][g-tool]. При провале блокировать опасный путь; не переносить выполнение на незащищённый [хост][g-host] | [Агент][g-development-agent]; [SM-03][id-sm-03]/08 |
| <a id="rsk-08"></a>[RSK-08][id-rsk-08] | Утечка [секретов][g-secret] через [APK][g-apk], [лог][g-audit], [пакет][g-package] или ответ; 2×3=6 | Маркерный [secret][g-secret] scan и негативные [API][g-api]/экспорт-тесты перед релизом | Исключить секретные поля/маскировать, повторить тест всех поверхностей. При реальной утечке отозвать/сменить [credential][g-secret] через [владельца][g-owner], не включать значение в отчёт | [Агент][g-development-agent] + [владелец][g-owner] доступа; [BL-025][id-bl-025]/081/103 |
| <a id="rsk-09"></a>[RSK-09][id-rsk-09] | Повтор [сообщения][g-message]/гонка [апрува][g-approval] запускает действие дважды или после [stop][g-cancelled]; 2×3=6 | Повтор/конфликт [payload][g-payload]/late [approve][g-approval-decision] тесты 10–11.10 | Транзакционные проверки, [dedupe][g-idempotency], привязка аргументов, серверные counters; [unknown][g-unknown] внешний исход сверять, не повторять слепо | [Агент][g-development-agent]; [SM-04][id-sm-04]/05/08 |
| <a id="rsk-10"></a>[RSK-10][id-rsk-10] | Чужой [Supabase][g-supabase] user или отозванная сессия сохраняет доступ; 2×3=6 | Проверка [owner][g-owner]/device, logout/Back/[SSE][g-sse] 09–11.10 | Закрыть доступ по умолчанию; серверная проверка [owner][g-owner], отзыв device, отвязка [push][g-push] и повтор отрицательных тестов | [Агент][g-development-agent]; [SM-02][id-sm-02]/08 |
| <a id="rsk-11"></a>[RSK-11][id-rsk-11] | Недоверенный ZIP или непереносимые пути/[алиасы][g-secret-reference] ломают [импорт][g-export-import]; 2×3=6 | Отрицательные [пакеты][g-package] 10.10; matrix 11.10 | Проверять схему/пути/symlink/размеры, не исполнять код при [импорте][g-export-import]; явно переназначить секретные ссылки, сравнить конфигурацию | [Агент][g-development-agent]; [SM-06][id-sm-06]/07 |
| <a id="rsk-12"></a>[RSK-12][id-rsk-12] | Неизвестный [usage][g-usage] или общие резервы обходят бюджет; 2×3=6 | Конкурентный/граничный тест и неполный [usage][g-usage] 09–11.10 | Резервировать общий остаток, консервативно оценивать расход; при неограничимой оценке блокировать следующий платный вызов. Отдельно действуют время/[шаги][g-step] | [Агент][g-development-agent]; [BL-034][id-bl-034]/064; [NFR-010][id-nfr-010] |
| <a id="rsk-13"></a>[RSK-13][id-rsk-13] | Теория и подготовка [владельца][g-owner] вытеснены кодом; 2×3=6 | Нет черновика/источников 09–10.10 или репетиции 11.10 | Сохранить время [CW][g-work-ids] в ежедневном плане; [агент][g-development-agent] готовит материал, [владелец][g-owner] проверяет понимание и объясняет проект. Готовые [ТЗ][g-requirement-ids]/[SRS][g-requirement-ids] не заменяют теоретическую главу | [Владелец][g-owner] + [агент][g-development-agent]; [CW-001][id-cw-001]–004; [SM-11][id-sm-11] |
| <a id="rsk-14"></a>[RSK-14][id-rsk-14] | Сбой/[cleanup][g-cleanup] уничтожает материал защиты или оставляет старый [run][g-run]; 2×3=6 | [Recovery][g-recovery] и копирование [свидетельств][g-evidence] 11.10 | Проверить [interrupted][g-interrupted]/инвалидацию [апрувов][g-approval]; закрепить [APK][g-apk]/revision и скопировать материалы до срока очистки. Новый [run][g-run] вручную; сверить сохранённый результат | [Агент][g-development-agent]; [BL-036][id-bl-036]/044/119; [SM-08][id-sm-08]/10 |
| <a id="rsk-15"></a>[RSK-15][id-rsk-15] | [Метрики][g-audit] измерены некорректно либо производительность не достигает порогов; 2×2=4 | Нет сырых выборок/согласованных часов или провал [PERF-01][g-smart] 11.10 | Разделить [LLM][g-model]/[FCM][g-push]/[SSE][g-sse], учесть погрешность и потерянные [события][g-event]; исправить причину и повторить полный затронутый тест. Не подставлять цель в графу «факт» | [Агент][g-development-agent]; [SM-09][id-sm-09] |
| <a id="rsk-16"></a>[RSK-16][id-rsk-16] | Аналоги/документы устаревают, [C4][g-architecture-notations] путают с [UML][g-uml], внедряется неподтверждённая функция; 2×2=4 | Перед зависимым артефактом и итоговой сдачей | Проверить первичный источник и дату; вести один текущий раздел и стабильные ID; не утверждать отсутствие функции по молчанию страницы | [Агент][g-development-agent]; [DOC-009][id-doc-009]/024/027; [CW-003][id-cw-003] |

## 3. Порядок пересмотра

Пересмотр — на контрольных точках 08–11.10 и после нового факта. Для [RSK][g-risk] записываются дата, наблюдаемый сигнал, выполненная мера, [свидетельство][g-evidence], новая В/И и статус: открыт / реализовался как проблема / снижен / закрыт. Закрытие требует проверяемого основания; наличие инструкции по устранению не закрывает [риск][g-risk]. Новый внешний блокер фиксируется как проблема с исполнителем следующего действия, а не как «отменённая [задача][g-task]».

Этот реестр обновляется на месте по фактам реализации; прежний [DOC-067][id-doc-067] не требует отдельного файла. Угрозы/контроли прежних [DOC-048][id-doc-048]/049 описаны в сводном [DOC-040](40-agent-execution-and-security.md), проверки — в [DOC-057](57-verification-and-traceability.md).


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
[g-app-lifecycle]: 07-glossary.md#term-app-lifecycle
[g-approval]: 07-glossary.md#term-approval
[g-approval-decision]: 07-glossary.md#term-approval-decision
[g-architecture-notations]: 07-glossary.md#term-architecture-notations
[g-audit]: 07-glossary.md#term-audit
[g-cancelled]: 07-glossary.md#term-cancelled
[g-ci-cd]: 07-glossary.md#term-ci-cd
[g-cleanup]: 07-glossary.md#term-cleanup
[g-event]: 07-glossary.md#term-event
[g-evidence]: 07-glossary.md#term-evidence
[g-export-import]: 07-glossary.md#term-export-import
[g-fixture]: 07-glossary.md#term-fixture
[g-flutter]: 07-glossary.md#term-flutter
[g-host]: 07-glossary.md#term-host
[g-host-resources]: 07-glossary.md#term-host-resources
[g-https]: 07-glossary.md#term-https
[g-idempotency]: 07-glossary.md#term-idempotency
[g-interrupted]: 07-glossary.md#term-interrupted
[g-mcp]: 07-glossary.md#term-mcp
[g-message]: 07-glossary.md#term-message
[g-model]: 07-glossary.md#term-model
[g-model-endpoint]: 07-glossary.md#term-model-endpoint
[g-model-protocol]: 07-glossary.md#term-model-protocol
[g-mvp]: 07-glossary.md#term-mvp
[g-owner]: 07-glossary.md#term-owner
[g-package]: 07-glossary.md#term-package
[g-payload]: 07-glossary.md#term-payload
[g-policy]: 07-glossary.md#term-policy
[g-priorities]: 07-glossary.md#term-priorities
[g-push]: 07-glossary.md#term-push
[g-recovery]: 07-glossary.md#term-recovery
[g-requirement-ids]: 07-glossary.md#term-requirement-ids
[g-run]: 07-glossary.md#term-run
[g-secret]: 07-glossary.md#term-secret
[g-secret-reference]: 07-glossary.md#term-secret-reference
[g-smart]: 07-glossary.md#term-smart
[g-sse]: 07-glossary.md#term-sse
[g-step]: 07-glossary.md#term-step
[g-stride]: 07-glossary.md#term-stride
[g-supabase]: 07-glossary.md#term-supabase
[g-task]: 07-glossary.md#term-task
[g-tool]: 07-glossary.md#term-tool
[g-uml]: 07-glossary.md#term-uml
[g-unknown]: 07-glossary.md#term-unknown
[g-usage]: 07-glossary.md#term-usage
[id-bl-021]: 01-product-backlog.md#bl-021
[id-bl-025]: 01-product-backlog.md#bl-025
[id-bl-034]: 01-product-backlog.md#bl-034
[id-bl-036]: 01-product-backlog.md#bl-036
[id-cw-001]: 01-product-backlog.md#cw-001
[id-cw-003]: 01-product-backlog.md#cw-003
[id-doc-007]: 07-glossary.md
[id-doc-009]: 09-analogue-analysis.md
[id-doc-011]: 11-initial-risk-register.md
[id-doc-048]: 01-product-backlog.md#doc-048
[id-doc-059]: 01-product-backlog.md#doc-059
[id-doc-067]: 01-product-backlog.md#doc-067
[id-mx-05]: 04-smart-goals-and-metrics.md#mx-05
[id-nfr-010]: 03-software-requirements-specification.md#nfr-010
[id-rsk-01]: 11-initial-risk-register.md#rsk-01
[id-rsk-02]: 11-initial-risk-register.md#rsk-02
[id-rsk-03]: 11-initial-risk-register.md#rsk-03
[id-rsk-04]: 11-initial-risk-register.md#rsk-04
[id-rsk-05]: 11-initial-risk-register.md#rsk-05
[id-rsk-06]: 11-initial-risk-register.md#rsk-06
[id-rsk-07]: 11-initial-risk-register.md#rsk-07
[id-rsk-08]: 11-initial-risk-register.md#rsk-08
[id-rsk-09]: 11-initial-risk-register.md#rsk-09
[id-rsk-10]: 11-initial-risk-register.md#rsk-10
[id-rsk-11]: 11-initial-risk-register.md#rsk-11
[id-rsk-12]: 11-initial-risk-register.md#rsk-12
[id-rsk-13]: 11-initial-risk-register.md#rsk-13
[id-rsk-14]: 11-initial-risk-register.md#rsk-14
[id-rsk-15]: 11-initial-risk-register.md#rsk-15
[id-rsk-16]: 11-initial-risk-register.md#rsk-16
[id-sm-01]: 04-smart-goals-and-metrics.md#sm-01
[id-sm-02]: 04-smart-goals-and-metrics.md#sm-02
[id-sm-03]: 04-smart-goals-and-metrics.md#sm-03
[id-sm-04]: 04-smart-goals-and-metrics.md#sm-04
[id-sm-05]: 04-smart-goals-and-metrics.md#sm-05
[id-sm-06]: 04-smart-goals-and-metrics.md#sm-06
[id-sm-07]: 04-smart-goals-and-metrics.md#sm-07
[id-sm-08]: 04-smart-goals-and-metrics.md#sm-08
[id-sm-09]: 04-smart-goals-and-metrics.md#sm-09
[id-sm-11]: 04-smart-goals-and-metrics.md#sm-11
[g-development-agent]: 07-glossary.md#term-development-agent
[g-risk]: 07-glossary.md#term-risk
[g-work-ids]: 07-glossary.md#term-work-ids

[g-acceptance]: 07-glossary.md#term-acceptance

[g-traceability]: 07-glossary.md#term-traceability
