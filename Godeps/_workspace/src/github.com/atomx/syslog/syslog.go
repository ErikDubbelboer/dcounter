package syslog

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

// See: http://en.wikipedia.org/wiki/Syslog#Severity_levels
const (
	ALERT   = 1
	ERROR   = 3
	WARNING = 4
	INFO    = 5
)

type Syslog struct {
	conn    net.Conn
	mutex   sync.Mutex
	network string
	address string
	tag     string
}

func New(network, address, tag string) *Syslog {
	return &Syslog{
		network: network,
		address: address,
		tag:     tag,
	}
}

func (sl *Syslog) Write(p []byte) (n int, err error) {
	s := strings.TrimRightFunc(string(p), unicode.IsSpace)

	if strings.Contains(s, " [ERR] ") {
		sl.outputAndRetry(ERROR, s)
	} else if strings.Contains(s, " [WARN] ") {
		sl.outputAndRetry(WARNING, s)
	} else if strings.Contains(s, " [INFO] ") {
		sl.outputAndRetry(INFO, s)
	} else {
		sl.outputAndRetry(INFO, s)
	}

	return len(p), nil
}

func (sl *Syslog) connect() (err error) {
	if sl.conn != nil {
		sl.conn.Close() // Ignore any error from Close.
	}

	sl.conn, err = net.Dial(sl.network, sl.address)
	return
}

func (sl *Syslog) outputAndRetry(priority int, message string) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	var err error

	if sl.conn != nil {
		if err = sl.output(priority, message); err == nil {
			return
		}
	}

	if err = sl.connect(); err == nil && sl.conn != nil {
		if err = sl.output(priority, message); err == nil {
			return
		}
	}

	// If we can't write to syslog at all we write to stderr as last resort.
	fmt.Fprintf(os.Stderr, "%v\n", err)
	fmt.Fprintf(os.Stderr, "%s\n", message)
}

func (sl *Syslog) output(priority int, message string) (err error) {
	timestamp := time.Now().Format(time.Stamp)

	//if _, file, line, ok := runtime.Caller(3); ok {
	//	message += "\n" + file + ":" + strconv.FormatInt(int64(line), 10)
	//}
	//message += "\n\n"+ string(debug.Stack())

	// output() generates and writes a syslog formatted string. The
	// format is as follows: <PRI>TIMESTAMP TAG[PID]:MSG
	// For PRI see: http://en.wikipedia.org/wiki/Syslog#Calculating_Priority_Value
	// We use local0 (16) as facility.
	_, err = fmt.Fprintf(sl.conn, "<%d>%s %s[%d]: %s\n",
		(16*8)+priority, timestamp, sl.tag, os.Getpid(), message,
	)
	return
}
