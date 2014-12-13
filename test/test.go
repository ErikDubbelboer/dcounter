package main

import (
	"log"
	"math/rand"
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
			l.Lock()
			amount, consistent, err := api.Get("campaign:1")
			l.Unlock()
			log.Printf("%f %v %v\n", amount, consistent, err)

			time.Sleep(time.Second * 10)
		}
	}()

	go func() {
		for {
			l.Lock()
			if err := api.Reset("campaign:1"); err != nil {
				log.Println(err)
			}
			l.Unlock()

			time.Sleep(time.Minute * 5)
		}
	}()

	r := rand.New(rand.NewSource(0))

	for {
		l.Lock()
		if err := api.Inc("campaign:1", r.Float64()); err != nil {
			log.Println(err)
		}
		l.Unlock()

		time.Sleep(time.Millisecond * 10)
	}
}
