# Execution: SSE и хранение истории

12.10.2026; D-22, BL-032/064/065 частично, DOC-036 §6 и DOC-029 §7. Срез дополняет [execution foundation](23-execution-foundation.md). `POST /v1/runs` остаётся закрытым до готовности Runtime, model capabilities и dispatch fencing. Тестовые runs создаются только в disposable PostgreSQL через trusted Go fixture.

## Поток событий

`GET /v1/runs/{id}/events` принимает одновременно `Authorization: Bearer …` и `X-Device-Session: …`. Токены в URL запрещены. Возобновление: один `Last-Event-ID: <run UUID>:<seq>` либо один `cursor` query с тем же форматом. Отсутствие курсора означает 0; конфликт, повтор параметра, другой run, отрицательное/неканоническое число →400; курсор впереди snapshot →409.

API получает страницы до 100 событий по приватному HTTP от execution; каждый запрос execution проверяет JWT/device через control authority. Установка и owner проверяются в execution, сервисный ключ не заменяет пользовательские credentials. DB-роль API не получает доступа к execution-схеме. Kafka продолжает доставлять межсервисные события; публичный replay читает сохранённую execution history, поэтому недоступность брокера не создаёт пробелы в SSE.

SSE `id` = run UUID:seq, `event` = сохранённый type, `data` = Event JSON. Доступны `run.created`, `run.state_changed`, а также `approval_requested`/`approval.resolved` из [бизнес-логики согласований](25-approval-workflow.md). События модельного ответа или исполнения команды не имитируются. В payload нет task, instructions, токенов или provider secrets. Страница проверяется на непрерывность seq; дубли/пропуски не маскируются. После replay чтение продолжается раз в секунду; полные страницы дочитываются сразу. Heartbeat — комментарий каждые 10 секунд без ID, `retry: 1000` — подсказка переподключения.

Перед HTTP 200 проверяются авторизация и курсор. После открытия потока ошибка зависимости, истечение JWT, отзыв устройства или истечение истории закрывает соединение. Клиент переподключается с последним обработанным ID и получает обычный HTTP 401/410/503. Такие сбои не создают искусственных бизнес-событий. GET run и stop run возвращают `snapshot_cursor`, согласованный с `snapshot_seq` и телом snapshot; содержимое chain сохраняет обычные Run DTO.

Поток живёт не более 30 минут, затем клиент переподключается. На процесс API допускаются четыре потока; пятый →429 с `Retry-After: 2`. HTTP-проверка каждой страницы ограничена двумя секундами, запись клиенту — двумя секундами. Блокировка на медленном клиенте не держит DB-соединение. Для SSE обычные 20/25-секундные request/write timeouts заменены отдельными ограничениями; shutdown API отменяет открытые потоки. Бюджет соединений Supabase остаётся прежним: API 2 + relay 1 + audit 1 + execution 2. Цель закрытия после revoke ≤5 секунд проверяется в интеграции; это не измерение остановки внешних процессов.

## Очистка без потери replay

Миграция 006 добавляет `runs.events_pruned_through`, expiry index и локальный `maintenance_log`. HistoryPage читает ownership, aggregate high-water, watermark и события в одной read-only repeatable-read транзакции. Параллельная очистка возвращает клиенту либо согласованную старую страницу, либо 410, без пустого пропуска.

Execution выполняет cleanup при запуске и раз в минуту с таймаутом 5 секунд. За транзакцию удаляются максимум 200 событий одной завершённой chain, 200 истёкших request keys и 200 maintenance records старше 30 суток. Неизменяемые snapshots/результаты не удаляются.

Удаляется только непрерывный префикс конкретного run. Событие должно быть старше семи суток, а outbox — иметь ACK не младше семи суток. Pending/свежий ACK/свежее событие защищает весь следующий суффикс; active/stopping chain не очищается. Блокировка установки согласует cleanup с командами. Deletion outbox/history, watermark и запись итогов в `maintenance_log` атомарны. Ошибка аудита откатывает очистку. Пул и DB-права не расширяются на чужие схемы; execution получает DELETE только своих events/outbox и обслуживание собственного журнала.

Курсор ниже watermark →410 `HISTORY_EXPIRED` до открытия SSE. `error.details` содержит `snapshot_url`, текущий `snapshot_cursor` и `events_pruned_through`. Клиент читает snapshot по URL, показывает «Часть истории недоступна» и возобновляет поток с курсором именно полученного snapshot. Если cleanup обгоняет открытый медленный поток, соединение закрывается; следующий запрос также получает 410.

## Границы проверки

Свидетельства выполнения — [протокол](evidence/execution-sse-2026-10-12.txt). В Docker проверяются реальная PostgreSQL, Kafka, отдельные DB-роли, control→execution→authority HTTP и Auth fixture. Проверки публичного auth guard/health через Funnel не равны проверке авторизованного SSE с телефона. Настоящий owner signup, авторизованный SSE через Funnel, p95 ≤2 секунд и Android пока не заявлены. Новых ключей или настроек Supabase Dashboard для этого среза нет.

Следующий этап: Runtime Manager/scoped grants, настоящие model capability probes и Harness; затем подключение согласований к реальному dispatch, durable Results и Kafka consumers execution-событий.
