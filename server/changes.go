package server

type Changes []string

func (c *Changes) Add(name string) {
	for _, x := range *c {
		if x == name {
			return
		}
	}

	*c = append(*c, name)
}

func (c *Changes) Peek() string {
	if len(*c) == 0 {
		return ""
	}

	return (*c)[0]
}

// Pop pops the top change of the queue.
// Pop assumes the queue is not empty.
func (c *Changes) Pop() {
	*c = (*c)[1:]
}
