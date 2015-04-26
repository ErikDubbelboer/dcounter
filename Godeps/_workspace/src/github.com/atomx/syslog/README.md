# dcounter
[![GoDoc](https://godoc.org/github.com/atomx/syslog?status.png)](https://godoc.org/github.com/atomx/syslog)

syslog is a simple io.Writer that writes to syslog over a network connection.
The difference with the [buildin syslog](https://golang.org/pkg/log/syslog/) package
is that it will detect the priority based on [ERR] [WARN] or [INFO] tags.

The main usage of this package is to import it for the side effects
cause by setting the SYSLOG environment variable. Setting this variable will
change the default writer of the log package.

SYSLOG should be set to network:address, for example udp:localhost:514

## Installation

```
$ go get github.com/atomx/syslog
```

### Usage

```go
package main

import (
  "log"
  "os"
  "path"

  "github.com/atomx/syslog"
)

func main() {
  log.SetOutput(syslog.New("udp", "localhost:514", path.Base(os.Args[0])))

  log.Printf("[ERR] this will be logged with the error severity")
}
```
