# Навигатор контрактов клиента

Читайте строки, относящиеся к текущему изменению. Числа, поля DTO, ошибки и переходы извлекайте из актуальных документов; эта карта не подменяет спецификацию.

| Область | Источники | Что установить до правки |
|---|---|---|
| Установка и auth | DOC-061 UI-01…04/17, § 6; DOC-036 `#conventions`, `#auth`; DOC-040; SRS FR-081/082/084/090 | HTTPS URL и смена A→B; регистрация → подтверждение → обычный вход; безопасный callback; отзыв устройства и честный offline logout. Не переносить Comet OTP/social sign-in |
| Сеть и DTO | DOC-036 `#conventions`, OpenAPI и backend handlers/tests | Реальные заголовки, envelope, idempotency scope, исключения login/refresh, version conflict; что уже реализовано и доступно в конкретной установке |
| Лента SSE | DOC-036 `#sse`; DOC-024; DOC-061 UI-10/§ 5; DOC-029 | Event ID/seq/cursor, авторизация заголовками, dedupe, gap/resync, 400/409/410 до открытия потока, heartbeat, reconnect/revocation |
| Апрув | DOC-061 § 4/UI-11/12; DOC-025; DOC-029/040 | Server deadline/offset, revalidation после фона, блокировки POST/unknown, потеря ACK и повтор исходной операции, terminal states. Approved ≠ tool executed |
| Push | DOC-061 § 7; DOC-036 `#auth`; DOC-040; DOC-057 TC-012/032 | Объяснение перед permission, token rotation/unlink, безопасный payload, общий coordinator auth→GET latest→destination, foreground dedupe; normal terminated отличается от force-stop |
| Пакеты | DOC-038; DOC-036 `#packages`; DOC-061 UI-15/16 и § 6 | Системный picker/SAF, scoped URI и отмена, upload/staging/preview/mapping/activation, unresolved, write-only secret refs; лимиты в SRS § 4.2 |
| Результат/ревью | DOC-061 UI-13/14; DOC-026; DOC-036 `#runs`; SRS M-части FR-059/070 | Pinned result/revision, отдельный reviewer, pending/inconclusive/negative состояния, ручная новая задача и отсутствие auto-fix |
| Архитектура и приёмка | DOC-020 `#flutter-architecture`; DOC-052; DOC-057 | Текущие ADR, команды сборки, реальные критерии и evidence; клиентская библиотека не считается выбранной, пока решение не найдено или не принято |

`DOC-NNN` означает существующий `docs/NN-...md`; точное имя найдите через `rg --files docs`. Не создавайте угаданный файл из номера. Перечень UI/FR проверяйте по текущему DOC-061/SRS, особенно после изменения требований.

При проверке повторов отделяйте read/reconnect от mutation: автоматическое восстановление SSE не даёт права повторять опасный инструмент или хранить постоянную offline-очередь решений.
