#!/bin/sh

HOST="localhost"
PORT=${HEALTHCHECK_PORT:-9999}

redis-cli -h "$HOST" -p "$PORT" PING 2>/dev/null | grep -q "PONG"
