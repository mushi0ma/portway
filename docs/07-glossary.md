# Portway — Глоссарий

| Поле | Значение |
|---|---|
| Артефакт | [DOC-007][id-doc-007] |
| Версия и дата | v0.5 · 10.10.2026 |
| Роль документа | Рабочий документ; обновлять при изменении своей области; актуальный состав — [навигатор](README.md#existing) |
| Статус | Актуализирован проектный документ; реализация и испытания не подтверждены |
| Основания | [Навигатор](README.md), [backlog](01-product-backlog.md), [ТЗ](02-technical-specification.md), [SRS](03-software-requirements-specification.md), [SMART](04-smart-goals-and-metrics.md), [MoSCoW](05-moscow-prioritization.md) |
| Связанные артефакты | [Акторы и заинтересованные стороны](08-actors-and-stakeholders.md#actors), [Анализ аналогов](09-analogue-analysis.md#analogues), [Границы проекта и ограничения](10-project-scope-and-constraints.md#scope), [Первоначальный реестр рисков](11-initial-risk-register.md#risks), [Use Case](12-use-case-diagrams-and-specifications.md), [сценарии качества](13-quality-attribute-scenarios-and-utility-tree.md), [приёмка](14-demo-acceptance-criteria.md) |

<a id="documentation-plan"></a>
## Состав документации

Подготовлены все восемь сводных контрактов сокращённого плана; всего 22 проектных артефакта. [Состав и актуальный остаток](README.md#remaining), [сопоставление прежних DOC-ID](01-product-backlog.md#legacy-registry). Новых обязательных проектных документов в очереди нет. Следующая работа — реализация и проверки; [BL][g-work-ids]/[CW][g-work-ids] и все [TC][g-acceptance]/[AC][g-acceptance] остаются неподтверждёнными.

<a id="glossary"></a>
## 1. Правила употребления

Каждая словарная запись имеет устойчивый якорь term-*. Упоминания проектных терминов в тексте, таблицах и заголовках связываются с определением; при первом введении нового понятия даются русский термин и английское имя. Далее допустим краткий вариант. Имена сущностей и полей сохраняются латиницей. В пользовательском интерфейсе предпочтительны «[запуск][g-run]», «[подтверждение действия][g-approval]», «связь», «результат». Точные JSON-поля, перечисления состояний и переходы будут закреплены в контрактах и диаграммах [DOC-029][id-doc-029]/033–036, а не выведены из английского названия в этой таблице.

Код, машинные поля, URL и исходники диаграмм сохраняют точный синтаксис; термины из них объясняются здесь и в окружающем тексте. Слова «проверка», «связь», «[роль][g-role]» и «результат» связываются по контексту: обычная проверка документа не является [reviewer-run][g-run].

Глоссарий объясняет уже принятые решения и не меняет приоритеты, лимиты или состав [MVP][g-mvp]. Численные defaults и сроки хранения задаёт [SRS][g-requirement-ids] § 4.2; методы проверки — [SMART][g-smart]. Наличие термина не означает включение функции в [базовую поставку][g-mvp] или её готовность. При расхождении сначала проверяются прямые решения [владельца][g-owner] и актуальные требования.

## 2. Пользователь, [агент][g-agent] и [модель][g-model]

| Термин / имя | Значение в Portway |
|---|---|
| <a id="term-owner"></a>[Владелец (Owner)](#term-owner) | Единственный пользователь конкретной [установки][g-host], допущенный серверной проверкой [owner][g-owner] ID. Также принимает глобальные решения по проекту. |
| <a id="term-development-agent"></a>[ИИ-агент разработки проекта](#term-development-agent) | Помощник [владельца][g-owner], создающий Portway и его документы. Не тождественен [агенту][g-agent], настроенному внутри продукта. |
| <a id="term-agent"></a>[Агент (Agent)](#term-agent) | Сохранённое описание исполнителя: инструкции, [роль][g-role], [навыки][g-skill], [инструменты][g-tool], [MCP][g-mcp] и настройки [модели][g-model]. Само описание не является работающим процессом. |
| <a id="term-agent-version"></a>[Версия агента (AgentVersion)](#term-agent-version) | Неизменяемый снимок конфигурации. Правка создаёт новую версию; существующий [запуск][g-run] сохраняет исходную версию, а действующие запреты всё равно применяются. |
| <a id="term-role"></a>[Роль / пресет (Role / preset)](#term-role) | Готовый набор инструкций, [инструментов][g-tool] и прав [агента][g-agent]. Не [роль][g-role] пользователя и не способ получить полномочия [владельца][g-owner]. |
| <a id="term-main-roles"></a>[Основные роли](#term-main-roles) | [`coder`][g-main-roles], [`reviewer`][g-main-roles], [`tester`][g-main-roles], [`researcher`][g-main-roles], [`documenter`][g-main-roles]. Каталог доступных [пресетов][g-role], а не пять обязательных одновременно работающих [агентов][g-agent]. |
| <a id="term-subagent-presets"></a>[Пресеты субагентов](#term-subagent-presets) | [`code_reader`][g-subagent-presets], [`test_runner`][g-subagent-presets], [`docs_lookup`][g-subagent-presets]: отдельная группа специализированных [ролей][g-role]. Выбор [пресета][g-role] сам по себе не создаёт дочерний [запуск][g-run]. |
| <a id="term-skill"></a>[Навык (Skill)](#term-skill) | Набор инструкций и ресурсов для определённого вида работы [агента][g-agent]; переносимый [пакет][g-package] предусматривает `skills/*/SKILL.md`. Не разрешение обойти [политику][g-policy]. |
| <a id="term-model"></a>[Языковая модель (LLM / model)](#term-model) | Выбранная [модель][g-model], генерирующая ответ и предложения вызовов [инструментов][g-tool]. [Модель][g-model] не равна [агенту][g-agent] и сама не получает прямой доступ к [хосту][g-host] или [секретам][g-secret]. |
| <a id="term-provider"></a>[Провайдер (Provider)](#term-provider) | Настроенное подключение к модельному сервису: адрес, [model][g-model] ID и при необходимости [ссылка на секрет][g-secret-reference]. Может вести к локальному или удалённому сервису. |
| <a id="term-model-endpoint"></a>[Совместимый endpoint](#term-model-endpoint) | Адрес сервиса с проверенным поддерживаемым [OpenAI-совместимым протоколом][g-model-endpoint]. Похожий URL не доказывает совместимость [stream][g-model-protocol]/[tool calls][g-tool]/[usage][g-usage]/errors. |
| <a id="term-gateway"></a>[LLM-шлюз](#term-gateway) | Подсистема [control plane][g-control-plane], которая приводит запросы, ответы, поток, вызовы [инструментов][g-tool] и ошибки [провайдера][g-provider] к внутреннему формату. |
| <a id="term-byok"></a>[BYOK](#term-byok) | Использование собственного ключа доступа [владельца][g-owner] к [провайдеру][g-provider]. Ключ хранится на сервере; локальный [endpoint][g-model-endpoint] может работать без него. |

## 3. Исполнение и результат

| Термин / имя | Значение в Portway |
|---|---|
| <a id="term-task"></a>[Задача](#term-task) | Пользовательское описание требуемого результата. В [backlog][g-work-ids] слово «[задача][g-task]» означает работу над продуктом с [BL][g-work-ids]/[CW-ID][g-work-ids]; это отдельный контекст. |
| <a id="term-run"></a>[Запуск (Run)](#term-run) | Один экземпляр выполнения [задачи][g-task] с собственным ID, [версией агента][g-agent-version], рабочей [ревизией][g-repository], состоянием и лимитами. Повторный ручной [запуск][g-run] после сбоя получает новый ID. |
| <a id="term-step"></a>[Шаг (Step)](#term-step) | Учитываемая итерация [цикла агента][g-harness]. Точная граница записи [шага][g-step] и [событий][g-event] уточняется контрактом; не отдельный пользователь или [модель][g-model]. |
| <a id="term-harness"></a>[Цикл агента (Harness)](#term-harness) | Управляемый [control plane][g-control-plane] цикл: запрос [LLM][g-model] → проверка предложенного действия → [инструмент][g-tool] → результат → дальнейшее решение; учитывает отмену и бюджеты. |
| <a id="term-tool"></a>[Инструмент (Tool) / вызов (ToolCall)](#term-tool) | Разрешённая операция и её конкретное выполнение с аргументами: файлы, shell, тесты, git или [MCP][g-mcp]. Предложенный [моделью][g-model] вызов ещё не является выполненным действием. |
| <a id="term-workspace"></a>[Рабочая область (Workspace)](#term-workspace) | Изолированные файлы конкретной работы. Доступ ограничен [политикой][g-policy]; [workspace][g-workspace] не означает произвольный каталог [хоста][g-host]. |
| <a id="term-repository"></a>[Репозиторий и ревизия](#term-repository) | Исходный код и зафиксированное состояние, по которому выполняется [задача][g-task]. В базовом демо используется локальный тестовый [репозиторий][g-repository] без [GitHub-интеграции][g-github]. |
| <a id="term-result"></a>[Результат (Result)](#term-result) | Наблюдаемый итог [запуска][g-run]: изменения/вывод и связанная [ревизия][g-repository]. Результат доступен независимо от наличия [PR][g-pr]. |
| <a id="term-review"></a>[Проверка (Review) / вердикт](#term-review) | Отдельный [запуск][g-run] [reviewer][g-main-roles] по зафиксированному [результату coder][g-result] и его заключение. Отрицательный [вердикт][g-review] не создаёт автоматический цикл исправлений. |
| <a id="term-control-plane"></a>[Control plane](#term-control-plane) | [Go-сервер][g-go] управления: [API][g-api], доступ, [Harness][g-harness], [модели][g-model], [политики][g-policy], [сообщения][g-message], бюджеты, [секреты][g-secret] и [пакеты][g-package]. |
| <a id="term-runtime"></a>[Runtime](#term-runtime) | Исполнитель разрешённых [инструментов][g-tool] в ограниченной среде. Не [LLM-шлюз][g-gateway] и не весь [control plane][g-control-plane]. |
| <a id="term-sandbox"></a>[Песочница / контейнер](#term-sandbox) | Изолированная среда [запуска][g-run] с ограничениями прав, ресурсов и сети. [Контейнер][g-sandbox] сам по себе не доказывает соблюдение всех ограничений. |
| <a id="term-devcontainer"></a>[devcontainer](#term-devcontainer) | Конфигурация среды [репозитория][g-repository]. [MVP][g-mvp] принимает разрешённое подмножество; произвольные lifecycle scripts, privileged и host mounts не поддерживаются. |
| <a id="term-runner"></a>[Runner](#term-runner) | Отдельный исполнитель, подключаемый исходящим соединением. Расширение S вне [базовой поставки][g-mvp]; локальный [Runtime][g-runtime] и перенос на другой [хост][g-host] не требуют [Runner][g-runner]. |
| <a id="term-mcp"></a>[MCP](#term-mcp) | [Model][g-model] Context Protocol; в проекте — подключение [инструментов][g-tool] внешнего сервиса через ограниченный адаптер. Один [MCP][g-mcp] документации обязателен; [Context7][g-context7] пока кандидат. |
| <a id="term-pr"></a>[PR (Pull Request)](#term-pr) | Предложение включить изменения в [GitHub-репозиторий][g-github]. Публикация через [GitHub App][g-github] — S вне базового [MVP][g-mvp]; не синоним результата или [review][g-review]. |

## 4. [Оркестрация][g-orchestration] и доставка

| Термин / имя | Значение в Portway |
|---|---|
| <a id="term-orchestration"></a>[Оркестрация](#term-orchestration) | Управление взаимодействием [агентов][g-agent], [событиями][g-event], ограничениями и результатом. [MVP][g-mvp]: ручная связь [coder][g-main-roles]→[reviewer][g-main-roles] и серверные правила, без агента-планировщика. |
| <a id="term-link"></a>[Связь](#term-link) | Разрешённое [владельцем][g-owner] направление взаимодействия [агентов][g-agent]. [Агент][g-agent] не расширяет топологию самостоятельно. |
| <a id="term-hook"></a>[Хук (Hook)](#term-hook) | Настроенная реакция на поддерживаемое [событие][g-event]. [Хук][g-hook] завершения [coder][g-main-roles] создаёт отдельный [reviewer-run][g-run]; повтор [события][g-event] не должен создавать второй логический [review][g-review]. |
| <a id="term-chain"></a>[Цепочка (Chain) / `correlation_id`](#term-chain) | Связанная работа нескольких [запусков][g-run] и [сообщений][g-message] с общим контекстом, бюджетом и остановкой. Идентификатор связывает их; присланное [агентом][g-agent] новое значение не сбрасывает лимиты. |
| <a id="term-message"></a>[Сообщение (Message) / конверт](#term-message) | Передаваемые [агенту][g-agent] данные и служебные поля: версия схемы, ID, отправитель/получатель, тип, [payload][g-payload], связь с [run][g-run]/[цепочкой][g-chain] и счётчики. [Конверт][g-message] валидируется сервером. |
| <a id="term-run-id"></a>[`run_id` / `reply_to`](#term-run-id) | Идентификатор [запуска][g-run]; в [конверте][g-message] принадлежит отправителю. `reply_to` связывает ответ с исходным [сообщением][g-message]. Ответ завершённому [coder][g-main-roles] не возобновляет его [запуск][g-run]. |
| <a id="term-payload"></a>[Payload](#term-payload) | Содержимое [сообщения][g-message]/[события][g-event]. Не доверенная системная инструкция и не источник новых прав. |
| <a id="term-hops"></a>[Hops](#term-hops) | Счётчик передач [сообщения][g-message] в [цепочке][g-chain]; ответ также увеличивает его. Считается сервером, ограничивает распространение работы. |
| <a id="term-depth"></a>[Depth](#term-depth) | Глубина дочерних [запусков][g-run]: корень — 0, новый дочерний [run][g-run] — следующий уровень. Не число [сообщений][g-message] и не число доступных [ролей][g-role]. |
| <a id="term-event"></a>[Событие (Event)](#term-event) | Сохранённая запись о произошедшем изменении/результате с ID, контекстом и серверным временем. [Сообщение][g-message] передаёт данные [агенту][g-agent]; [событие][g-event] сообщает о факте. |
| <a id="term-outbox"></a>[Transactional outbox](#term-outbox) | Сохранение изменения и записи для последующей доставки в одной транзакции [БД][g-supabase]. Не обещание однократной сетевой доставки. |
| <a id="term-at-least-once"></a>[At-least-once](#term-at-least-once) | Выбранный режим с возможными повторными доставками. Не означает, что операция обязана повторно исполняться при каждом получении. |
| <a id="term-idempotency"></a>[Дедупликация / идемпотентность](#term-idempotency) | Распознавание повторов и сохранение одного логического эффекта в установленном окне. Один ID с отличающимся [payload][g-payload] отклоняется. |
| <a id="term-retry"></a>[Retry / DLQ](#term-retry) | Повтор доставки в пределах правил; [DLQ][g-retry] — отделение исчерпавших доставку [сообщений][g-message] с наблюдаемым отказом. Это не автоматическое повторение опасного действия с неизвестным исходом. |
| <a id="term-sse"></a>[SSE / курсор](#term-sse) | Server-Sent [Events][g-event] — поток [событий][g-event] сервер→клиент. [Курсор][g-sse] позволяет получить сохранённые [события][g-event] после обрыва; срок хранения ограничен, порядок задаётся в пределах [run][g-run]. |

## 5. Доступ, подтверждения и ограничения

| Термин / имя | Значение в Portway |
|---|---|
| <a id="term-auth"></a>[Аутентификация / авторизация](#term-auth) | Проверка личности через [Supabase Auth][g-supabase] и серверная проверка права на операцию. Вход другого Supabase-пользователя не делает его [владельцем][g-owner] Portway. |
| <a id="term-device-session"></a>[Setup-доступ / device session](#term-device-session) | [Setup proof][g-device-session] ограничивает первичную регистрацию, [device proof][g-device-session] — случайный непрозрачный [секрет][g-secret] устройства. В [APK][g-apk] secure storage, на сервере hash; [JWT][g-jwt-jwks]+активный [device proof][g-device-session] проверяются вместе. Отзыв описан в [DOC-036][id-doc-036]/040. |
| <a id="term-policy"></a>[Политика (Policy)](#term-policy) | Серверные правила разрешения, запрета или требования [подтверждения действия][g-approval]. Текст [модели][g-model], [сообщения][g-message] и результаты [инструментов][g-tool] не могут повысить права. |
| <a id="term-approval"></a>[Апрув (Approval)](#term-approval) | Запрос и решение [владельца][g-owner] по конкретному [инструменту][g-tool] и неизменным аргументам до истечения срока. В [UI][g-ui] — «[подтверждение действия][g-approval]»; не общий допуск на будущие команды. |
| <a id="term-approval-decision"></a>[Approve / reject / deny](#term-approval-decision) | [Approve][g-approval-decision] — подтверждение [владельца][g-owner]; [reject][g-approval-decision] — его отказ; [deny][g-approval-decision] — запрет [политики][g-policy]. Отказ не разрешает выполнить первоначальное действие. |
| <a id="term-timeout"></a>[TTL / timeout](#term-timeout) | [TTL][g-timeout] — срок действительности записи/запроса; [timeout][g-timeout] — истечение отведённого времени ожидания/исполнения. Истёкший [апрув][g-approval] не может разрешить действие. |
| <a id="term-safe-alternative"></a>[Безопасная альтернатива](#term-safe-alternative) | Новое предложение после отказа, проходящее [политику][g-policy] заново. Повтор тех же запрещённых аргументов альтернативой не считается; [stop][g-cancelled] и бюджет прекращают продолжение. |
| <a id="term-budget"></a>[Бюджет run/chain](#term-budget) | Ограничения времени, [шагов][g-step] и токенов отдельного [запуска][g-run]/всей [цепочки][g-chain]. Общий остаток резервируется до конкурентных вызовов. |
| <a id="term-usage"></a>[LLM-токен / usage](#term-usage) | Единица объёма входа/выхода [модели][g-model] и сведения о расходе. Неизвестный [usage][g-usage] не равен нулю; при невозможности ограничить оценку следующий платный вызов блокируется как [`budget_uncertain`][g-budget]. |
| <a id="term-secret"></a>[Credential / секрет](#term-secret) | Данные доступа: пароль, ключ или [токен авторизации][g-secret]. Серверные [LLM][g-model]/DB/[FCM][g-push]/App-секреты не попадают в [пакет][g-package], [APK][g-apk] или [логи][g-audit]; выдача временного Git [credential][g-secret] ограничена доверенной операцией. Необходимые пользовательские токены клиент хранит отдельно защищённым способом. |
| <a id="term-secret-reference"></a>[Secret reference / alias](#term-secret-reference) | Ссылка на необходимый [секрет][g-secret] либо переносимое имя связи/[роли][g-role]. Позволяет сопоставить значение/объект на целевом [хосте][g-host] без [экспорта][g-export-import] самого [секрета][g-secret] и исходных server ID. |
| <a id="term-push-token"></a>[Push token](#term-push-token) | Адресная привязка [установки][g-host] [Android][g-flutter] для доставки [FCM][g-push]; не пользовательская сессия и не разрешение выполнить [инструмент][g-tool]. |
| <a id="term-audit"></a>[Аудит / логи / метрики](#term-audit) | [Аудит][g-audit] фиксирует субъект, действие и исход; [логи][g-audit] помогают разбирать работу приложения; [метрики][g-audit] описывают численные характеристики. [Метрики][g-audit] не заменяют журнал [аудита][g-audit]. |

Три разных значения слова «токен»: **[LLM-токен][g-usage]** измеряет объём, **[токен авторизации][g-secret]** предоставляет ограниченный доступ, **[FCM token][g-push-token]** адресует уведомление. В требованиях и интерфейсных контрактах всегда указывается вид.

## 6. Состояния и восстановление

Это пояснение смысла терминов из требований, а не полный enum [API][g-api]. Полная [модель][g-model] состояний, допустимые переходы и терминальные причины — [DOC-033][id-doc-033].

| Термин | Значение и ограничение |
|---|---|
| <a id="term-active-run"></a>[Активный запуск](#term-active-run) | Незавершённая управляемая работа; ожидание [LLM][g-model]/[инструмента][g-tool]/[апрува][g-approval] не равно загрузке отдельной [модели][g-model]. Два активных [run][g-run] могут делить один модельный сервис. |
| <a id="term-awaiting-approval"></a>[Ожидание апрува](#term-awaiting-approval) | Конкретное действие приостановлено до действительного решения. Ожидание учитывается в лимите времени [run][g-run]/[chain][g-chain]. |
| <a id="term-completed"></a>[`completed`](#term-completed) | [Запуск][g-run] завершён с результатом; это не автоматически положительный [review][g-review], созданный [PR][g-pr] или выполненная вся [цепочка][g-chain]. |
| <a id="term-interrupted"></a>[`failed` / `interrupted`](#term-interrupted) | [`failed`][g-interrupted] — неуспешное завершение; [`interrupted`][g-interrupted] — причина после обнаруженного прерывания при рестарте. Запись «[failed][g-interrupted]/[interrupted][g-interrupted]» не задаёт два отдельных состояния. |
| <a id="term-cancelled"></a>[`cancelled` / stop](#term-cancelled) | Отменённый исход и команда остановки. Принятие [stop][g-cancelled] сервером ещё не доказывает прекращение процессов; уже совершённый внешний эффект не откатывается автоматически. |
| <a id="term-unknown"></a>[`unknown`](#term-unknown) | Неизвестен исход внешней операции. Не следует автоматически считать её ни выполненной, ни отменённой; слепой повтор опасной операции запрещён. |
| <a id="term-terminal-run"></a>[Терминальный запуск](#term-terminal-run) | [Запуск][g-run] с окончательным исходом. Новый ручной [запуск][g-run] получает новый ID; старый не «оживает» от позднего [сообщения][g-message]. |
| <a id="term-recovery"></a>[Recovery / reconciliation](#term-recovery) | Сверка сохранённого состояния с прерванной работой при старте; старые [апрувы][g-approval] недействительны. Не автоматическое продолжение прежнего [Harness][g-harness]. |
| <a id="term-cleanup"></a>[Cleanup](#term-cleanup) | Остановка процессов и уборка временной среды/[credentials][g-secret] с сохранением результата по [политике][g-policy] хранения. Не удаление экспортированных [владельцем][g-owner] [пакетов][g-package]. |

## 7. Телефон, [пакет][g-package] и размещение

| Термин / имя | Значение в Portway |
|---|---|
| <a id="term-apk"></a>[APK / Flutter-клиент](#term-apk) | Устанавливаемое [Android-приложение][g-flutter] для управления сервером. [APK][g-apk] не делает адрес сервера ненужным и не является местом исполнения серверных [инструментов][g-tool]. |
| <a id="term-push"></a>[Push / FCM](#term-push) | Уведомление через Firebase Cloud Messaging о необходимости открыть актуальное состояние. [Push][g-push] не разрешает действие; [APK][g-apk] перечитывает запрос после проверки входа. |
| <a id="term-app-lifecycle"></a>[Background / terminated / force-stop](#term-app-lifecycle) | Фон; обычное закрытое состояние приложения; принудительная остановка через настройки [Android][g-flutter]. Это разные условия приёмки [push][g-push], зафиксированные в [SRS][g-requirement-ids] § 4.2. |
| <a id="term-https"></a>[HTTPS endpoint / туннель / домен](#term-https) | Адрес защищённого [API][g-api]; способ предоставить доступ к серверу на ПК; имя адреса. Отдельно купленный [домен][g-https] не обязателен, доступность [API][g-api] с телефона обязательна. |
| <a id="term-host"></a>[Установка / хост](#term-host) | Настроенное окружение Portway; машина/ВМ, где исполняются [control plane][g-control-plane] и [Runtime][g-runtime]. [Supabase][g-supabase] и [FCM][g-push] остаются внешними зависимостями выбранного [MVP][g-mvp]. |
| <a id="term-self-hosted"></a>[Self-hosted исполнение](#term-self-hosted) | [Control plane][g-control-plane] и [инструменты][g-tool] работают на [хосте][g-host] [владельца][g-owner]. В данном проекте не означает полностью автономную работу без внешних сервисов. |
| <a id="term-package"></a>[Agent Package](#term-package) | ZIP с конфигурацией выбранной версии: manifest, инструкции, [навыки][g-skill], [MCP][g-mcp] и переносимые ссылки. Без секретных значений, полной памяти и активных [run][g-run]. |
| <a id="term-export-import"></a>[Export / import](#term-export-import) | Создание [пакета][g-package] и проверяемое восстановление конфигурации с новыми локальными привязками. [Импорт][g-export-import] не исполняет включённый код и не активирует частично проверенного [агента][g-agent]. |
| <a id="term-portability"></a>[Переносимость / A/M1 → A/M2 → B/M2](#term-portability) | Проверка одной контрольной [задачи][g-task] при смене [модели][g-model], затем [хоста][g-host]. A/B — разные [установки][g-host] на независимых [хостах][g-host]; явно обозначенная ВМ допустима, второй каталог A недостаточен. |
| <a id="term-ci-cd"></a>[CI / CD / deployment](#term-ci-cd) | Автоматические проверки; процесс доставки изменений; развёртывание выбранной версии. Для [MVP][g-mvp] обязателен [CI][g-ci-cd], автоматический production-деплой из этого не следует. |
| <a id="term-runbook"></a>[Rollback / runbook](#term-runbook) | Возврат к совместимой рабочей версии и инструкция эксплуатации/восстановления. Порядок и ограничения, включая данные/[миграции][g-environment], будут определены в документах развёртывания. |

## 8. Проектирование и проверка

| Термин / обозначение | Значение |
|---|---|
| <a id="term-mvp"></a>[MVP / базовая поставка](#term-mvp) | Обязательный работающий срез к защите 12.10.2026 по [MoSCoW][g-priorities] и [SMART][g-smart]. Не весь долгосрочный [backlog][g-work-ids]. |
| <a id="term-priorities"></a>[M / S / C / W](#term-priorities) | [Must][g-priorities] — обязательное; [Should][g-priorities] — ценное расширение; [Could][g-priorities] — дополнительное; [Won’t][g-priorities] — вне текущей версии. [S/C][g-priorities] не входят в базовый календарь. |
| <a id="term-work-ids"></a>[BL / CW / SP / DOC](#term-work-ids) | [Задача][g-task] реализации; работа по курсовой; проверка неопределённости; проектный артефакт. Количество строк не является оценкой часов. |
| <a id="term-requirement-ids"></a>[SYS / QLT / FR / NFR](#term-requirement-ids) | Системные/качественные требования [ТЗ][g-requirement-ids]; функциональные/нефункциональные требования [SRS][g-requirement-ids]. Численные критерии берутся из актуальных требований. |
| <a id="term-decision-ids"></a>[D / Q / TBD](#term-decision-ids) | Принятое решение; вопрос [ТЗ][g-requirement-ids]; уточнение [SRS][g-requirement-ids]. Сохранённый TBD-ID может уже иметь закрытое решение. |
| <a id="term-smart"></a>[SM / MX / PERF-01](#term-smart) | [SMART-цель][g-smart]; метод измерения; согласованный профиль нагрузки. Целевой показатель не равен измеренному факту. |
| <a id="term-fixture"></a>[Fixture / fake / smoke / E2E](#term-fixture) | Контрольные данные/[задача][g-task]; тестовый [дублёр][g-fixture]; ранняя проверка подключения; сквозная проверка. [Дублёр][g-fixture] не закрывает требование реального [Android][g-flutter]/[FCM][g-push]/[Runtime][g-runtime]/[модели][g-model]/переноса. |
| <a id="term-evidence"></a>[p95 / свидетельство (evidence)](#term-evidence) | 95-й перцентиль измеренной выборки; проверяемое подтверждение результата: протокол, вывод, запись, артефакт с версией. Методику задаёт [SMART][g-smart]. |
| <a id="term-uml"></a>[State / Sequence / Activity / Class / Component](#term-uml) | Диаграммы жизненного цикла; взаимодействий во времени; действий; структуры типов/сущностей; компонентов и интерфейсов. Планируются в [DOC-024][id-doc-024]/030/033/039/040. |
| <a id="term-architecture-notations"></a>[C4 / BPMN / DFD / ER](#term-architecture-notations) | Представления архитектуры; [модель][g-model] бизнес-процессов; потоки данных; структура данных и связей. [C4 Component][g-architecture-notations] и [UML Component][g-uml] в [DOC-024][id-doc-024] представлены отдельно. |
| <a id="term-traceability"></a>[ADR / RTM / ПМИ](#term-traceability) | [ADR][g-traceability] — запись решения (16 записей [DOC-020][id-doc-020]); [RTM][g-traceability] — связь требований/реализации/проверок, [ПМИ][g-traceability] — метод испытаний ([DOC-057][id-doc-057]). Проектные документы подготовлены; реализация и фактические результаты отдельно. |



## 9. Интерфейсы, проектные артефакты и дополнительные обозначения

| Термин / имя | Значение в Portway |
|---|---|
| <a id="term-api"></a>[API / OpenAPI](#term-api) | Прикладной интерфейс [APK][g-apk] и [Go][g-go]. HTTP DTO, ошибки, [SSE][g-sse] и envelope заданы в [DOC-036][id-doc-036]; исполняемый [OpenAPI][g-api] YAML появляется с кодом и контрактными проверками. |
| <a id="term-supabase"></a>[Supabase / PostgreSQL / RLS](#term-supabase) | Внешний сервис [Auth][g-auth] и рабочая [PostgreSQL][g-supabase]. [RLS][g-supabase] и права ограничивают доступ к бизнес-таблицам; [APK][g-apk] получает бизнес-данные через [Go][g-go] [API][g-api]. Локальная [БД][g-supabase] допустима в тестах. |
| <a id="term-flutter"></a>[Flutter / Android](#term-flutter) | [Flutter][g-flutter] — технология обязательного [Android-клиента][g-flutter]; принимается [APK][g-apk] на [Android 16][g-flutter]. Наличие термина не добавляет Web/iOS/Desktop в [MVP][g-mvp]. |
| <a id="term-go"></a>[Go / модульный монолит](#term-go) | Один серверный процесс с модулями управления, [Harness][g-harness], gateway, [policy][g-policy], [outbox][g-outbox] и [аудита][g-audit]. [Runtime][g-runtime] выполняет [инструменты][g-tool] отдельно. Контракты модулей определятся архитектурными артефактами. |
| <a id="term-github"></a>[GitHub App / installation token](#term-github) | Необязательная S-интеграция с ограниченным [credential][g-secret]. Чтение [репозитория][g-repository] и публикация имеют разные права; публикация после положительного [review][g-review] и отдельного [апрува][g-approval]. |
| <a id="term-context7"></a>[Context7](#term-context7) | Первый кандидат внешнего [MCP][g-mcp] документации; доступность и протокол ещё проверяются. Нужен один реальный выбранный [MCP][g-mcp], имя кандидата не означает успешное подключение. |
| <a id="term-tunnel-providers"></a>[ngrok / Cloudflare Tunnel / Railway](#term-tunnel-providers) | Варианты сетевого доступа/размещения, а не реализованные подключения. Первый кандидат — ПК с [ngrok][g-tunnel-providers]; ограничения [SSE][g-sse] и способ размещения [Runtime][g-runtime] проверяются до выбора альтернативы. |
| <a id="term-use-case"></a>[Use Case / UC](#term-use-case) | Сценарий взаимодействия внешнего [актора][g-actors] с Portway. [DOC-012][id-doc-012] содержит 20 обязательных [UC][g-use-case]; код [UC][g-use-case] не является идентификатором теста. |
| <a id="term-quality-scenario"></a>[Сценарий качества / QS / Utility Tree](#term-quality-scenario) | [QS][g-quality-scenario] задаёт источник, воздействие, условия, артефакт, ответ и меру. [Utility Tree][g-quality-scenario] связывает [QS][g-quality-scenario] с атрибутами; I/D и H/M/L — оценки важности/трудности, отдельные от [MoSCoW][g-priorities]. [DOC-013][id-doc-013] не является проведённой ATAM-экспертизой. |
| <a id="term-acceptance"></a>[Критерий приёмки / AC / тест-кейс / TC](#term-acceptance) | [AC][g-acceptance] задаёт обязательный результат ([DOC-014][id-doc-014]), [TC][g-acceptance] — входы, действия, oracle и [evidence][g-evidence] (32 строки [DOC-057][id-doc-057]). NOT_RUN/PASS/FAIL/BLOCKED/INCONCLUSIVE — статусы проверки; OUT_OF_BASE относится к исключённому S/C требованию, не к состоянию [run][g-run]. |
| <a id="term-vision"></a>[Vision](#term-vision) | Краткое видение пользователя, проблемы, ценности и основного сценария ([DOC-006][id-doc-006]); приоритеты и численные ограничения определяют [MoSCoW][g-priorities]/[SRS][g-requirement-ids]. |
| <a id="term-planning"></a>[Roadmap / WBS / Gantt](#term-planning) | Порядок вех, декомпозиция работ и календарное представление оценок. Краткий план и оценки ведутся в [backlog](01-product-backlog.md); самостоятельные [DOC-064][id-doc-064]/065 сняты с очереди; перечень строк и целевые даты не подтверждают достижимость плана. |
| <a id="term-stride"></a>[STRIDE / модель угроз](#term-stride) | Структура анализа угроз по потокам данных/границам доверия. Угрозы и контроли объединены в сводном [DOC-040][id-doc-040] (прежние [DOC-048][id-doc-048]/049); список открытых [RSK][g-risk] не заменяет [модель угроз][g-stride]. |
| <a id="term-observability-tools"></a>[Prometheus / Grafana](#term-observability-tools) | Дополнительные [метрики][g-audit]/панели, [Could][g-priorities] вне базы. Базовые [аудит][g-audit], [логи][g-audit] и диагностика [Go][g-go] обязательны без этих [инструментов][g-tool]. |
| <a id="term-ui"></a>[UI / User Flow / wireframe](#term-ui) | Представление экранов и пользовательского пути. Потоки/экраны/нужные эскизы объединены в [DOC-061](61-android-screens-flows-and-api.md); технический контракт и код не включаются в продуктовый путь без пользовательской необходимости. |
| <a id="term-architecture-doc"></a>[arc42 / архитектурный документ](#term-architecture-doc) | Раздел сводной архитектуры [DOC-020](20-architecture-and-decisions.md), включающий прежний [DOC-070][id-doc-070] со ссылками на решения, [C4][g-architecture-notations]/[UML][g-uml], контракты и развёртывание; термин не означает готовность этого документа. |
| <a id="term-environment"></a>[Environment / compose / migration](#term-environment) | Конфигурация окружения/[запуска][g-run] и управляемое изменение схемы [БД][g-supabase]. [Секреты][g-secret] задаются вне [репозитория][g-repository]; ошибка конфигурации/[миграции][g-environment] блокирует [ready][g-environment]. |
| <a id="term-hooks-events"></a>[run.completed / tool_result / approval_requested / agent_message](#term-hooks-events) | Четыре поддержанных event_type [hooks][g-hook]. Каталог и versioned payload закреплены в [DOC-036][id-doc-036]; [run][g-run].completed создаёт только настроенный отдельный [review][g-review], [terminal run][g-terminal-run] не возобновляет. |
| <a id="term-host-resources"></a>[vCPU / GiB / MiB / non-root](#term-host-resources) | Единицы ресурсов и ограниченный пользователь [контейнера][g-sandbox]. Численные defaults берутся из [SRS][g-requirement-ids] § 4.2; лимиты [контейнеров][g-sandbox] не включают [модель][g-model] и сборку [APK][g-apk]. |
| <a id="term-model-protocol"></a>[Streaming / tool calling / inference](#term-model-protocol) | Частичный ответ [модели][g-model], предложения вызовов и работа модельного сервиса. Возможности проверяются на конкретных [model][g-model] ID; два активных [run][g-run] могут пользоваться очередью одного inference-сервиса. |

| <a id="term-actors"></a>[Актор / стейкхолдер / ACT / STK](#term-actors) | [Актор][g-actors] взаимодействует с системой через её границу, [стейкхолдер][g-actors] заинтересован в результате/решениях. [ACT-01][id-act-01]–08 и [STK-01][id-stk-01]–04 ведутся в [DOC-008][id-doc-008]. Coder/reviewer — внутренние исполнители, [ИИ-агент][g-development-agent] разработки — участник проекта. |
| <a id="term-risk"></a>[Риск / RSK](#term-risk) | Возможное [событие][g-event] с влиянием на результат; сигнал, реакция и ответственность ведутся в [DOC-011][id-doc-011]. Наблюдаемый сигнал отставания [RSK-01][id-rsk-01] не закрывается наличием плана; [DOC-067][id-doc-067] объединён с обновлением того же реестра; отдельный файл не создаётся. |

## 10. Механизмы подготовленных контрактов

| Термин | Значение |
|---|---|
| <a id="term-jwt-jwks"></a>[JWT / JWKS](#term-jwt-jwks) | Подписанный пользовательский access token и опубликованный набор публичных signing keys [Auth][g-auth]. Signature/expiry/issuer/audience/subject проверяет сервер; валидная подпись сама по себе не доказывает отсутствие отзыва device/session. [DOC-040 § identity-secrets](40-agent-execution-and-security.md#identity-secrets). |
| <a id="term-atomicity"></a>[Транзакция / CAS / row lock / UnitOfWork](#term-atomicity) | Одна согласованная запись, условное изменение ожидаемой версии и блокировка конкурирующих агрегатов. Граница атомарности ограничена БД; внешний [tool][g-tool]/HTTP эффект не становится exactly-once от SQL commit. [DOC-029 § indexes](29-data-and-state-model.md#indexes). |
| <a id="term-fencing"></a>[Fencing / grant / dispatch_id / stop_epoch](#term-fencing) | Scoped разрешение immutable [tool][g-tool] call и идентичность выдачи. Epoch прекращает новые выдачи после [stop][g-cancelled]; [grant][g-fencing] проверяет trusted Manager, duplicate dispatch не создаёт второй процесс. [DOC-040][id-doc-040]#[approval][g-approval]. |
| <a id="term-config-hash"></a>[SHA-256 / config_hash / canonical configuration](#term-config-hash) | Hash нормализованных согласованных полей либо байтов файла. Portable config hash исключает target bindings; hash не является подписью автора или гарантией functional result. [DOC-038 § normalization](38-agent-package-format.md#normalization). |
| <a id="term-encryption"></a>[AES-256-GCM / nonce / key_version / HMAC-SHA-256](#term-encryption) | Authenticated encryption [секретов][g-secret] с отдельным ключом/уникальным [nonce][g-encryption] и версиями; HMAC отдельно аутентифицирует [Runtime][g-runtime] [grant][g-fencing]. Шифрование, checksum и полномочия имеют разное назначение. [DOC-040 § identity-secrets](40-agent-execution-and-security.md#identity-secrets). |
| <a id="term-staging"></a>[Staging / mapping / activation / unresolved](#term-staging) | Временная проверенная область [импорта][g-export-import], сопоставление target dependencies и атомарное создание [AgentVersion][g-agent-version]. [Unresolved][g-staging] блокирует [activation][g-staging]/[run][g-run]; данные [пакета][g-package] не исполняются. [DOC-038 § pipeline](38-agent-package-format.md#pipeline). |
| <a id="term-readiness"></a>[Health / readiness / degraded](#term-readiness) | [Health][g-readiness] сообщает жизнь процесса, [ready][g-readiness] — допустимость работы после [миграций][g-environment]/[reconciliation][g-recovery]; состояние внешних зависимостей имеет причину и checked_at. [FCM][g-push] accepted не равен доставке. [DOC-036 § operations](36-api-events-and-messages.md#operations). |
| <a id="term-accessibility"></a>[Доступность UI / dp / TalkBack](#term-accessibility) | Размеры touch targets в независимых от плотности единицах, семантика screen reader, масштабирование текста и контраст. Проектные критерии [DOC-061][id-doc-061]#[accessibility][g-accessibility] проверяются на реальном [APK][g-apk]. |
| <a id="term-identifier-formats"></a>[UUID / UTC / RFC 3339 / UTF-8](#term-identifier-formats) | Форматы уникальной идентичности, времени и текста контрактов. [UTC][g-identifier-formats] хранится сервером, [RFC3339][g-identifier-formats] передаётся в [API][g-api], локальная зона — представление [UI][g-ui]; [UUID][g-identifier-formats] сам по себе не разрешает доступ. |

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
[g-acceptance]: 07-glossary.md#term-acceptance
[g-agent]: 07-glossary.md#term-agent
[g-agent-version]: 07-glossary.md#term-agent-version
[g-api]: 07-glossary.md#term-api
[g-apk]: 07-glossary.md#term-apk
[g-approval]: 07-glossary.md#term-approval
[g-approval-decision]: 07-glossary.md#term-approval-decision
[g-architecture-notations]: 07-glossary.md#term-architecture-notations
[g-audit]: 07-glossary.md#term-audit
[g-auth]: 07-glossary.md#term-auth
[g-budget]: 07-glossary.md#term-budget
[g-cancelled]: 07-glossary.md#term-cancelled
[g-chain]: 07-glossary.md#term-chain
[g-ci-cd]: 07-glossary.md#term-ci-cd
[g-context7]: 07-glossary.md#term-context7
[g-control-plane]: 07-glossary.md#term-control-plane
[g-environment]: 07-glossary.md#term-environment
[g-event]: 07-glossary.md#term-event
[g-evidence]: 07-glossary.md#term-evidence
[g-export-import]: 07-glossary.md#term-export-import
[g-fixture]: 07-glossary.md#term-fixture
[g-flutter]: 07-glossary.md#term-flutter
[g-gateway]: 07-glossary.md#term-gateway
[g-github]: 07-glossary.md#term-github
[g-go]: 07-glossary.md#term-go
[g-harness]: 07-glossary.md#term-harness
[g-hook]: 07-glossary.md#term-hook
[g-hooks-events]: 07-glossary.md#term-hooks-events
[g-host]: 07-glossary.md#term-host
[g-https]: 07-glossary.md#term-https
[g-interrupted]: 07-glossary.md#term-interrupted
[g-main-roles]: 07-glossary.md#term-main-roles
[g-mcp]: 07-glossary.md#term-mcp
[g-message]: 07-glossary.md#term-message
[g-model]: 07-glossary.md#term-model
[g-model-endpoint]: 07-glossary.md#term-model-endpoint
[g-model-protocol]: 07-glossary.md#term-model-protocol
[g-mvp]: 07-glossary.md#term-mvp
[g-orchestration]: 07-glossary.md#term-orchestration
[g-outbox]: 07-glossary.md#term-outbox
[g-owner]: 07-glossary.md#term-owner
[g-package]: 07-glossary.md#term-package
[g-payload]: 07-glossary.md#term-payload
[g-policy]: 07-glossary.md#term-policy
[g-pr]: 07-glossary.md#term-pr
[g-priorities]: 07-glossary.md#term-priorities
[g-provider]: 07-glossary.md#term-provider
[g-push]: 07-glossary.md#term-push
[g-push-token]: 07-glossary.md#term-push-token
[g-quality-scenario]: 07-glossary.md#term-quality-scenario
[g-repository]: 07-glossary.md#term-repository
[g-requirement-ids]: 07-glossary.md#term-requirement-ids
[g-result]: 07-glossary.md#term-result
[g-retry]: 07-glossary.md#term-retry
[g-review]: 07-glossary.md#term-review
[g-role]: 07-glossary.md#term-role
[g-run]: 07-glossary.md#term-run
[g-runner]: 07-glossary.md#term-runner
[g-runtime]: 07-glossary.md#term-runtime
[g-sandbox]: 07-glossary.md#term-sandbox
[g-secret]: 07-glossary.md#term-secret
[g-secret-reference]: 07-glossary.md#term-secret-reference
[g-skill]: 07-glossary.md#term-skill
[g-smart]: 07-glossary.md#term-smart
[g-sse]: 07-glossary.md#term-sse
[g-step]: 07-glossary.md#term-step
[g-stride]: 07-glossary.md#term-stride
[g-subagent-presets]: 07-glossary.md#term-subagent-presets
[g-supabase]: 07-glossary.md#term-supabase
[g-task]: 07-glossary.md#term-task
[g-timeout]: 07-glossary.md#term-timeout
[g-tool]: 07-glossary.md#term-tool
[g-tunnel-providers]: 07-glossary.md#term-tunnel-providers
[g-ui]: 07-glossary.md#term-ui
[g-uml]: 07-glossary.md#term-uml
[g-usage]: 07-glossary.md#term-usage
[g-use-case]: 07-glossary.md#term-use-case
[g-work-ids]: 07-glossary.md#term-work-ids
[g-workspace]: 07-glossary.md#term-workspace
[id-doc-006]: 06-product-vision.md
[id-doc-007]: 07-glossary.md
[id-doc-012]: 12-use-case-diagrams-and-specifications.md
[id-doc-013]: 13-quality-attribute-scenarios-and-utility-tree.md
[id-doc-014]: 14-demo-acceptance-criteria.md
[id-doc-018]: 01-product-backlog.md#doc-018
[id-doc-024]: 01-product-backlog.md#doc-024
[id-doc-029]: 29-data-and-state-model.md
[id-doc-033]: 01-product-backlog.md#doc-033
[id-doc-034]: 01-product-backlog.md#doc-034
[id-doc-036]: 36-api-events-and-messages.md
[id-doc-048]: 01-product-backlog.md#doc-048
[id-doc-049]: 01-product-backlog.md#doc-049
[id-doc-059]: 01-product-backlog.md#doc-059
[id-doc-064]: 01-product-backlog.md#doc-064
[id-doc-070]: 01-product-backlog.md#doc-070
[g-actors]: 07-glossary.md#term-actors
[g-development-agent]: 07-glossary.md#term-development-agent
[g-risk]: 07-glossary.md#term-risk

[g-traceability]: 07-glossary.md#term-traceability

[id-doc-011]: 11-initial-risk-register.md

[id-doc-020]: 20-architecture-and-decisions.md

[id-doc-040]: 40-agent-execution-and-security.md

[id-doc-057]: 57-verification-and-traceability.md

[id-doc-067]: 01-product-backlog.md#doc-067

[id-rsk-01]: 11-initial-risk-register.md#rsk-01

[g-accessibility]: 07-glossary.md#term-accessibility

[g-device-session]: 07-glossary.md#term-device-session

[g-encryption]: 07-glossary.md#term-encryption

[g-fencing]: 07-glossary.md#term-fencing

[g-identifier-formats]: 07-glossary.md#term-identifier-formats

[g-jwt-jwks]: 07-glossary.md#term-jwt-jwks

[g-readiness]: 07-glossary.md#term-readiness

[g-recovery]: 07-glossary.md#term-recovery

[g-staging]: 07-glossary.md#term-staging

[g-terminal-run]: 07-glossary.md#term-terminal-run

[id-doc-038]: 38-agent-package-format.md

[id-doc-061]: 61-android-screens-flows-and-api.md

[id-act-01]: 08-actors-and-stakeholders.md#act-01

[id-doc-008]: 08-actors-and-stakeholders.md

[id-stk-01]: 08-actors-and-stakeholders.md#stk-01
