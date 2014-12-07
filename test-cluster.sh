#!/bin/sh

rm -f test1.sqlite test2.sqlite

./dcounter -id=1 -db=test1.sqlite -bind="127.0.0.1:9371" -client="127.0.0.1:9372" &
P1=$!

./dcounter -id=2 -db=test2.sqlite -bind="127.0.0.1:9381" -client="127.0.0.1:9382" &
P2=$!

sleep 1

set -x

./cli/cli -connect="127.0.0.1:9372" join "127.0.0.1:9381"
./cli/cli -connect="127.0.0.1:9372" inc test 1
sleep 1
./cli/cli -connect="127.0.0.1:9382" get test

set +x

kill -2 $P1
kill -2 $P2
wait $P1
wait $P2

rm -f test1.sqlite test2.sqlite

