#!/bin/sh
printf "PING\r\n" | nc -w1 127.0.0.1 "${HEALTHCHECK_PORT}" | grep -q "^+PONG"
