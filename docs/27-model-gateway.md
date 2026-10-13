# Потоковый модельный адаптер

12.10.2026. IP-01 из [плана](implementation-plan.md), часть BL-005/021/023/031; требования [SRS FR-012–016](03-software-requirements-specification.md#fr-012). `internal/modelgateway.Client` реализует интерфейс `execution.Gateway`. Это библиотека протокола, ещё не подключённый production service.

## Контракт

- Constructor получает доверенные provider ID, endpoint, model, API key и allowlist. Endpoint сравнивается точно; URL credentials/query/fragment запрещены. TLS verification штатная; private/local HTTP разрешается только явной записью в операторском allowlist. Влияние environment proxy отключено; redirect и автоматический retry POST запрещены. В error/log не возвращаются upstream body, URL или credential. Allowlist задаёт доверенный сервер; пакет не обещает защиту от подмены DNS у уже доверенного hostname.
- Запрос — `POST <base>/chat/completions`, `stream=true`, `include_usage=true`, `n=1`, `parallel_tool_calls=false`. Профиль использует legacy `max_tokens` для совместимых локальных серверов; это не обещание поддержки всех OpenAI reasoning models. Такой профиль должен пройти IP-02 для конкретных endpoint/model/config revision.
- Prompt строится из pinned role/config, skills, task, source revision и сохранённых observations. Provider key остаётся только в HTTP header. Task/skills/tool output не становятся server policy. Assistant tool-call + tool-result пары имеют детерминированные protocol IDs; ID из LLM не выдаёт server ToolCall/approval identity.
- Текущий набор — text final и **один shell proposal за шаг**. Неизвестный alias проходит серверный deny; другие объявленные активные виды инструментов пока дают `MODEL_TOOL_UNSUPPORTED`, а не скрыто пропускаются. Write/read/docs adapters должны появиться вместе с Runtime/MCP. Tool arguments собираются полностью и проходят строгий JSON parser: duplicate/unknown fields отклонены.
- SSE ограничен 1 MiB на ответ, 128 KiB на frame/line, 64 KiB накопленного текста и 8 KiB tool args. Finish и `[DONE]` обязательны. Несколько choices/tools, смена completion ID, content после finish, оборванный JSON/stream не принимаются. Preamble рядом с tool call не становится final answer. `length`/`content_filter` нормализуются как ошибка модели, с сохранением подтверждённого usage.
- `usage` принимается только с полными неотрицательными prompt/completion/total и согласованной суммой; превышение input/output reservation даёт unknown. Отсутствие usage — unknown, не ноль. Harness при unknown списывает весь резерв и прекращает run. Протокольные ошибки не создают approval/result.
- HTTP timeout ≤120 с, bounded headers и connection timeouts; Harness сужает deadline до run/chain и передаёт cancellation. Адаптер не retry-ит при обрыве/401/429/500.

## Учет до вызова

`TokenCounter` — обязательная для Harness доверенная функция подсчёта **всего сериализованного запроса с framing/tool schemas** под выбранные tokenizer/chat template. Constructor не подставляет оценку по длине строки. Без функции `InputTokens` возвращает `TOKEN_COUNT_UNAVAILABLE`; модель не вызывается. Один и тот же snapshot/body используется при count и dispatch. В тестах явный fixture counter не выдаётся за реальный tokenizer.

В IP-02 реализован жизненный цикл отчётов ниже. Токенизация/проверенная верхняя граница и два реальных model ID ещё не подтверждены. Production wiring должно сохранять service ownership: execution не читает control secrets через SQL; потребуется аутентифицированный внутренний gateway contract и повторная проверка актуальных configuration/device/epochs перед dispatch.

<a id="capability-check"></a>
## Проверка провайдера (IP-02, 13.10.2026)

`POST /v1/providers/{id}/capability-check` требует действующие owner/device, `Idempotency-Key`, текущий `If-Match` и строгое тело `{"profile":"chat-completions-v1"}`. Полный контракт — [OpenAPI](../api/openapi.json). Запрос явно инициирует максимум две генерации с фиксированным несекретным prompt, `max_tokens=128` каждая, `n=1`, общим model timeout 12 с и attempt deadline 15 с. Максимум — 256 запрошенных output tokens плюс входные токены двух небольших фиксированных prompt. Сервер модели может нарушить заявленные лимиты; протокольная проба не обещает доказанный input bound или денежный бюджет run.

Первый запрос принудительно предлагает `portway_probe` с точными тестовыми аргументами; команда **не исполняется**. Второй запрос получает synthetic tool observation и должен вернуть её маркер. Успех требует корректных SSE/finish/`[DONE]`, tool proposal/result и reported usage в обоих ответах. В отчёте сохраняются только флаги, число запросов и безопасный код ошибки; upstream body/credential/task/argv не сохраняются. Проверка `/models` и общая совместимость любых моделей этим профилем не заявляются.

Control API расшифровывает активный secret с installation/secret/key-version AAD; значение используется только в Authorization header выбранного allowlisted endpoint. Execution не получает control secrets через SQL. Проверяются текущие config version/check ID/device/deadline перед каждым HTTP-запросом. SQL-транзакция не держится во время сети. Проверка перед HTTP не заменяет future dispatch fencing; уже отправленный запрос может завершиться после изменения настроек/отзыва.

До model I/O одной транзакцией сохраняются `checking` report, audit `provider.check_started` и неизменяемый **202 acknowledgement** (`check_id`, provider/config/profile, `status_url=/v1/providers`). Первый запрос выполняет пробу синхронно в пределах deadline; параллельный replay может получить 202 ещё во время проверки. Replay того же ключа не повторяет I/O даже после ошибки/обрыва/падения процесса. Другой ключ во время живого attempt получает `CAPABILITY_CHECK_IN_PROGRESS`; после deadline новый ключ явно начинает новую пробу. История результатов отдельных checks не публикуется: GET возвращает только текущий report, который может уже принадлежать следующему check.

Audit actions `provider.check_started` и `provider.check_completed` включены в строгий контракт `portway.control.audit.v1`; outbox/relay/consumer передают только metadata, без prompt/secret/report body. `check_completed` означает завершение операции проверки; успешность протокола читается из report.

Ответ публикуется CAS по provider/config/check ID. Изменение провайдера очищает report и `checked_at`; поздний старый ответ не восстанавливает их. Отзыв устройства не позволяет опубликовать успех. Падение до settlement оставляет истекающий `checking`, не успех. `capabilities` содержит endpoint/model/config version, профиль, время начала/deadline/проверки/expiry и `protocol`. Отчёт хранится в существующем JSONB `providers.capabilities`; новая миграция не требуется.

`GET /v1/providers` добавляет вычисляемый `protocol_valid`: completed success, точное совпадение текущего binding/profile, allowlist и DB time до `expires_at` (30 мин после проверки). Unknown/checking/failed/expired дают false. `capabilities.runnable` всегда false: потребуются verified TokenCounter, private dispatch contract, Runtime и admission/fencing IP-04. AgentVersion пока сохраняет unresolved для production dependencies.

По ответу владельца от 13.10 локального модельного сервера пока нет; разработка продолжается без загрузки модели. Все текущие доказательства — HTTP fixtures и disposable PostgreSQL; реальные M1/M2 остаются непроверенными. [Протокол проверок](evidence/provider-capabilities-2026-10-13.txt).

## Cloud free-only и private service contract

13.10 добавлен [OpenRouter free-only/private Gateway](28-free-model-connections.md): актуальный zero-price каталог, env provisioning, специальный SSE accounting frame, provider version pin, counted receipt и encrypted step journal. Generic wire-profile выше остаётся отдельным. Реальные бесплатные пробы выполнены, full accepted roundtrip ещё отсутствует.

## Проверка и ограничения

HTTP fixture проверяет потоковый tool call → approval → fixture outcome → следующий HTTP model request → final Result в PostgreSQL. Unit/race cases проверяют fragments, missing usage, неверные totals, invalid/duplicate arguments, bounds, redirect, errors, отмену и совпадение counted/sent body. Реальный LLM, Runtime и деньги не используются. Streaming пока агрегируется внутри Gateway; `model.delta` в публичный SSE ещё не публикуется и входит в IP-04. [Протокол](evidence/model-gateway-2026-10-12.txt).

Источники протокола, проверены 12.10.2026: [Chat Completions create](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create), [streaming chunks](https://developers.openai.com/api/reference/resources/chat/subresources/completions/streaming-events). Они описывают wire-формат, а не доказывают совместимость локального endpoint Portway.
