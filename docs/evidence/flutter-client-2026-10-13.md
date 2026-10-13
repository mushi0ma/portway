# Первый Flutter-клиент Portway — 13.10.2026

Дата отчёта: 13.10.2026, Asia/Almaty. Рабочее дерево dirty; backend параллельно менялся в чате «Продолжить разработку Go-бэкенда». Проверенный клиент идентифицирован [SHA-256 manifest](flutter-client-2026-10-13/manifest.json). Backend-файлы этой работой не изменялись; документация дополнена точечно с сохранением его native Codex разделов.

## Реализованный срез

- Flutter Android `apps/portway`, RU ARB, Material 3, Dark/Light и системная тема. Figma прочитан через connector: файл BZ7UxNzlF3OmJRhd5JS19s, UI-04 `21:9469`, UI-05 Dark `21:8230`/Light `21:8319`. Токены зафиксированы в `DESIGN.md`. Оригинальный Figma-файл не изменялся.
- HTTPS origin/client-config, owner signup/login/refresh, Android callback без credential, безопасное хранилище, смена установки/поколения навигации, logout с revoke-only tombstone.
- Главная, read-only агенты, профиль, Codex prompt/history/snapshot/events/answer/stop по OpenAPI **0.7.0**, шесть публичных операций `/v1/codex/*`. Нет встроенного fixture режима в APK.
- Explicit retry одного pending body/key; ключ UUIDv4; host/device binding. Retry отключается через 6 суток, раньше серверного 7-дневного окна. Polling приостанавливается в фоне и начинает со snapshot после resume; events dedup и последовательность, до 256 событий, до трёх страниц за чтение. Stop не означает мгновенного физического завершения процесса.

## Среда и команды

Flutter **3.47.2**, Dart **3.13.2**, Linux x86_64, Android target SDK **36**. `pubspec.lock` сохранён. Flutter обновляет кеш за пределами sandbox, поэтому SDK/Gradle команды запускались с одобренным расширением доступа. Использован установленный SDK, без upgrade.

Рабочая папка: `/home/mushi/Workspace/Portway/apps/portway`.

| Проверка | Команда | Наблюдаемый результат |
|---|---|---|
| Формат | `dart format lib test` | Отформатировано; итоговый format gate отдельно проверен |
| Анализ | `flutter analyze` | No issues found |
| Контракты и UI | `flutter test` | **21 тест PASS** без обновления эталонов |
| Android | `flutter build apk --debug` | Debug APK собран; aapt: app.portway.portway, 0.1.0+1, min24/target36 |
| Источники | `git diff --check` | PASS |
| Устройства | `adb devices`, `emulator -list-avds` | Подключённых устройств/готовых AVD нет |

13 unit/HTTP fixture тестов: URL, запрет redirect credentials, concurrent single-flight refresh и mutation key, uncertain refresh, поздний ответ после logout, device revoked, потерянный POST/restore/retry, definitive busy, смена device, offline logout, revoke ACK, duplicate events, gap. Секреты тестовые, не действующие credentials.

8 widget-тестов: login при 200% в обеих темах; главная/навигация в обеих темах; два login golden; rail на ширине 900 при 200%; длинный ответ при 200%. Первый прогон обнаружил ошибку ожидания HTTP futures в fake async тестовой среде: `pumpAndSettle` не завершался. Исправлен test harness (`runAsync`), затем все сценарии прошли. Эталоны создавались впервые, время инъецировано фиксированным. Перед итоговым обычным `flutter test` они осмотрены.

## Визуальная проверка

Widget render, **не скриншоты Android**: [главная Dark](../../apps/portway/test/goldens/home-dark.png), [главная Light](../../apps/portway/test/goldens/home-light.png), [вход Dark](../../apps/portway/test/goldens/login-dark.png), [вход Light](../../apps/portway/test/goldens/login-light.png).

Проверены иерархия, спокойные поверхности, один основной CTA, читаемость, прокрутка и размеры кнопок. Системные status/navigation/keyboard рисует Android; нарисованные элементы Figma не копировались. Реальные insets, клавиатура, TalkBack и predictive Back требуют устройства.

Контраст (`skills/portway-figma-flutter-tokens/scripts/contrast.py`, threshold4.5):

| Пара foreground/background | Ratio | Итог |
|---|---:|---|
| Dark muted `#9ca8ac` / `#141a1c` | 7.210 | PASS |
| Light muted `#5f6d72` / `#eff2f2` | 4.762 | PASS |
| Dark CTA `#07282b` / `#35c6ce` | 7.511 | PASS |
| Light CTA `#f3f6f6` / `#006970` | 5.937 | PASS |

## Ограничения и следующая приёмка

Это первый срез, не готовый полный MVP. Реальное end-to-end телефон → HTTPS → auth → Codex ещё **NOT_RUN**. Backend real native roundtrip описан в [статусе параллельного среза](../implementation-status.md); его итоговый протокол оформляется backend-чатом и не подменяет запуск APK. Не создавались реальные owner/account и задачи от имени пользователя из клиента.

Поставляется debug APK с debug signing. Release signing, удалённый CI, физический Android16, реальный secure-storage/callback/IME/TalkBack пока не проверены. SSE обычных runs, approval/reviewer, FCM (3 background+3 terminated), SAF/ZIP, редактор агента/устройств остаются следующим объёмом. Полные TC-003/022/032 и другие acceptance TC не переводились в PASS.

Следующий путь: установить APK на телефон → подключить HTTPS origin → войти → убедиться, что мост на связи → отправить небольшой read-only запрос → проверить ответ/историю/stop → background/resume/offline → logout/revoke. Снимать native screenshots и вести evidence отдельно от widget goldens.

## Первичные технические источники

[Android16 SDK/target36](https://developer.android.com/about/versions/16/setup-sdk), [Flutter localization](https://docs.flutter.dev/ui/accessibility-and-internationalization/internationalization), [flutter_secure_storage](https://github.com/juliansteenbakker/flutter_secure_storage). Figma-привязки и семантические роли — `DESIGN.md`.
