package server

import (
	"io"
)

type buffer []byte

func (b *buffer) Write(p []byte) (int, error) {
	lb := len(*b)
	lp := len(p)

	if lb+lp > cap(*b) {
		return 0, io.EOF
	}

	*b = append(*b, p...)

	return lp, nil
}
