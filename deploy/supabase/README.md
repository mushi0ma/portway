# Настройка email callback

1. В Supabase → Authentication → URL Configuration установить Site URL равным `PORTWAY_PUBLIC_URL` из `.env`.
2. В Redirect URLs добавить точный `PORTWAY_PUBLIC_URL/auth/confirm`.
3. В Authentication → Email → Templates → Confirm signup вставить `confirm-signup.html` из этого каталога и сохранить. Go signup передаёт свой callback через `redirect_to`; шаблон использует `{{ .RedirectTo }}` и `{{ .TokenHash }}`.
4. Оставить email confirmation включённым. Административные настройки Supabase нельзя менять через publishable key.

Текущая установка использует Tailscale Funnel и постоянное имя узла:

```text
Site URL: https://cachyos-x8664.tail1443b8.ts.net
Redirect URL: https://cachyos-x8664.tail1443b8.ts.net/auth/confirm
Local origin: http://127.0.0.1:8080
```

Основной Compose публикует `${PORTWAY_PORT}` только на loopback и направляет его на порт 8080 внутри API-контейнера. Текущий `.env`: `PORTWAY_PORT=8080`, поскольку 8000 занят другим проектом. При смене URL туннеля изменить `.env` и Supabase URL Configuration, затем пересоздать API. Сам шаблон с `.RedirectTo` менять не требуется.

```sh
# Для настройки Funnel на уже авторизованном хосте:
tailscale funnel --bg http://127.0.0.1:8080
tailscale funnel status
```

`--bg` сохраняет публикацию при перезапуске Tailscale/хоста. В текущей установке Funnel уже включён; повторная настройка при каждом запуске API не нужна. После смены HTTPS URL обновить `PORTWAY_PUBLIC_URL`, Site URL и Redirect URL в Supabase, затем `docker compose up -d --no-deps --wait api`. Команда `make check-live` подтвердит, что по настроенному URL отвечает именно эта installation. На хосте с Tailscale DNS может вести через приватный tailnet; доступ из интернета дополнительно проверять с устройства вне tailnet. [Документация Funnel](https://tailscale.com/docs/reference/tailscale-cli/funnel).

Go API, Kafka и workers уже работают в Docker. Текущий Tailscale daemon работает на хосте и обращается к опубликованному loopback-порту API. При переносе самого Tailscale в контейнер необходимо сохранять `TS_STATE_DIR=/var/lib/tailscale` в постоянном volume: без сохранённого состояния создаётся новый узел, поэтому прежнее DNS-имя нельзя считать перенесённым автоматически. Перенос действующего узла в контейнер пока не выполнялся. [Состояние Tailscale в Docker](https://tailscale.com/docs/features/containers/docker/docker-params).

Настройки URL Configuration и шаблон письма в Supabase Dashboard этим развёртыванием не изменялись. Проверка `/auth/confirm` без token_hash ожидаемо возвращает HTTP 400 и не заменяет регистрацию с настоящим письмом.

Письмо должно вести непосредственно на Go `/auth/confirm` с token_hash. Стандартный шаблон `{{ .ConfirmationURL }}` не завершает привязку единственного владельца Portway. Callback проверяет OTP сервером и возвращает `portway://auth/confirmed` без JWT; затем требуется login.

Реальная отправка письма выполняется владельцем при регистрации; автоматические проверки Auth settings/JWKS не рассылают письма.

[Шаблоны Supabase](https://supabase.com/docs/guides/auth/auth-email-templates).
