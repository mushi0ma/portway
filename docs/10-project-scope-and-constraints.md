# Portway — Границы проекта и ограничения

| Поле | Значение |
|---|---|
| Артефакт | [DOC-010][id-doc-010] |
| Версия и дата | v0.5 · 10.10.2026 |
| Роль документа | Справочный материал; использовать по необходимости, без самостоятельной очереди работ; актуальный состав — [навигатор](README.md#existing) |
| Статус | Актуализирован проектный документ; реализация и испытания не подтверждены |
| Основания | [Навигатор](README.md), [backlog](01-product-backlog.md), [ТЗ](02-technical-specification.md), [SRS](03-software-requirements-specification.md), [SMART](04-smart-goals-and-metrics.md), [MoSCoW](05-moscow-prioritization.md) |
| Связанные артефакты | [Глоссарий](07-glossary.md#glossary), [Акторы и заинтересованные стороны](08-actors-and-stakeholders.md#actors), [Анализ аналогов](09-analogue-analysis.md#analogues), [Первоначальный реестр рисков](11-initial-risk-register.md#risks), [Use Case](12-use-case-diagrams-and-specifications.md), [сценарии качества](13-quality-attribute-scenarios-and-utility-tree.md), [приёмка](14-demo-acceptance-criteria.md) |

<a id="documentation-plan"></a>
## Состав документации

Подготовлены все восемь сводных контрактов сокращённого плана; всего 22 проектных артефакта. [Состав и актуальный остаток](README.md#remaining), [сопоставление прежних DOC-ID](01-product-backlog.md#legacy-registry). Новых обязательных проектных документов в очереди нет. Следующая работа — реализация и проверки; [BL][g-work-ids]/[CW][g-work-ids] и все [TC][g-acceptance]/[AC][g-acceptance] остаются неподтверждёнными.

<a id="scope"></a>
## 1. Внутри и вне [базовой поставки][g-mvp]

Канонический состав — [MoSCoW](05-moscow-prioritization.md): **66 M / 8 S / 4 C / 1 W** по [BL][g-work-ids] и 4 обязательные [CW][g-work-ids]. Этот раздел раскрывает границы и не меняет приоритеты. Срок — защита 12.10.2026, кандидат/репетиция 11.10; достижимость всего объёма пока не доказана.

| Область | Внутри базового [MVP][g-mvp] | Вне базового [MVP][g-mvp] / ограничение | Связи |
|---|---|---|---|
| Пользователь и [UI][g-ui] | Один [владелец][g-owner]; [Flutter][g-flutter] [APK][g-apk] [Android 16][g-flutter]; Splash, регистрация/вход, главная, навигация, профиль, logout | Web/iOS/Desktop, команды, приглашения, публикация в магазине | [SYS-11][id-sys-11]/12/18/19; [SM-02][id-sm-02] |
| [Агент][g-agent] и [модель][g-model] | Неизменяемые версии, готовые 5 основных и 3 субагентских [пресета][g-role], один совместимый адаптер, две проверенные [модели][g-model], одна локальная | Обещание поддержки любой [модели][g-model], отдельный cloud-адаптер, редактор произвольных [ролей][g-role] | [SYS-02][id-sys-02]/03; [SM-03][id-sm-03]/07 |
| Исполнение | [Go][g-go] [control plane][g-control-plane] и контейнерный [Runtime][g-runtime] на [хосте][g-host]; файлы/shell/tests/git, ограниченный [devcontainer][g-devcontainer] и один [MCP][g-mcp] документации | Привилегированное выполнение, произвольные host mounts, host socket; отдельный [Runner][g-runner] — S | [SYS-05][id-sys-05]; [SM-03][id-sm-03]/08 |
| [Оркестрация][g-orchestration] | Ручная связь [coder][g-main-roles]→[reviewer][g-main-roles], отдельный [review][g-review] фиксированного результата, общий бюджет, [dedupe][g-idempotency] и [stop][g-cancelled] | Автопланировщик, автоматическое создание команды и циклы исправлений | [SYS-07][id-sys-07]/08; [SM-04][id-sm-04] |
| Подтверждения | Серверная [policy][g-policy], [TTL][g-timeout] 180 с, [FCM][g-push] на обычном закрытом [APK][g-apk], актуальный [апрув][g-approval], [безопасная альтернатива][g-safe-alternative] после отказа | [Push][g-push] как разрешение действия; гарантированная доставка при системном [force-stop][g-app-lifecycle]; обход отказа | [SYS-06][id-sys-06]/16; [SM-05][id-sm-05] |
| Перенос | ZIP конфигурации без [секретов][g-secret], [Android][g-flutter] [export][g-export-import]/[import][g-export-import] и матрица [A/M1→A/M2→B/M2][g-portability]; отдельная ВМ допустима явно | Полная память/активный [run][g-run], перенос [credentials][g-secret] внутри ZIP; другой каталог A не считается B | [SYS-10][id-sys-10]; [SM-06][id-sm-06]/07 |
| Данные и наблюдаемость | [Supabase PostgreSQL][g-supabase]/[Auth][g-auth], сохранённые [события][g-event], [аудит][g-audit]/[логи][g-audit]/состояние через защищённый [API][g-api] | Полностью offline-система, отдельный web-продукт мониторинга; [Prometheus/Grafana][g-observability-tools] — C | [SYS-01][id-sys-01]/13; [SM-08][id-sm-08] |
| Результат и [GitHub][g-github] | Локальный [репозиторий][g-repository], результат, [ревизия][g-repository] и [вердикт][g-review] на телефоне | [GitHub App][g-github]/[PR][g-pr] и private GitHub-доступ — S, вне базового демо | [SYS-09][id-sys-09]; [BL-072][id-bl-072]/118; [SM-12][id-sm-12] вне базы |
| Восстановление | [Interrupted][g-interrupted] после рестарта, старый [апрув][g-approval] недействителен, новый [запуск][g-run] вручную, неизвестный внешний исход виден | Автоматическое продолжение старого [Harness][g-harness] или слепой повтор внешнего действия | [SYS-17][id-sys-17]; [NFR-024][id-nfr-024] |
| Приёмка и курс | [CI][g-ci-cd]/[дублёры][g-fixture] плюс реальные проверки, [APK][g-apk]/инструкция, теория/источники и репетиция первого семестра | Автоматическое добавление критериев второго семестра; утверждение полного соответствия ГОСТ | [BL-119][id-bl-119]; [CW-001][id-cw-001]–004; [SM-10][id-sm-10]/11 |

Расписания [BL-090][id-bl-090]/091/115 — C; расширенные история, архив и [аудит][g-audit] — S. Они не блокируют минимальное чтение результата/[событий][g-event] и базовую диагностику. Правило допуска расширений и запрет начинать новые [S/C][g-priorities] с 11.10 до защиты — [MoSCoW][g-priorities] § 6.

## 2. Условия и открытые технические проверки

| Условие | Зафиксировано | Что ещё проверить / кто ведёт |
|---|---|---|
| Ресурс | [Владелец][g-owner] + [агент][g-development-agent]; 12–16 ч участия [владельца][g-owner] 8–11.10 | [Агент][g-development-agent] учитывает фактическое время; [владелец][g-owner] предоставляет доступы, телефон и проводит защиту |
| [Хост][g-host] A | CachyOS, i5-13420H, RAM 16 GB, свободно 275 GB | [Runtime][g-runtime]/изоляция/лимиты и пригодная локальная [модель][g-model]; GPU не заявлен |
| [Хост][g-host] B / [модель][g-model] M2 | Обязательны для доказательства переноса; ВМ допустима | [Агент][g-development-agent] выбирает/готовит доступное окружение; доступы/расходы при необходимости решает [владелец][g-owner] |
| Телефон | [Android 16][g-flutter] и [Google Play Services][g-flutter] подтверждены | Реальные permission, [push][g-push], deep link, повторное подключение и [установка][g-host] [APK][g-apk] |
| Внешние подключения | [Supabase][g-supabase] и [FCM][g-push] обязательны; HTTPS-туннель и [MCP-кандидат][g-mcp] выбраны | Аккаунты/[credentials][g-secret]/маршрут ещё не проверены; собственный [домен][g-https] покупать не требуется |
| Бюджет | Минимизировать стоимость, по возможности бесплатно | Численный денежный лимит не задан; платные подключения/покупки не предполагаются согласованными |
| Качество | ≥2 активных [run][g-run], [p95][g-evidence] [событий][g-event] ≤2 с, [stop][g-cancelled] ≤5 с, подготовленный старт ≤10 мин | Реальные измерения по [SMART][g-smart], включая запись погрешности и неуспешных прогонов |

Контрольная [задача][g-task] и её ожидаемый результат фиксируются до приёмки [моделей][g-model]/переноса. Малый [fixture][g-fixture] сокращает объём демо, но не заменяет настоящие инструментальные вызовы, изоляцию или два разных [model][g-model] ID. Анализ аналогов не является основанием менять выбранный стек без технической причины.

<a id="status-2026-10-10"></a>
### Состояние на 10.10.2026

В каталоге проекта подготовлены 22 проектных артефакта, включая весь сокращённый комплект из восьми сводных контрактов. Реализация [BL][g-work-ids]/[CW][g-work-ids], аккаунты, интеграции и испытания не подтверждены. Контрольные точки 08–09.10 уже прошли без [свидетельств][g-evidence] выполнения; они остаются исходными целями, а не будущими сроками. Общие 12–16 часов участия [владельца][g-owner] относятся ко всему периоду 08–11.10 и не считаются остатком на сегодня; фактический расход времени неизвестен.

Ближайший порядок реализации: проверить доступы/[HTTPS][g-https]/[FCM][g-push]/[Runtime][g-runtime]/[модели][g-model]/[хост][g-host] B → получить один вертикальный [APK][g-apk]/[API][g-api]/[Auth][g-auth]/run-срез с [политиками][g-policy] → связать [coder][g-main-roles]→[reviewer][g-main-roles] и мобильный [пакет][g-package] → провести матрицу переноса, отрицательные и количественные проверки → закрепить кандидат, теорию и репетицию 11.10. Эта последовательность определяет зависимости; она не подтверждает, что весь объём помещается в 10–11.10. Подготовка обязательного проектного комплекта завершена; следующий этап — реализация по [backlog][g-work-ids] и проверки [DOC-057][id-doc-057]. Критерии готовности заданы в [DOC-014](14-demo-acceptance-criteria.md); приёмка ещё не проводилась.

Защита остаётся 12.10.2026; [M-требования][g-priorities] не сняты, новые [S/C][g-priorities] с 11.10 до защиты не начинаются. Отклонение по срокам и необходимость оценки обязательного остатка фиксируются в [RSK-01](11-initial-risk-register.md#rsk-01). Общая актуальная карта — [навигатор](README.md).

## 3. Изменение границ

[Агент][g-agent] самостоятельно уточняет технический контракт и устраняет дефекты в пределах требований. Смена глобальной цели, снятие прямо обязательного результата, изменение срока или дополнительные расходы выносятся [владельцу][g-owner] с конкретным вариантом и последствиями. После решения синхронизируются [backlog][g-work-ids], [ТЗ][g-requirement-ids]/[SRS][g-requirement-ids], [SMART][g-smart], [MoSCoW][g-priorities] и этот раздел. Невыполненный M не превращается в S только из-за приближения защиты.


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
[g-audit]: 07-glossary.md#term-audit
[g-auth]: 07-glossary.md#term-auth
[g-cancelled]: 07-glossary.md#term-cancelled
[g-ci-cd]: 07-glossary.md#term-ci-cd
[g-control-plane]: 07-glossary.md#term-control-plane
[g-devcontainer]: 07-glossary.md#term-devcontainer
[g-event]: 07-glossary.md#term-event
[g-evidence]: 07-glossary.md#term-evidence
[g-export-import]: 07-glossary.md#term-export-import
[g-fixture]: 07-glossary.md#term-fixture
[g-flutter]: 07-glossary.md#term-flutter
[g-github]: 07-glossary.md#term-github
[g-go]: 07-glossary.md#term-go
[g-harness]: 07-glossary.md#term-harness
[g-host]: 07-glossary.md#term-host
[g-https]: 07-glossary.md#term-https
[g-idempotency]: 07-glossary.md#term-idempotency
[g-interrupted]: 07-glossary.md#term-interrupted
[g-main-roles]: 07-glossary.md#term-main-roles
[g-mcp]: 07-glossary.md#term-mcp
[g-model]: 07-glossary.md#term-model
[g-mvp]: 07-glossary.md#term-mvp
[g-observability-tools]: 07-glossary.md#term-observability-tools
[g-orchestration]: 07-glossary.md#term-orchestration
[g-owner]: 07-glossary.md#term-owner
[g-package]: 07-glossary.md#term-package
[g-policy]: 07-glossary.md#term-policy
[g-portability]: 07-glossary.md#term-portability
[g-pr]: 07-glossary.md#term-pr
[g-priorities]: 07-glossary.md#term-priorities
[g-push]: 07-glossary.md#term-push
[g-repository]: 07-glossary.md#term-repository
[g-requirement-ids]: 07-glossary.md#term-requirement-ids
[g-review]: 07-glossary.md#term-review
[g-role]: 07-glossary.md#term-role
[g-run]: 07-glossary.md#term-run
[g-runner]: 07-glossary.md#term-runner
[g-runtime]: 07-glossary.md#term-runtime
[g-safe-alternative]: 07-glossary.md#term-safe-alternative
[g-secret]: 07-glossary.md#term-secret
[g-smart]: 07-glossary.md#term-smart
[g-supabase]: 07-glossary.md#term-supabase
[g-task]: 07-glossary.md#term-task
[g-timeout]: 07-glossary.md#term-timeout
[g-traceability]: 07-glossary.md#term-traceability
[g-ui]: 07-glossary.md#term-ui
[g-work-ids]: 07-glossary.md#term-work-ids
[id-bl-072]: 01-product-backlog.md#bl-072
[id-bl-090]: 01-product-backlog.md#bl-090
[id-bl-119]: 01-product-backlog.md#bl-119
[id-cw-001]: 01-product-backlog.md#cw-001
[id-doc-001]: 01-product-backlog.md
[id-doc-007]: 07-glossary.md
[id-doc-010]: 10-project-scope-and-constraints.md
[id-doc-015]: 01-product-backlog.md#doc-015
[id-nfr-024]: 03-software-requirements-specification.md#nfr-024
[id-sm-02]: 04-smart-goals-and-metrics.md#sm-02
[id-sm-03]: 04-smart-goals-and-metrics.md#sm-03
[id-sm-04]: 04-smart-goals-and-metrics.md#sm-04
[id-sm-05]: 04-smart-goals-and-metrics.md#sm-05
[id-sm-06]: 04-smart-goals-and-metrics.md#sm-06
[id-sm-08]: 04-smart-goals-and-metrics.md#sm-08
[id-sm-10]: 04-smart-goals-and-metrics.md#sm-10
[id-sm-12]: 04-smart-goals-and-metrics.md#sm-12
[id-sys-01]: 02-technical-specification.md#sys-01
[id-sys-02]: 02-technical-specification.md#sys-02
[id-sys-05]: 02-technical-specification.md#sys-05
[id-sys-06]: 02-technical-specification.md#sys-06
[id-sys-07]: 02-technical-specification.md#sys-07
[id-sys-09]: 02-technical-specification.md#sys-09
[id-sys-10]: 02-technical-specification.md#sys-10
[id-sys-11]: 02-technical-specification.md#sys-11
[id-sys-17]: 02-technical-specification.md#sys-17
[g-development-agent]: 07-glossary.md#term-development-agent

[g-acceptance]: 07-glossary.md#term-acceptance

[id-doc-057]: 57-verification-and-traceability.md
