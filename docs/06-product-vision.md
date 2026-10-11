# Portway — [Vision][g-vision]

**[DOC-006][id-doc-006] · v0.4 · 10.10.2026.** Рабочее видение продукта на основе решений [владельца][g-owner]; реализация и результаты испытаний не подтверждены. Основания: [ТЗ](02-technical-specification.md), [SRS](03-software-requirements-specification.md), [SMART](04-smart-goals-and-metrics.md), [MoSCoW](05-moscow-prioritization.md).

**Роль:** справочный материал для архитектуры и курсовой; отдельной очереди работ не создаёт.

<a id="documentation-plan"></a>
## Состав документации

Подготовлены все восемь сводных контрактов сокращённого плана; всего 22 проектных артефакта. [Состав и актуальный остаток](README.md#remaining), [сопоставление прежних DOC-ID](01-product-backlog.md#legacy-registry). Новых обязательных проектных документов в очереди нет. Следующая работа — реализация и проверки; [BL][g-work-ids]/[CW][g-work-ids] и все [TC][g-acceptance]/[AC][g-acceptance] остаются неподтверждёнными.

## Пользователь и проблема

Первый пользователь Portway — разработчик, который хочет управлять своими [ИИ-агентами][g-agent] с телефона: ставить [задачи][g-task] по коду, видеть ход работы, разрешать опасные действия и получать результат, когда компьютера нет под рукой. Его потребность — сохранять контроль над исполнением и переносить настройки [агента][g-agent] при смене совместимой [модели][g-model] или [хоста][g-host]. [MVP][g-mvp] рассчитан на одного [владельца][g-owner] [установки][g-host]; разработчик проекта одновременно является его первым пользователем.

## Видение и ценность

**Portway — мобильный пульт для управляемой совместной работы [ИИ-агентов][g-agent] разработки с переносимой конфигурацией.** [Владелец][g-owner] задаёт [роли][g-role] и [связь агентов][g-link], наблюдает выполнение и принимает необходимые решения с [Android][g-flutter]. Основная ценность — воспроизводимая [оркестрация][g-orchestration] и подтверждённый перенос: инструкции, [навыки][g-skill] и настройки сохраняют смысл при смене окружения, а новые [секреты][g-secret] и привязки задаются отдельно. Совместимость каждой выбранной [модели][g-model] проверяется, универсальная поддержка любых [моделей][g-model] не обещается.

## Основной сценарий

[Владелец][g-owner] входит в [APK][g-apk], выбирает [агента][g-agent] и [репозиторий][g-repository], запускает [задачу][g-task]. [Coder][g-main-roles] работает в ограниченной среде; приложение показывает [события][g-event] и позволяет остановить работу. Действие, требующее разрешения, вызывает [push][g-push] даже при обычном закрытии приложения. Отказ не разрешает исходную команду; [агент][g-agent] может предложить [безопасную альтернативу][g-safe-alternative]. Завершённый результат проверяет отдельный [reviewer][g-main-roles]. [Владелец][g-owner] читает [вердикт][g-review] и при необходимости вручную запускает исправление. Затем экспортирует конфигурацию с телефона, импортирует её и повторяет контрольную [задачу][g-task] на второй [модели][g-model] и другом [хосте][g-host].

## Граница первой поставки

Обязательны [Flutter][g-flutter] [APK][g-apk], регистрация/вход/профиль/выход, версии и готовые [роли][g-role] [агентов][g-agent], ручная связь двух [агентов][g-agent], изолированные [инструменты][g-tool] разработки и один [MCP][g-mcp], [события][g-event], бюджеты, остановка, [апрувы][g-approval]/[FCM][g-push], мобильный [импорт][g-export-import]/[экспорт][g-export-import] и внутренний сервис [аудита][g-audit]/[логов][g-audit]/состояния. Сервер первоначально работает на ПК [владельца][g-owner]; [APK][g-apk] подключается по [HTTPS][g-https]. Данные и [авторизация][g-auth] используют [Supabase][g-supabase]; система зависит от внешних сервисов. Покупка собственного [домена][g-https] не обязательна.

[GitHub][g-github] [PR][g-pr], отдельный [Runner][g-runner], расписания и [Prometheus/Grafana][g-observability-tools] находятся вне [базовой поставки][g-mvp]. Web/iOS/Desktop, многопользовательские команды, автоматический planner, циклы исправлений и перенос памяти/[активного запуска][g-active-run] не входят в [MVP][g-mvp]. [Секреты][g-secret] не включаются в переносимый [пакет][g-package]. Точный объём и правила расширений закреплены в [MoSCoW][g-priorities].

## Успех и ограничения

К защите **12 октября 2026 года** нужен работающий учебный [MVP][g-mvp] первого семестра, теория и демонстрация [владельцем][g-owner]. Успех подтверждают реальный сценарий [coder][g-main-roles]→[reviewer][g-main-roles], [push][g-push]/[апрув][g-approval] и перенос [A/M1 → A/M2 → B/M2][g-portability] с телефона. Целевые показатели: минимум два [активных запуска][g-active-run], [p95][g-evidence] [событий][g-event] ≤2 с, остановка управляемой [цепочки][g-chain] ≤5 с, подготовленный старт ≤10 мин — в условиях [SMART][g-smart]/[SRS][g-requirement-ids]. Выполнение этих целей ещё не доказано.

Проект выполняют [владелец][g-owner] и [ИИ-агент][g-development-agent]; доступно 12–16 часов участия [владельца][g-owner] 8–11 октября. Стоимость стремимся минимизировать. [Модели][g-model], доступы и второй [хост][g-host] ещё требуют проверки; срок напряжённый. Глобальные решения остаются за [владельцем][g-owner], обычные технические — за [агентом][g-agent].


## Связи и сопровождение

[Карта документов и результаты сверки](README.md#audit); [термины](07-glossary.md#glossary); [актуальные критерии приёмки](14-demo-acceptance-criteria.md). Численные правила — [SRS § 4.2](03-software-requirements-specification.md#defaults), методики — [SMART](04-smart-goals-and-metrics.md#methods), состав — [MoSCoW](05-moscow-prioritization.md#scope). Изменение этого артефакта проверяется по связанным требованиям и зависимым документам.

## История изменений

| Версия | Дата | Изменение |
|---|---|---|
| v0.1 | 08.10.2026 | Подготовлено видение продукта |
| v0.2 | 10.10.2026 | Подготовлен [DOC-014][id-doc-014]; выполнена сверка документов, актуализированы календарный статус, ссылки и отдельные [DOC-007][id-doc-007]–011. Термины связаны с каноническим глоссарием; приоритеты и готовность реализации не повышены |
| v0.3 | 10.10.2026 | По запросу [владельца][g-owner] очередь сокращена с 57 позиций до 8 сводных документов; определена роль существующего материала, ссылки на старые DOC-ID сохранены как сопоставление. Требования и приоритеты [BL][g-work-ids]/[CW][g-work-ids] не изменены |
| v0.4 | 10.10.2026 | Подготовлены все восемь сводных контрактов; обновлены навигация/статусы/ссылки и связь с [RTM][g-traceability]/[TC][g-acceptance]; требования, приоритеты и NOT_RUN сохранены |

<!-- glossary-and-id-links: определения ссылок для текста выше -->
[g-active-run]: 07-glossary.md#term-active-run
[g-agent]: 07-glossary.md#term-agent
[g-apk]: 07-glossary.md#term-apk
[g-approval]: 07-glossary.md#term-approval
[g-audit]: 07-glossary.md#term-audit
[g-auth]: 07-glossary.md#term-auth
[g-chain]: 07-glossary.md#term-chain
[g-event]: 07-glossary.md#term-event
[g-evidence]: 07-glossary.md#term-evidence
[g-export-import]: 07-glossary.md#term-export-import
[g-flutter]: 07-glossary.md#term-flutter
[g-github]: 07-glossary.md#term-github
[g-host]: 07-glossary.md#term-host
[g-https]: 07-glossary.md#term-https
[g-link]: 07-glossary.md#term-link
[g-main-roles]: 07-glossary.md#term-main-roles
[g-mcp]: 07-glossary.md#term-mcp
[g-model]: 07-glossary.md#term-model
[g-mvp]: 07-glossary.md#term-mvp
[g-observability-tools]: 07-glossary.md#term-observability-tools
[g-orchestration]: 07-glossary.md#term-orchestration
[g-owner]: 07-glossary.md#term-owner
[g-package]: 07-glossary.md#term-package
[g-portability]: 07-glossary.md#term-portability
[g-pr]: 07-glossary.md#term-pr
[g-priorities]: 07-glossary.md#term-priorities
[g-push]: 07-glossary.md#term-push
[g-repository]: 07-glossary.md#term-repository
[g-requirement-ids]: 07-glossary.md#term-requirement-ids
[g-review]: 07-glossary.md#term-review
[g-role]: 07-glossary.md#term-role
[g-runner]: 07-glossary.md#term-runner
[g-safe-alternative]: 07-glossary.md#term-safe-alternative
[g-secret]: 07-glossary.md#term-secret
[g-skill]: 07-glossary.md#term-skill
[g-smart]: 07-glossary.md#term-smart
[g-supabase]: 07-glossary.md#term-supabase
[g-task]: 07-glossary.md#term-task
[g-tool]: 07-glossary.md#term-tool
[g-vision]: 07-glossary.md#term-vision
[id-doc-006]: 06-product-vision.md
[id-doc-007]: 07-glossary.md
[id-doc-014]: 14-demo-acceptance-criteria.md
[g-development-agent]: 07-glossary.md#term-development-agent

[g-work-ids]: 07-glossary.md#term-work-ids

[g-acceptance]: 07-glossary.md#term-acceptance

[g-traceability]: 07-glossary.md#term-traceability
