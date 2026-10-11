#!/bin/sh
set -eu
# Disposable process-level check of the real production image. No Supabase credentials.
project="portway-smoke-$$"
work=$(mktemp -d /tmp/portway-smoke.XXXXXX)
db="${project}-db"
api="${project}-api"
cleanup() {
  docker rm -f "$api" "$db" >/dev/null 2>&1 || true
  docker network rm "$project" >/dev/null 2>&1 || true
  rm -rf "$work"
}
trap cleanup EXIT INT TERM
umask 077
printf '%s' "postgres://portway_app:disposable-app-password@${db}:5432/portway_test?sslmode=disable" > "$work/database_url"
printf '%s' "postgres://postgres:disposable-migration-password@${db}:5432/portway_test?sslmode=disable" > "$work/migration_database_url"
openssl rand -base64 32 > "$work/master_key"
openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n' > "$work/setup_token"
docker build --target runtime -t portway-api:smoke .
# A bridge is needed for the loopback port publication checked below.
docker network create "$project" >/dev/null
docker run -d --name "$db" --network "$project" --tmpfs /var/lib/postgresql/data \
  -e POSTGRES_PASSWORD=disposable-migration-password -e POSTGRES_DB=portway_test \
  -v "$PWD/deploy/test/init.sql:/docker-entrypoint-initdb.d/001-roles.sql:ro" postgres:17-alpine@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24 >/dev/null
i=0
until docker exec "$db" pg_isready -U postgres -d portway_test >/dev/null 2>&1; do
  i=$((i+1)); test "$i" -lt 60 || exit 1; sleep 1
done
# pg_isready can see the bootstrap socket; require the role over TCP after init finishes.
i=0
until docker exec -e PGPASSWORD=disposable-app-password "$db" psql -h 127.0.0.1 -U portway_app -d portway_test -c 'SELECT 1' >/dev/null 2>&1; do
  i=$((i+1)); test "$i" -lt 60 || exit 1; sleep 1
done
run_image() {
  docker run --network "$project" --user "$(id -u):$(id -g)" --read-only --cap-drop ALL --security-opt no-new-privileges \
    -e PORTWAY_ENV=test -e PORTWAY_HTTP_ADDR=0.0.0.0:8080 \
    -e PORTWAY_INSTALLATION_ID=00000000-0000-4000-8000-000000000001 \
    -e PORTWAY_PUBLIC_URL=http://localhost:8080 -e PORTWAY_AUTH_URL=http://unavailable.test/auth/v1 \
    -e PORTWAY_AUTH_PUBLISHABLE_KEY=public-test-key \
    -e PORTWAY_DATABASE_URL_FILE=/run/secrets/database_url \
    -e PORTWAY_MASTER_KEY_FILE=/run/secrets/master_key -e PORTWAY_SETUP_TOKEN_FILE=/run/secrets/setup_token \
    -v "$work/database_url:/run/secrets/database_url:ro" \
    -v "$work/master_key:/run/secrets/master_key:ro" \
    -v "$work/setup_token:/run/secrets/setup_token:ro" "$@"
}
run_image --rm --no-healthcheck \
  -e PORTWAY_MIGRATION_DATABASE_URL_FILE=/run/secrets/migration_database_url \
  -v "$work/migration_database_url:/run/secrets/migration_database_url:ro" portway-api:smoke migrate
run_image -d --name "$api" -p 127.0.0.1::8080 portway-api:smoke api >/dev/null
address=$(docker port "$api" 8080/tcp)
i=0
until curl --fail --silent "http://${address}/healthz" >/dev/null; do
  i=$((i+1)); if test "$i" -ge 30; then docker logs "$api"; exit 1; fi; sleep 1
done
test "$(curl -s -o /dev/null -w '%{http_code}' "http://${address}/readyz")" = 503
test "$(curl -s -o /dev/null -w '%{http_code}' "http://${address}/v1/agents")" = 401
curl --fail --silent "http://${address}/v1/client-config" | python3 -c 'import json,sys; d=json.load(sys.stdin)["data"]; assert d["setup_required"] and d["api_version"]=="v1"'
docker restart "$api" >/dev/null
# Docker may allocate a different ephemeral host port when restarting.
address=$(docker port "$api" 8080/tcp)
i=0
until curl --fail --silent "http://${address}/healthz" >/dev/null; do
  i=$((i+1)); test "$i" -lt 30 || exit 1; sleep 1
done
echo 'Production image smoke passed: non-root, read-only, migrations, health, auth guard, readiness and restart.'
