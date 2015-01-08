package server

type Changes struct {
	nodes []string
	head  int
	tail  int
	count int
}

func NewChanges() *Changes {
	return &Changes{
		nodes: make([]string, 32),
	}
}

func (c *Changes) Add(name string) {
	if c.count > 0 {
		if c.head < c.tail {
			for _, x := range c.nodes[c.head:c.tail] {
				if x == name {
					return
				}
			}
		} else {
			for _, x := range c.nodes[c.head:] {
				if x == name {
					return
				}
			}
			for _, x := range c.nodes[:c.tail] {
				if x == name {
					return
				}
			}
		}
	}

	if c.head == c.tail && c.count > 0 {
		nodes := make([]string, len(c.nodes)*2)
		copy(nodes, c.nodes[c.head:])
		copy(nodes[len(c.nodes)-c.head:], c.nodes[:c.head])
		c.head = 0
		c.tail = len(c.nodes)
		c.nodes = nodes
	}
	c.nodes[c.tail] = name
	c.tail = (c.tail + 1) % len(c.nodes)
	c.count++
}

func (c *Changes) Peek() string {
	if c.count == 0 {
		return ""
	}

	return c.nodes[c.head]
}

// Pop pops the top change of the queue.
// Pop assumes the queue is not empty.
func (c *Changes) Pop() {
	c.head = (c.head + 1) % len(c.nodes)
	c.count--
}

func (c *Changes) Len() int {
	return c.count
}
