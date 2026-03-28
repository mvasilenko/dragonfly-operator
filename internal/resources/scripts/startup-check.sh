#!/bin/sh

HOST="localhost"
PORT=${HEALTHCHECK_PORT:-6379}

RESPONSE=$(redis-cli -h "$HOST" -p "$PORT" --no-auth-warning PING 2>/dev/null)

# Empty response means connection refused (Dragonfly not up yet)
[ -z "$RESPONSE" ] && exit 1

# LOADING means Dragonfly is still loading the dataset from disk
case "$RESPONSE" in
  *LOADING*) exit 1 ;;
esac

# Any other response (PONG, NOAUTH, ERR, etc.) means Dragonfly is running
exit 0
