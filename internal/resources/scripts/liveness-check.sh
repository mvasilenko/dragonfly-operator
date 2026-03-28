#!/bin/sh

HOST="localhost"
PORT=${HEALTHCHECK_PORT:-6379}

redis-cli -h "$HOST" -p "$PORT" --no-auth-warning PING 2>/dev/null | grep -q "PONG"
