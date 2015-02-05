package server

type Resets [][]byte

func (r *Resets) Add(b []byte) {
	*r = append(*r, b)
}

func (r *Resets) Peek() []byte {
	if len(*r) == 0 {
		return nil
	}

	return (*r)[0]
}

// Pop pops the top change of the queue.
// Pop assumes the queue is not empty.
func (r *Resets) Pop() {
	*r = (*r)[1:]
}
