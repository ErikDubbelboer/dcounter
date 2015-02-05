#!/bin/sh

export GOMAXPROCS=4

echo start 1
./dcounter server -bind="127.0.0.1:9001" -client="127.0.0.1:10001" &
P1=$!
sleep 1
echo start 2
./dcounter server -bind="127.0.0.1:9002" -client="127.0.0.1:10002" &
P2=$!
sleep 1

echo join 1 with 2
./dcounter cli -connect="127.0.0.1:10001" join "127.0.0.1:9002"
echo increment 1 by 1.0
./dcounter cli -connect="127.0.0.1:10001" inc test 1
echo increment 2 by 1.0
./dcounter cli -connect="127.0.0.1:10002" inc test 1
sleep 1
echo expect 2.0 from 2
./dcounter cli -connect="127.0.0.1:10002" get test
echo reset 2
./dcounter cli -connect="127.0.0.1:10002" set test 0
sleep 1
echo expect 0.0 from 1
./dcounter cli -connect="127.0.0.1:10001" get test
echo expect 0.0 from 2
./dcounter cli -connect="127.0.0.1:10002" get test
echo increment 1 by 1.0
./dcounter cli -connect="127.0.0.1:10001" inc test 1
echo increment 2 by 1.0
./dcounter cli -connect="127.0.0.1:10002" inc test 1
sleep 1
echo expect 2.0 from 1
./dcounter cli -connect="127.0.0.1:10001" get test
echo expect 2.0 from 2
./dcounter cli -connect="127.0.0.1:10002" get test

echo stop 1
kill -2 $P1
wait $P1

echo expect 2.0 from 2
./dcounter cli -connect="127.0.0.1:10002" get test

echo start 3
./dcounter server -bind="127.0.0.1:9003" -client="127.0.0.1:10003" &
P3=$!
sleep 1

echo join 3 with 2
./dcounter cli -connect="127.0.0.1:10003" join "127.0.0.1:9002"
sleep 1

echo expect 2.0 from 3
./dcounter cli -connect="127.0.0.1:10003" get test

echo stop 2
kill -2 $P2
wait $P2

echo expect 2.0 from 3
./dcounter cli -connect="127.0.0.1:10003" get test

echo stop 3
kill -2 $P3
wait $P3
