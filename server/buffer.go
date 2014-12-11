package server

import (
	"io"
)

type Buffer []byte

func (b *Buffer) Write(p []byte) (int, error) {
	lb := len(*b)
	lp := len(p)

	if lb+lp > cap(*b) {
		return 0, io.EOF
	}

	*b = append(*b, p...)

	return lp, nil
}
