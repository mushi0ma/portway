# Execution: транзакционная основа

11.10.2026, D-22; BL-030/032/033/064/065 частично. Этот срез добавляет отдельный процесс `execution`, владение состоянием и внутренний протокол. LLM/Runtime dispatch пока отсутствует: **POST /v1/runs возвращает 503 EXECUTION_NOT_READY**, запуск из HTTP не создаёт ни run, ни очередь работы. Fixture admission существует только как Go-метод для тестов будущего trusted adapter; production endpoint его не вызывает.

## Граница сервиса

`cmd/execution`, Docker target `execution-runtime`, Compose profile `execution`. Scratch, non-root, read-only, без Docker socket и опубликованных портов. `portway_execution` — отдельная DB-role и закрытая схема, миграция 005; control-api и workers не читают её таблицы. Execution не читает control identity, конфигурации или secrets через SQL. FK существуют только внутри собственной схемы.

Control API проксирует `/v1/runs`, `/v1/runs/{id}`, `/v1/chains/{id}` и stop на приватный `http://execution:8082/internal/v1/...`. Внутренний запрос требует отдельный `PORTWAY_EXECUTION_SERVICE_KEY` и исходные JWT/device proof. Execution на **каждом** запросе вызывает `/internal/v1/execution/authorize` control-api с этими credentials; сервисный ключ без действующего owner/device не даёт доступа. Недоступный control-api закрывает авторизацию, Kafka не используется как единственный источник отзыва. Токены не сохраняются в execution DB/outbox.

Внутренний HTTP допустим только на одном доверенном Compose-хосте. Для нескольких хостов требуются mTLS/ACL. Проверка прав перед stop безопасно уменьшает полномочия. Для будущего допуска платной/опасной работы одной предварительной проверки недостаточно: нужно завершить протокол текущего device/epoch admission непосредственно перед dispatch и выдачей Runtime grant. Поэтому creation admission сейчас закрыт.

## Транзакции и лимиты

Коллекция `/v1/runs` возвращает краткие состояния/бюджеты без полных tasks и инструкций; подробный pinned snapshot читается по ID.

Chain aggregate хранится в JSONB с отдельным индексом runs. Бюджеты, immutable snapshot, stop epochs и номера событий изменяются под одним chain row lock; запись состояния, history и outbox атомарна. Это собственная локальная транзакция execution, без cross-service JOIN. Installation advisory lock сериализует начальный небольшой поток команд и idempotency; масштабирование блокировок потребует измерений.

- Run: максимум 900 секунд, 20 шагов, 20 000 токенов, output 2048; запрос может только сузить immutable agent limits. Chain: 1800 секунд, 40 шагов, 40 000 токенов. Очередь входит в deadline.
- Одна chain пока содержит один корневой run. Admission fixture ограничен двумя активными chains; дочерние runs/hooks/hops/depth ещё не добавлены.
- До model dispatch предполагается commit reservation `input + output cap` сразу в run и chain. Повтор step ID не разрешает повторный dispatch. Settlement с подтверждённым actual usage освобождает разницу ровно один раз; конфликтующий settlement отклоняется.
- Unknown usage полностью списывает reservation и завершает run с `budget_uncertain`. Превышение reservation возвращает BUDGET_UNCERTAIN, удерживая резерв и блокируя следующий шаг до сверки. Этот срез не делает модельных вызовов и не заявляет фактические model usage.
- Stop записывает `stopping` и увеличивает epoch, запрещает новые reservations. `cancelled` возможен только после отдельного подтверждения и settlement всех reservations; HTTP stop возвращает 202, повтор terminal — 200 с текущим состоянием. Прежний `stopping` не возвращается из idempotency cache после терминального исхода.
- Startup reconciliation под singleton session lock помечает нетерминальные runs `failed/interrupted`, списывает неизвестные reserves консервативно и не переисполняет работу. Terminal state не меняется. Потеря leadership connection завершает процесс.

Пул execution — 2 соединения, одно удерживает singleton session lock. Нужен direct PostgreSQL или Supabase **Session pooler**, не transaction pooling. Общий верхний бюджет приложений: API 2 + relay 1 + audit 1 + execution 2 = 6, мигратор кратковременно ещё 1. Нет обещания производительности Free-плана.

## События и Kafka

Собственная history хранит per-run `seq` и safe lifecycle payload. Snapshot и его seq находятся в одном JSON aggregate; history query ограничивается этим high-water. Внутреннее чтение `events?after=N` возвращает до 100 записей, cursor ahead →409. Публичный SSE, watermark/410 и retention cleanup этого сервиса ещё не реализованы; `snapshot_seq` пока не является готовым SSE-контрактом.

Relay внутри процесса execution публикует `portway.execution.events.v1` с `acks=all`, затем отмечает outbox published. Key = installation/chain; crash после ACK до DB commit даёт прежний event ID повторно. Topic имеет 3 partitions и такой же dev retention, как control audit. В payload нет task, agent instructions, provider credentials или JWT. Admission fixture блокируется при 10 000 pending либо backlog старше 24 часов; stop разрешён и при недоступном брокере. Relay не удаляет просроченные pending события и блокируется для operator reconciliation.

Отдельный consumer execution events для audit-query/notifications ещё впереди; наличие topic не означает готовые downstream effects. Pending и опубликованные execution history/outbox пока не очищаются: retention и bounded cleanup должны быть добавлены до открытия production admission.

## Развёртывание

```sh
python3 scripts/provision-db.py
# После генерации отдельного service key установить в .env:
# PORTWAY_EXECUTION_URL=http://execution:8082
docker compose --profile execution up --build -d
docker compose exec execution /execution probe
make check-live
```

Provisioner создаёт только отсутствующие роли, сохраняет новый DSN и случайный service key в ignored `.env`. Существующие пароли не меняет. Profile добавлен явно, поэтому обычный запуск foundation не требует пустого execution DSN. В настроенной установке сохранён `COMPOSE_PROFILES=execution,funnel`, чтобы обычный `docker compose up -d` поднимал все подключённые сервисы.

`execution /readyz` подтверждает доступность процесса, DB и Kafka для чтения/stop/relay; это **не** допуск model/tool execution. Общий API `/readyz` остаётся 503; `/v1/status.operations.run=false`.

## Оставшиеся обязательные части

Следующий срез — Runtime Manager/scoped grants, настоящие capability checks и Harness dispatch; затем approvals TTL/CAS, отмена внешней работы и fencing, SSE с проверкой device ≤5 секунд, durable results, дочерние runs/links/hooks, notifications/FCM и audit-query. Базовая state machine не заменяет эти механизмы и не подтверждает NFR-009 (фактическая остановка процессов ≤5 секунд).
