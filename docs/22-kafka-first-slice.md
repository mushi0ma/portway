# Первый рабочий срез Kafka

11.10.2026. Реализуется D-22. Это доставка фактов аудита; execution, FCM и публичный API общей проекции ещё не реализованы.

## Процессы и права

| Контейнер | Данные и доступ |
|---|---|
| `api` | Control API, роль `portway_app`, Supabase Auth, локальный master key. Изменение и аудит атомарны; DB trigger создаёт outbox |
| `outbox-relay` | Бинарь `events relay`, роль `portway_relay`: читает outbox, меняет только `published_at`, вызывает ограниченную очистку. Нет Auth/master/provider secrets |
| `audit` | Бинарь `events audit`, роль `portway_audit`: только `portway_audit.inbox` и `entries`, без доступа к control-таблицам |
| `kafka` | Apache Kafka 4.3.1, digest закреплён; один combined KRaft broker/controller |
| `kafka-init` | Создаёт topic `portway.control.audit.v1`, 3 partitions, replication factor 1 |
| `migrate` | Отдельный одноразовый оператор схем, миграции 001–004; DSN не передаётся API/worker |

Go остаётся в одном репозитории. У worker отдельная точка сборки `cmd/events` и Docker target `events-runtime`. Миграции пока оркестрируются одним operator binary; перенос их владения в отдельные сервисные каталоги — следующий этап разделения.

## Контракт и сбои

Envelope v1: `schema_version`, `event_id`, `installation_id`, `producer`, `type`, `aggregate_id`, `correlation_id`, `occurred_at`, `payload`. Тип пока один: `audit.recorded`. Payload содержит только UUID участников/объектов и известные action/object_type/outcome; имён, email, инструкций, stdout, токенов и credentials нет. Correlation ID первого среза равен event ID; причинная связь run/request будет отдельным расширением контракта.

Одна installation на выделенный локальный broker; сообщение другой installation отклоняется. Key — installation/aggregate. Порядок между partitions не обещается, проекция append-only и от порядка не зависит. Это не доставка команд исполнения.

Relay сериализуется по installation транзакционным advisory lock, читает первое pending-событие, публикует с `acks=all`, затем отмечает публикацию и коммитит DB. Сетевое ожидание ограничено; потеря ACK/DB commit может дать дубль с прежним event ID. Producer idempotence включён библиотекой, но не заменяет consumer dedupe. Небольшие audit-сообщения отправляются без compression: Snappy требует native library в исполняемом временном каталоге, тогда как этот профиль Kafka использует noexec `/tmp`.

Consumer отключает auto-commit и обрабатывает одну запись: receipt с hash канонического события и строка проекции коммитятся вместе; Kafka offset — после DB commit. Повтор с тем же ID и содержимым не создаёт второй эффект; тот же ID с другим содержимым блокирует обработку. После ошибки обработки consumer завершается и перезапускается от committed offsets; продолжать poll после ошибки нельзя. Пустой poll timeout отделён от timeout во время DB/offset операции.

Неизвестный тип, нарушенный контракт и событие старше 24 часов блокируют partition/consumer и требуют operator reconciliation. Автоматический DLQ/replay CLI ещё не реализован. События не удаляются при ошибках. Обычные конфигурационные операции закрываются при 10 000 pending-событиях или pending старше 24 часов; отзыв устройства разрешён, чтобы очередь не блокировала отзыв доступа. В этой ситуации сначала восстанавливается доставка/разбирается просроченный backlog.

Публичный `/v1/audit` пока читает исходный authoritative audit, без cross-service SQL. Проекция worker подготовлена для будущего query API. `/v1/status` показывает backlog и допуск конфигурационных операций; worker `/readyz` проверяет DB, Kafka и состояние обработки. Полная API readiness остаётся 503 до реализации execution.

## Локальное развёртывание

После настройки migration/app DSN выполнить `python3 scripts/provision-db.py`: проверяет TLS, создаёт отсутствующие роли, генерирует отдельные worker DSN и сохраняет их в ignored `.env`. Существующие пароли не меняет. Затем `docker compose up --build -d`. Скрипту оператора нужны Python, psql и Docker Compose; рабочие контейнеры этих инструментов не требуют.

`compose.kafka.yaml` включается в основной и тестовый Compose. Kafka имеет только внутреннюю сеть `eventbus`, порт брокера не опубликован на хост. API и run sandboxes к этой сети не подключаются. Только relay/audit подключены одновременно к eventbus и сети для исходящих TLS-запросов Supabase. SASL/TLS внутри локальной Docker-сети этого профиля ещё нет; перенос Kafka между хостами требует отдельной конфигурации аутентификации и ACL.

Лимиты: Kafka heap 256 MiB, контейнер 1 GiB/1 CPU; каждый Go-процесс до 256 MiB. Пулы: API 2, relay 1, audit 1, кратковременно migrator 1. Kafka не занимает PostgreSQL connections. Один брокер не даёт HA или защиты от потери его диска.

Topic retention: 7 суток **или** 64 MiB на partition, смотря что наступит раньше; segment 8 MiB. Это политика очистки, не жёсткая квота Docker volume и не гарантия хранения 7 дней при превышении объёма. Offset lag и свободное место нужно контролировать до расширения нагрузки. Published outbox и projection/receipts очищаются порциями по 1000 после 7 суток; pending не удаляется. Core authoritative audit пока сохраняется без фоновой очистки.

`docker compose down` сохраняет Kafka volume. Не использовать `down --volumes` для рабочего стека без намерения удалить историю брокера. Тестовый runner создаёт уникальный project и удаляет только собственные disposable volumes.

## Проверки

`make docker-test` использует настоящий Kafka и disposable PostgreSQL: rollback исходной транзакции; недоступность publish; сбой после фактического broker ACK до DB commit; повтор события; конфликтующий дубль; восстановление consumer по offsets; закрытые схемы и невозможность worker читать secrets; закрытие configuration admission при просроченной очереди и сохранение возможности revoke.

Auth/email end-to-end, длительный реальный broker outage, broker disk loss/restore, DLQ, нагрузка и execution/stop/approval ещё требуют отдельных проверок. Локальные тесты не подтверждают готовность полного MVP.

Источники: [официальный Docker Kafka](https://kafka.apache.org/43/getting-started/docker/), [franz-go offset management](https://github.com/twmb/franz-go/blob/master/docs/producing-and-consuming.md).
