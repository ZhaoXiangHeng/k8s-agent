#!/usr/bin/env bash
set -euo pipefail

POSTGRES_CONTAINER="k8s-ai-pg"
REDIS_CONTAINER="k8s-ai-redis"
POSTGRES_PORT="55432"
REDIS_PORT="56379"
HOLD_SECONDS="0"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --postgres-port) POSTGRES_PORT="$2"; shift 2 ;;
    --redis-port) REDIS_PORT="$2"; shift 2 ;;
    --hold-seconds) HOLD_SECONDS="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

if docker inspect "$POSTGRES_CONTAINER" >/dev/null 2>&1; then
  docker start "$POSTGRES_CONTAINER" >/dev/null
else
  docker run -d \
    --name "$POSTGRES_CONTAINER" \
    -e POSTGRES_USER=k8s_ai \
    -e POSTGRES_PASSWORD=k8s_ai \
    -e POSTGRES_DB=k8s_ai \
    -p "$POSTGRES_PORT:5432" \
    postgres:16-alpine >/dev/null
fi

if docker inspect "$REDIS_CONTAINER" >/dev/null 2>&1; then
  docker start "$REDIS_CONTAINER" >/dev/null
else
  docker run -d \
    --name "$REDIS_CONTAINER" \
    -p "$REDIS_PORT:6379" \
    redis:7-alpine >/dev/null
fi

echo "PostgreSQL: postgres://k8s_ai:k8s_ai@localhost:${POSTGRES_PORT}/k8s_ai?sslmode=disable"
echo "Redis: localhost:${REDIS_PORT}"

if [[ "$HOLD_SECONDS" != "0" ]]; then
  echo "Keeping WSL session alive for ${HOLD_SECONDS} seconds."
  sleep "$HOLD_SECONDS"
fi
