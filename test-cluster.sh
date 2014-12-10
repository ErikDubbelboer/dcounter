#!/bin/sh

export GOMAXPROCS=4

rm -f test1.sqlite test2.sqlite

./dcounter server -db=test1.sqlite -bind="127.0.0.1:9001" -client="127.0.0.1:10001" &
P1=$!

./dcounter server -db=test2.sqlite -bind="127.0.0.1:9002" -client="127.0.0.1:10002" &
P2=$!

sleep 1

./dcounter cli -connect="127.0.0.1:10001" join "127.0.0.1:9002"
./dcounter cli -connect="127.0.0.1:10001" inc test 1
sleep 1
echo 1.0
./dcounter cli -connect="127.0.0.1:10002" get test

kill -2 $P1
wait $P1

echo 1.0
./dcounter cli -connect="127.0.0.1:10002" get test

kill -2 $P2
wait $P2

rm -f test1.sqlite test2.sqlite

