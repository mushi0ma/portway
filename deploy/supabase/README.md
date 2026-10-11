# Настройка email callback

1. В Supabase → Authentication → URL Configuration установить Site URL равным `PORTWAY_PUBLIC_URL` из `.env`.
2. В Redirect URLs добавить точный `PORTWAY_PUBLIC_URL/auth/confirm`.
3. В Authentication → Email → Templates → Confirm signup вставить `confirm-signup.html` из этого каталога и сохранить. Go signup передаёт свой callback через `redirect_to`; шаблон использует `{{ .RedirectTo }}` и `{{ .TokenHash }}`.
4. Оставить email confirmation включённым. Административные настройки Supabase нельзя менять через publishable key.

Текущая установка использует Tailscale Funnel и постоянное имя узла:

```text
Site URL: https://portway.tail1443b8.ts.net
Redirect URL: https://portway.tail1443b8.ts.net/auth/confirm
Container origin: http://api:8080
Local diagnostics: http://127.0.0.1:8080
```

Основной Compose публикует `${PORTWAY_PORT}` только на loopback и направляет его на порт 8080 внутри API-контейнера. Текущий `.env`: `PORTWAY_PORT=8080`, поскольку 8000 занят другим проектом. При смене URL туннеля изменить `.env` и Supabase URL Configuration, затем пересоздать API. Сам шаблон с `.RedirectTo` менять не требуется.

```sh
docker compose up -d
# COMPOSE_PROFILES=execution,funnel сохранён в .env.
docker compose exec funnel tailscale funnel status
```

Funnel работает отдельным Docker-узлом `portway`; контейнер обращается к `http://api:8080` по внутренней сети. После смены HTTPS URL обновить `PORTWAY_PUBLIC_URL`, Site URL и Redirect URL в Supabase, затем `docker compose up -d --no-deps --wait api`. Шаблон письма с `.RedirectTo` менять не требуется. `make check-live` проверяет, что URL ведёт в нужную installation. На хосте с Tailscale DNS может вести через приватный tailnet; публичный путь дополнительно проверен через внешний DNS. [Настройка Docker Funnel и persistent volume](../tailscale/README.md).

Состояние узла хранится в named volume `tailscale-state`. Перезапуск сохраняет node ID и домен; вход при каждом запуске не нужен. Старый host Funnel пока сохранён для перехода и не используется новым `PORTWAY_PUBLIC_URL`. После изменения Supabase URL Configuration его можно отдельно отключить по инструкции Docker Funnel.

Настройки URL Configuration и шаблон письма в Supabase Dashboard этим развёртыванием не изменялись. Проверка `/auth/confirm` без token_hash ожидаемо возвращает HTTP 400 и не заменяет регистрацию с настоящим письмом.

Письмо должно вести непосредственно на Go `/auth/confirm` с token_hash. Стандартный шаблон `{{ .ConfirmationURL }}` не завершает привязку единственного владельца Portway. Callback проверяет OTP сервером и возвращает `portway://auth/confirmed` без JWT; затем требуется login.

Реальная отправка письма выполняется владельцем при регистрации; автоматические проверки Auth settings/JWKS не рассылают письма.

[Шаблоны Supabase](https://supabase.com/docs/guides/auth/auth-email-templates).
