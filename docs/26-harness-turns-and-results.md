# Harness: модельные шаги и результаты

12.10.2026. Следующий бизнес-срез после [согласований](25-approval-workflow.md), по [DOC-040 §2–5](40-agent-execution-and-security.md#harness) и [DOC-036](36-api-events-and-messages.md#runs). BL-030/032/033/050/054 и результат FR-059/070 реализованы частично.

## Реализованный сценарий

`internal/execution.Harness.Advance` выполняет один модельный шаг, используя trusted `Gateway`. Перед отправкой Gateway считает или ограничивает сверху **полный** запрос с instructions, task, tools и предыдущими наблюдениями; подсчёт сам не вызывает модель. Подсчёт и отправка получают один snapshot. Под chain lock сверяются version, deadline, состояние и оба бюджета; reservation, turn, history/outbox фиксируются до внешнего вызова. Конкурирующий worker не получает второе разрешение. Gateway обязан выполнять не более одного запроса, соблюдать context и не делать скрытых retry.

Полный нормализованный ответ содержит `final`, один `tool` proposal или `error`, а usage поступает от trusted adapter отдельно от текста модели. Partial stream не принимается за ответ. Settlement, переход и Result/approval сохраняются одной транзакцией. Повтор идентичного ответа не списывает токены и не публикует событие заново; конфликтующий ответ отклоняется. Потерянный ответ, неизвестное usage или usage выше резерва списывают весь резерв и дают `failed/budget_uncertain`. Ошибка/невалидный ответ с известным usage сохраняет фактическое списание и безопасную причину ошибки.

В этом срезе proposal поддерживает только объявленный alias `shell.exec` с валидным argv/working_dir/timeout. Alias разрешается по pinned конфигурации, policy проверяется сервером. Модель не передаёт permissions или owner decision. Разрешённое предложение создаёт существующий approval TTL=180 секунд; Harness возвращает паузу. Даже approved не разрешает следующую модельную итерацию без результата инструмента.

Reject/expiry попадает в следующий запрос как `Observation` с code, tool_call_id, denied_args_hash и подсказками. Повторное чтение решения не увеличивает счётчик. Два последовательных отказа → `failed/policy_exhausted`. Повтор той же команды под другим shell alias также отклоняется. Успешное безопасное действие сбрасывает счётчик; модель может вместо новой команды дать итоговое объяснение.

## Граница Runtime

`BeginToolDispatch` закрепляет один dispatch ID и deadline команды, проверяя approval/expiry, epochs, pinned policy и бюджет. **Это запись намерения, не Runtime grant.** Методы `BeginToolDispatch`/`CompleteTool` доступны только Go-адаптеру, публичного HTTP execute/outcome нет. Тесты вызывают их как fixture; команд на хосте они не исполняют. Будущий адаптер обязан проверить актуальные device и ужесточения policy, получить scoped grant и обеспечить fencing в Manager. Эта предварительная запись не доказывает атомарный отзыв между сервисами.

`CompleteTool` принимает только исход закреплённого dispatch: completed/failed/unknown/timeout/cancelled и bounded observation ≤16 KiB, очищенный адаптером от credentials. Дубли идемпотентны, конфликтующие результаты запрещены. Completed/failed сохраняют наблюдение до следующего шага; unknown/timeout/cancelled завершают run без повторного исполнения. Прошедший deadline не превращается в успех. Stop не допускает новые шаги; поздние результаты сохраняются без оживления run. `ConfirmStopped` отказывает, пока есть незавершённый dispatch.

При restart незавершённый dispatch помечается unknown, turn — interrupted, run — failed/interrupted. Автоматического resume нет. Это учёт исхода, а не доказательство физической остановки внешнего процесса: production Runtime и его reconciliation ещё необходимы.

## Результат и API

Final answer сохраняет `Result` в execution aggregate: ID, run/step, summary ≤64 KiB, pinned source revision/config hash, SHA-256 текста и server time. `GET /v1/runs/{id}/result` требует owner/device и возвращает 404 `RESULT_NOT_FOUND`, пока результата нет; чужой run →404. Result также виден в полном snapshot. Повтор/stop/reconciliation не заменяет терминальный результат. Миграция не требуется: новые поля additive в существующем JSONB aggregate.

`run.completed` содержит только ссылки/hash; `tool_result` — tool/step ID и безопасный outcome. Они проходят существующие Kafka outbox и SSE. Model answer, argv, task и output не публикуются в событиях. Summary — ответ модели, **не доказательство изменения repository**: result revision, diff, артефакты и reviewer verdict пока не материализуются. Поэтому completion не подключён к reviewer hook.

## Фактическая готовность

Harness ещё не подключён к production worker. `POST /v1/runs` по-прежнему возвращает `503 EXECUTION_NOT_READY`, `operations.run=false`. Нет встроенного fixture endpoint и включающего его environment flag. Остались реальные Gateway/capability probes/token accounting для двух моделей, streaming, Runtime/grants/authority fencing, workspace finalizer, reviewer/links/hooks, MCP и FCM. Поддержка workspace.read/write/docs.lookup в цикле ещё впереди.

Проверки: domain cases для usage, malformed/partial replies, replay, deadline, aliases, отказов, approval fencing, timeout, stop и restart; disposable PostgreSQL для reservation-before-dispatch, цикла proposal→approve→fixture outcome→final, конкурентных workers, отмены context после stop, rollback вместе с event insert failure и durable result; HTTP через публичный control proxy и приватный execution, owner/device guards. Fixture-прогон не заменяет проверку реальной модели или Runtime. Результаты запуска — [протокол](evidence/harness-2026-10-12.txt).

Продолжение: [IP-01 — потоковый Gateway](27-model-gateway.md) реализует HTTP/SSE адаптер; выбранные реальные модели, capability report и tokenizer относятся к IP-02. Production wiring всё ещё закрыт.
