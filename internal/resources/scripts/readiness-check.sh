#!/bin/sh

HOST="localhost"
PORT=${HEALTHCHECK_PORT:-6379}

RESPONSE=$(redis-cli -h "$HOST" -p "$PORT" --no-auth-warning PING 2>/dev/null)

# Fail if Dragonfly is still loading the dataset
case "$RESPONSE" in
  *LOADING*)
    exit 1
    ;;
  PONG)
    exit 0
    ;;
  *)
    exit 1
    ;;
esac
