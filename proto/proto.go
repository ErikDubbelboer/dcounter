package proto

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

type Proto struct {
	r *bufio.Reader
	w *bufio.Writer
}

func (p Proto) Error(err error) error {
	return p.Write("ERR", []string{err.Error()})
}

func (p Proto) Read() (cmd string, args []string, err error) {
	if s, err := p.r.ReadString('\n'); err != nil {
		return "", nil, err
	} else {
		// Strip the '\n'.
		s = s[:len(s)-1]

		if strings.HasPrefix(s, "ERR ") {
			return "", nil, fmt.Errorf(s[4:])
		} else {
			ca := strings.Split(s, " ")

			return ca[0], ca[1:], nil
		}
	}
}

func (p Proto) Write(cmd string, args []string) error {
	cmdargs := append([]string{cmd}, args...)

	if _, err := p.w.WriteString(strings.Join(cmdargs, " ") + "\n"); err != nil {
		return err
	} else if err := p.w.Flush(); err != nil {
		return err
	}

	return nil
}

func New(conn net.Conn) *Proto {
	return &Proto{
		r: bufio.NewReader(conn),
		w: bufio.NewWriter(conn),
	}
}
