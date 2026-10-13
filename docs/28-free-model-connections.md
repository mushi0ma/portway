# Бесплатные cloud-модели и внутренний Gateway

Актуально: 13.10.2026, Asia/Almaty. Продолжение IP-02 по запросу владельца: начать подключение cloud-моделей и подготовку запусков, использовать пока только бесплатные модели. Исходные требования к Runtime, локальной модели, двум моделям и переносу не считаются принятыми этим срезом. [План](implementation-plan.md), [протокол](evidence/free-model-gateway-2026-10-13.txt).

## Настройка

В ignored `.env` добавлены только отсутствовавшие placeholders; прежние значения сохранены, права 0600:

```dotenv
OPENROUTER_API_KEY=
PORTWAY_OPENROUTER_MODEL=
PORTWAY_MODEL_POLICY=free_only
```

`OPENROUTER_API_KEY` уже заполнен владельцем; значение не выводилось. `_FILE` поддерживается конфигурацией API для отдельно управляемого secret-файла. Compose передаёт ключ только `api`, исключая migrate/execution/relay/audit. API добавляет фиксированный `https://openrouter.ai/api/v1` в allowlist при наличии ключа. Изменение env применяется после пересоздания API; оно не вращает автоматически ключи уже сохранённых providers. GROQ_API_KEY не требуется для текущего профиля.

`PORTWAY_OPENROUTER_MODEL` — необязательный явный model ID для CLI-диагностики. Пока ни один полный реальный tool/result probe не принят, рекомендуемый ID не активируется автоматически.

```sh
# Metadata-only, без ключа/генерации:
docker compose exec -T api /portway models
# Явно инициирует максимум две ограниченные бесплатные генерации:
docker compose exec -T api /portway model-probe 'vendor/model:free'
```

Эти команды доступны после сборки/обновления API-образа; текущий production этим срезом не обновлялся. CLI probe диагностический, не создаёт Run и не утверждает готовность Runtime.

## Политика стоимости

- Production config принимает только `free_only`. Hosted endpoints, кроме фиксированного OpenRouter, отклоняются; Groq Free зависит от тарифа аккаунта и пока не включён в этот проверяемый профиль. Тестовые self-hosted fixtures задаются напрямую trusted test config.
- Разрешены только явные `author/model:free`, исключая `openrouter/free`, auto/router, платные ID и modifiers вроде `:online`.
- Перед **каждой** генерацией выполняется bounded metadata GET `/models` без credential. Prompt/completion цены должны присутствовать; все опубликованные компоненты pricing должны быть точным неотрицательным нулём. Неизвестная/null/ненулевая цена или ошибка каталога закрывает вызов. Decimal zero проверяется без округления/разворачивания больших exponent. Дубликаты полей/ID и oversized/deep JSON отклоняются.
- Каждый запрос закрепляет `provider.max_price`=0 для prompt/completion/request/image/audio, `require_parameters=true`, `allow_fallbacks=false`. Нет fallback models, plugins, BYOK, web search или media generation. Optional OpenAI поля `n`, `parallel_tool_calls`, `stream_options` не отправляются; defaults не разрешают несколько choices/tools в нашем parser. Follow-up с запретом tools не объявляет tools и не отправляет `tool_choice:none`.
- OpenRouter автоматически включает usage. Специальный profile принимает accounting frame с пустым delta и повторённым finish_reason либо usage на финальном choice; новый content/tool, изменённый finish и двойной usage запрещены. Cost обязателен и должен быть ровно 0; unknown/nonzero cost не становится успешным ответом. Generic Chat Completions parser сохраняет прежний строгий профиль.
- HTTP 402/429/404, unsupported parameters и account data policy нормализуются в безопасные коды. Upstream body/key не возвращаются и не логируются; автоматических retry нет. Нулевые фильтры не обещают наличие свободной квоты или доступного endpoint.

## API подключения

Все публичные маршруты требуют действующие owner/device:

| Маршрут | Поведение |
|---|---|
| `GET /v1/models` | Текущий отфильтрованный metadata-каталог, supported_parameters, context_length, checked_at; без генерации. Metadata не доказывает реальное tool calling |
| `POST /v1/providers/openrouter` | `{model_id}` + Idempotency-Key. Берёт ключ из окружения control API, сохраняет новый AES-GCM secret/provider/audit атомарно. Ответ/дедуп не содержат ключ. Повтор не создаёт ещё одну запись |
| `POST /v1/providers/{id}/capability-check` | Ограниченный protocol probe из [DOC Gateway](27-model-gateway.md#capability-check), теперь с free-only guard для production |

Полный DTO/ошибки — [OpenAPI 0.6](../api/openapi.json). Изменение provider инвалидирует report. На реальных endpoints пока встречались недоступность/rate limits и нестабильное tool поведение; успешный полный capability report не выдуман.

## Private Gateway и запуск шага

`modelgateway.RPC` реализует `execution.Gateway`; production `cmd/execution` ещё не запускает worker через него. Фиксированный control URL, service key **и** live owner/device JWT+proof обязательны. Credentials остаются в памяти вызова; provider key/control DSN не попадают в execution. Контракт:

1. `POST /internal/v1/model-gateway/count` принимает только run ID. Control читает текущий Run через authenticated execution HTTP и собственные provider/agent строки. Проверяет текущую AgentVersion, pinned provider version, capabilities/TTL, deadline, run state. Подсчитывает **полный** request доверенным TokenCounter и возвращает receipt: run/epoch/provider version/input count/request hash. Сеть модели не вызывается.
2. Harness атомарно резервирует Step в execution до внешней работы.
3. `POST /internal/v1/model-gateway/complete` получает receipt + step ID. Control заново читает Run/turn/reservation, восстанавливает тот же prompt без нового waiting turn и проверяет count/hash/epoch/config. Строка dispatch фиксируется до HTTP-вызова. Повтор готового шага возвращает зашифрованный cached reply; повтор неопределённой попытки запрещён и не генерирует заново.
4. Перед dispatch, во время I/O и перед принятием ответа проверяется актуальность. Stop/revoke/config/expired report или ошибка authority отменяют HTTP. Чужая service key, stale count/provider binding и изменённый hash не дают вызова модели. Это ещё не полная распределённая fencing/physical Runtime SLA; провайдер может продолжить вычисление после обрыва.
5. Ответ нормализуется; потерянный usage/ошибка списываются Harness по прежней conservative settlement policy. Completion journal хранит ответ зашифрованным с step AAD; execution SQL-role читать его не может.

Миграция `008_model_dispatch.sql`: собственный control journal, 7 суток retention, cleanup при новом claim, предел 10000 записей, started/finished, hash, encrypted reply/nonce; нет cross-schema FK/SQL. Исторические snapshots могут не иметь provider_version; private gateway такие snapshots не допускает.

**TokenCounter отсутствует в production по умолчанию.** Подстановка fixture counter запрещена; вызов заканчивается `TOKEN_COUNT_UNAVAILABLE` до generation. Реальный pre-dispatch подсчёт/доказанная верхняя граница полного prompt ещё требуется. Public `POST /v1/runs` остаётся закрытым; настоящий sandbox/Runtime, workspace и worker/admission — следующий этап. Локальная интеграция проверяет отдельные control/execution HTTP, Harness → model fixture → encrypted replay/result, stop в полёте и отсутствие SQL-доступа к secrets/journal. Она не заменяет реального агента в контейнере.

## Источники, проверены 13.10.2026

[OpenRouter provider routing](https://openrouter.ai/docs/guides/routing/provider-selection), [streaming/accounting frame](https://openrouter.ai/docs/api_reference/streaming), [usage/cost](https://openrouter.ai/docs/cookbook/administration/usage-accounting), [free limits](https://openrouter.ai/docs/api_reference/limits), [Groq Free/Developer billing](https://console.groq.com/docs/billing-faqs). Конкретная доступность фиксируется в evidence, а не выводится из документации.
