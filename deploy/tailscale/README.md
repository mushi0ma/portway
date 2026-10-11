# Отдельный узел Funnel в Docker

Профиль `funnel` создаёт отдельный узел `portway` и проксирует только Go API (`http://api:8080`). Доступ к Docker socket, host networking и `/dev/net/tun` не нужен: используется userspace networking. Kafka, execution и Supabase через Funnel не публикуются.

## Первое подключение

```sh
docker compose --profile funnel up -d funnel
# Выполнить локально: команда показывает одноразовую ссылку авторизации.
docker compose exec funnel tailscale up --hostname=portway --accept-dns=false
```

Открыть полученную ссылку и авторизовать узел в своём tailnet. `TS_BOOT_TIMEOUT=30m` оставляет время для интерактивного входа; стандартные 60 секунд приводят к перезапуску и смене ссылки до завершения входа. Ключ Tailscale в `.env` не требуется. Если политика tailnet не разрешает Funnel, разрешить его для этого узла в Tailscale admin console. `TS_SERVE_CONFIG` задаёт HTTPS и AllowFunnel; имя подставляет containerboot после авторизации.

```sh
docker compose exec funnel tailscale funnel status
docker compose exec funnel tailscale status --json
```

Сохранить `COMPOSE_PROFILES=execution,funnel` в `.env` после настройки сервисов: обычный `docker compose up -d` будет включать оба профиля.

Скопировать фактический HTTPS-домен из `funnel status` в `PORTWAY_PUBLIC_URL`. При конфликте имён Tailscale может добавить суффикс, поэтому домен нельзя угадывать. Обновить Site URL и точный `/auth/confirm` в Supabase по [инструкции](../supabase/README.md), затем:

```sh
docker compose up -d --no-deps --wait api
make check-live
```

С телефона вне tailnet проверить `/healthz` и совпадение installation в `/v1/client-config`. Только после проверки нового адреса можно отдельно отключить старый Funnel на хосте: `tailscale funnel --https=443 off`. Это не требуется для работы нового узла; Tailscale хоста остаётся отдельным устройством.

## Перезапуск и версия

Состояние и ключ узла находятся в named volume `tailscale-state`, исключённом из Git/build context. Не удалять volume и не запускать два узла с копией одного состояния. Обычные restart/recreate сохраняют identity; `docker compose down --volumes` её удалит. Для старта после обычного `down` снова указать `--profile funnel`; уже созданный контейнер автоматически перезапускается Docker по `unless-stopped`.

Образ закреплён digest в `compose.funnel.yaml`. Для планового обновления: `docker pull tailscale/tailscale:stable`, затем `docker image inspect tailscale/tailscale:stable --format '{{index .RepoDigests 0}}'` и сохранить результат в `PORTWAY_TAILSCALE_IMAGE` в `.env`. Тег `stable` используется только для поиска нового образа; рабочая конфигурация сохраняет digest.

Образ v1.102.5 загружен, контейнер запускается с userspace networking. Для первого входа предусмотрен таймаут 30 минут. Фактический статус авторизации и публичного пути фиксируется в [протоколе среза](../../docs/evidence/execution-foundation-2026-10-12.txt). Старый host Funnel сохраняется до проверки нового узла.

Источники: [Docker parameters](https://tailscale.com/docs/features/containers/docker/docker-params), [Funnel CLI](https://tailscale.com/docs/reference/tailscale-cli/funnel), [boot timeout в исходниках v1.102.5](https://github.com/tailscale/tailscale/blob/v1.102.5/cmd/containerboot/settings.go).
