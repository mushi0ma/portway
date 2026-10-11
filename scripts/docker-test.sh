#!/bin/sh
set -eu
# A unique project never stops the user's other stacks or another test run.
project="portway-test-$$"
cleanup() { docker compose -p "$project" -f compose.test.yaml down --volumes --remove-orphans >/dev/null; }
trap cleanup EXIT INT TERM
docker compose -p "$project" -f compose.test.yaml build tests
docker compose -p "$project" -f compose.test.yaml up -d --wait postgres kafka
docker compose -p "$project" -f compose.test.yaml run --rm --no-deps kafka-init
if ! docker compose -p "$project" -f compose.test.yaml run --rm --no-deps tests; then
  docker compose -p "$project" -f compose.test.yaml logs --tail 100 kafka
  exit 1
fi
