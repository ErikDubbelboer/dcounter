package syslog

import (
	"log"
	"os"
	"path"
	"strings"
)

func init() {
	if os.Getenv("SYSLOG") != "" {
		parts := strings.SplitN(os.Getenv("SYSLOG"), ":", 2)

		if len(parts) != 2 {
			log.Fatalf("[ERR] invalid SYSLOG, expected network:addr, got %s", os.Getenv("SYSLOG"))
		}

		tag := os.Getenv("NAME")
		if tag == "" {
			tag = path.Base(os.Args[0])
		}

		s := New(parts[0], parts[1], tag)

		log.SetOutput(s)
		log.SetFlags(log.Lshortfile)
	}
}
