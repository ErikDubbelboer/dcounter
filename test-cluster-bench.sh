#!/bin/sh

export GOMAXPROCS=4

rm -f test1.sqlite test2.sqlite

echo start 1
./dcounter server -db=test1.sqlite -bind="127.0.0.1:9001" -client="127.0.0.1:10001" &
P1=$!
sleep 1
echo start 2
./dcounter server -db=test2.sqlite -bind="127.0.0.1:9002" -client="127.0.0.1:10002" &
P2=$!
sleep 1

echo join 1 with 2
./dcounter cli -connect="127.0.0.1:10001" join "127.0.0.1:9002"

./dcounter bench -connect="127.0.0.1:10001" inc test 0.1 &
A=$!

./dcounter bench -connect="127.0.0.1:10002" inc test -0.1 &
B=$!

stop() {
  kill $A
  kill $B

  echo stop 1 and 2
  kill -2 $P1
  kill -2 $P2

  wait $P1
  wait $P2

  rm -f test1.sqlite test2.sqlite

  exit 0
}

trap stop SIGINT

while :
do
  ./dcounter cli -connect="127.0.0.1:10001" get test
  sleep 1
done
