package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/codegangsta/cli"
)

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:  "genkey",
		Usage: "Generate a random encryption key.",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) {
			data := make([]byte, 32)
			if _, err := rand.Read(data); err != nil {
				log.Printf("[ERR] %v", err)
			} else {
				fmt.Println(base64.StdEncoding.EncodeToString(data))
			}
		},
	})
}
