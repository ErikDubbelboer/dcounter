package main

import (
	"fmt"
)

type Buffer []byte

func (b *Buffer) Write(p []byte) (int, error) {
	lb := len(*b)
	lp := len(p)

	if lb+lp > cap(*b) {
		return 0, fmt.Errorf("buffer is full")
	}

	*b = append(*b, p...)

	return lp, nil
}
