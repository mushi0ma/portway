# Согласование действий владельцем

12.10.2026; BL-050/051/052/054 частично, DOC-036 §5, DOC-040 §3–4 и DOC-029 §6. Реализована бизнес-логика решения владельца в execution service. Самостоятельного публичного вызова shell нет. До появления Harness/Runtime `POST /v1/runs` остаётся закрытым; в production новые согласования пока некому создавать.

## Сценарий

1. Trusted Harness вызывает `Chain.RequestShellApproval` внутри `Store.Change`. Проверяются активный run, deadline, объявленный `shell.exec`, pinned policy=approval и форма команды. Команда — argv, нормализованный относительный working_dir, timeout ≤60 секунд; не произвольные Docker parameters. Параметры закрепляются вместе с SHA-256, policy hash, run/chain epoch, tool_call_id и server expiry через 180 секунд. Harness обязан не включать credential values в предложение. Полные параметры доступны только владельцу, в события идут ссылки/hash.
2. Run переходит в `awaiting_approval`; команда ещё не выполняется. `GET /v1/approvals` возвращает pending и server time. `GET /v1/approvals/{id}` — параметры, hash, version, expiry и исход связанного tool.
3. Владелец отправляет `POST /v1/approvals/{id}/decision` с `Idempotency-Key` и `{decision: approve|reject, expected_version, args_hash}`. Изменённые hash/version →409. Противоположное позднее решение →409. Повтор того же решения возвращает текущее состояние, включая возможную последующую отмену tool.
4. Approve → `approved`, tool=`ready`; модельный цикл остаётся заблокирован до результата команды. Здесь нет Runtime grant, dispatch_id или исполнения. Reject → `rejected`, tool=`denied`, run снова может предложить безопасную альтернативу. Тот же отклонённый/истёкший hash повторно не принимается; другое действие заново требует решения.
5. При `now >= expires_at` решение отклоняется с 422 `APPROVAL_EXPIRED`, а expired сохраняется транзакционно. GET/list также сверяют expiry; фоновая сверка работает в execution loop. Недоступность Kafka может задержать фоновое отображение, но не продлевает допустимый срок решения. Error details включают текущий approval для обновления экрана.
6. Stop/restart делает pending cancelled. Уже записанный approved сохраняется, но его ещё не выполненный tool становится cancelled. Повтор старого approve не возвращает устаревший ready и не оживляет работу. Deadline, epoch, pinned policy и token budget повторно проверяются перед фиксацией approve.

Новая роль не нужна: migration 007 создаёт индекс согласований в закрытой `portway_execution`. Содержимое живёт в JSON aggregate под chain lock. Уникальный частичный индекс допускает один pending на run; локальная транзакция включает состояние, индекс, idempotency, history и outbox. Одновременные approve/reject с двух устройств выбирают ровно один исход. Нельзя читать согласования чужого owner/installation. Ключ execution service без JWT/device ничего не разрешает.

`approval_requested` и `approval.resolved` публикуются через существующие Kafka outbox и SSE, с per-run seq. В payload нет argv, task или secrets. Согласование, outcome команды и grant — разные сущности; approved не выдаётся за успешный запуск.

## Что ещё требуется для полного сценария

Нужны реальный Harness, текущие ужесточения policy, Runtime Manager, grant/dispatch fencing с повторной проверкой актуального device и отменой внешней работы, модельный цикл безопасных альтернатив с лимитом отказов, FCM и Android-экран. Проверка authority перед HTTP-командой сама по себе не даёт атомарного cross-service отзыва; поэтому этот срез не выдаёт полномочий исполнителю. Число сохранённых предложений ограничено 20 на run, но это не заменяет Harness-лимит последовательных альтернатив.

Проверки — [протокол](evidence/execution-sse-2026-10-12.txt): domain tests с точной границей 180 секунд, реальные PostgreSQL-транзакции и private HTTP, гонка решений, stop/replay, сохранение expired после 422 и безопасные event envelopes. Все runs/proposals проверки создаются в disposable PostgreSQL; production owner, email и реальные команды тест не создаёт.
