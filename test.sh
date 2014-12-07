#!/bin/sh

rm -f test.sqlite

./dcounter -id=1 -db=test.sqlite -client="127.0.0.1:9371" &
P=$!

sleep 1

./cli/cli -connect="127.0.0.1:9371" get test
./cli/cli -connect="127.0.0.1:9371" inc test 1
./cli/cli -connect="127.0.0.1:9371" inc test 1
./cli/cli -connect="127.0.0.1:9371" get test

kill -2 $P
wait $P

rm -f test.sqlite

