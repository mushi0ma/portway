# Карта документации Portway

Все пути — от корня рабочего репозитория. Это навигатор, а не копия требований или таблица текущей готовности.

| Что изменилось | Что прочитать и при необходимости обновить |
|---|---|
| Экран, переход, состояние, UX-текст | `docs/61-android-screens-flows-and-api.md`: нужные UI-ID, §§ 2–8; `docs/03-software-requirements-specification.md`: связанные FR/NFR |
| Токены, темы, компоненты, иконки | DOC-061: раздел дизайн-системы (добавить в этот документ, если отсутствует), соответствие Figma ↔ Flutter и решения по Comet; `docs/20-architecture-and-decisions.md`: значимые ADR |
| Состояние/навигация/сеть клиента | DOC-020 `#flutter-architecture`, DOC-061; ADR по принятому решению, не обязательная смена уже выбранной библиотеки |
| HTTP, DTO, ошибки, SSE | `docs/36-api-events-and-messages.md`, `api/openapi.json`, реализация и контрактные тесты; `docs/29-data-and-state-model.md`; для SSE — `docs/24-execution-sse-retention.md` |
| Апрув, безопасность, доступ | `docs/25-approval-workflow.md`, `docs/40-agent-execution-and-security.md`, DOC-029/036/061; соблюсти разделение approved/executed |
| ZIP/staging/mapping | `docs/38-agent-package-format.md`, DOC-036/061; параметры и пределы — SRS § 4.2 |
| Сборка, APK, установка, ручной прогон | `docs/52-build-deploy-and-operations.md`, `docs/57-verification-and-traceability.md`, `docs/14-demo-acceptance-criteria.md` |
| Любой законченный инженерный срез | `docs/implementation-status.md`, `docs/implementation-plan.md`, связанные BL в `docs/01-product-backlog.md`, связанные TC в DOC-057; корневой README — только затронутые сведения/навигация |
| Реальное изменение backend-контракта | Согласованно DOC-036 + OpenAPI + код/тесты; клиентская задача сама по себе не разрешает переписать контракт |

Если меняется требование или граница поставки, сначала проверь реестр D, SRS и `docs/05-moscow-prioritization.md`. Не превращай техническую синхронизацию в новое продуктовое решение.

При завершении клиентского среза проверь BL-008/009/053 и затронутые BL-110…125, TC-003/012/015/032. Это кандидаты на проверку связей, не приказ изменить все их статусы. Уточняй актуальные номера по документам.

Для glossary reference links используй относительный путь от конкретного документа. Для корневого README определение будет `docs/07-glossary.md#term-run`, для документа в docs — `07-glossary.md#term-run`. Проверяй якорь в целевом файле; пути в code fences не являются ссылками.
