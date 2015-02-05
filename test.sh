#!/bin/sh

export GOMAXPROCS=4

./dcounter server -client="127.0.0.1:9371" &
P=$!

sleep 1

echo 0
./dcounter cli -connect="127.0.0.1:9371" get test
./dcounter cli -connect="127.0.0.1:9371" inc test 1
./dcounter cli -connect="127.0.0.1:9371" inc test 1
echo 2
./dcounter cli -connect="127.0.0.1:9371" get test
./dcounter cli -connect="127.0.0.1:9371" set test 0
echo 0
./dcounter cli -connect="127.0.0.1:9371" get test
./dcounter cli -connect="127.0.0.1:9371" inc test -1
./dcounter cli -connect="127.0.0.1:9371" inc test -1
echo -2
./dcounter cli -connect="127.0.0.1:9371" get test
./dcounter cli -connect="127.0.0.1:9371" inc test 1
time ./dcounter cli -connect="127.0.0.1:9371" inc test 1
echo 0
./dcounter cli -connect="127.0.0.1:9371" get test


kill -2 $P
wait $P
