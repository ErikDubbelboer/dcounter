package main

import (
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/atomx/dcounter/api"
)

func main() {
	var l sync.Mutex

	api, err := api.Dial("tcp", "127.0.0.1:9374")
	if err != nil {
		panic(err)
	}
	defer api.Close()

	go func() {
		for {
			n := strconv.FormatInt(rand.Int63n(600*3), 10)

			l.Lock()
			if err := api.Reset("campaign:" + n); err != nil {
				log.Println(err)
			}
			l.Unlock()

			time.Sleep(time.Minute * 5)
		}
	}()

	for {
		n := strconv.FormatInt(rand.Int63n(600*3), 10)

		l.Lock()
		if err := api.Inc("campaign:"+n, rand.Float64()); err != nil {
			log.Println(err)
		}
		l.Unlock()

		time.Sleep(time.Millisecond * 10)
	}
}
