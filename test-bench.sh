#!/bin/sh

export GOMAXPROCS=4

./dcounter server -client="127.0.0.1:10001" &
P=$!

sleep 1

stop() {
#  kill $B

  echo stop
  kill -2 $P

  wait $P

  exit 0
}

trap stop SIGINT

./dcounter bench -connect="127.0.0.1:10001" get test
