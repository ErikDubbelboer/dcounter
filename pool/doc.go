/*
Package pool provides a simple blocking connection pool.

	package main

	import (
		"fmt"
		"github.com/atomx/dcounter/pool"
	)

	func main() {
		p := dcounter.New("tcp", "127.0.0.1:9374", 32)

		d, err := p.Get(time.Second)
		if err != nil {
			panic(err)
		}
		defer p.Put(d)

		fmt.Println(d.Get("test"))
	}
*/
package dcounter
