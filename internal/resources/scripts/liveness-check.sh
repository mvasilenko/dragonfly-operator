#!/bin/sh

HOST="localhost"
PORT=${HEALTHCHECK_PORT:-6379}

RESPONSE=$(redis-cli -h "$HOST" -p "$PORT" --no-auth-warning PING 2>/dev/null)

# Empty response means connection refused — Dragonfly is dead
[ -z "$RESPONSE" ] && exit 1

# Any response (PONG, NOAUTH, ERR, etc.) means process is alive
exit 0
