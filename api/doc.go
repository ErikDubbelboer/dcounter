/*
Package api exports a simple API to use a dcounter server.

	package main

	import (
		"fmt"
		"github.com/atomx/dcounter/api"
	)

	func main() {
		d, err := dcounter.Dial("tcp", "127.0.0.1:9374")
		if err != nil {
			panic(err)
		}

		d.Inc("test", 1.2)
		fmt.Println(d.Get("test"))
	}
*/
package dcounter
