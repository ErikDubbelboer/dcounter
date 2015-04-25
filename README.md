# dcounter
[![Build Status](https://travis-ci.org/atomx/dcounter.svg?branch=master)](https://travis-ci.org/atomx/dcounter) [![GoDoc](https://godoc.org/github.com/atomx/dcounter/api?status.png)](https://godoc.org/github.com/atomx/dcounter/api)

## Installation 

```
$ go get github.com/atomx/dcounter
```

### API usage

```go
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
```
