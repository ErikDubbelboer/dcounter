package server

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/atomx/dcounter/api"
)

type BorT interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	SkipNow()
}

type TestServer struct {
	client    string
	advertise string
	s         *Server
	t         BorT
	a         dcounter.API
}

func NewTestServerOn(t BorT, name, bind, advertise string) *TestServer {
	s := &TestServer{
		client:    "127.0.0.1:" + strconv.FormatInt(1000+rand.Int63n(10000), 10),
		advertise: advertise,
		t:         t,
	}

	var err error
	s.s, err = New(name, bind, advertise, s.client)
	if err != nil {
		t.Fatal(err)
	}

	s.s.Config.LogOutput = s
	s.s.Config.GossipInterval = 100 * time.Millisecond

	s.t.Logf("%s: starting", s.s.Config.Name)
	if err := s.s.Start(); err != nil {
		t.Fatal(err)
	}

	s.a, err = dcounter.Dial("tcp", s.client)
	if err != nil {
		s.t.Fatal(err)
	}

	return s
}

func NewTestServer(t BorT, name string) *TestServer {
	bind := "127.0.0.1:" + strconv.FormatInt(1000+rand.Int63n(10000), 10)
	return NewTestServerOn(t, name, bind, bind)
}

func (s *TestServer) Stop() {
	if err := s.a.Close(); err != nil {
		s.t.Error(err)
	}

	s.t.Logf("%s: stopping", s.s.Config.Name)
	if err := s.s.Stop(); err != nil {
		s.t.Fatal(err)
	}
}

func (s *TestServer) Kill() {
	s.t.Logf("%s: killing", s.s.Config.Name)
	s.s.Kill()
}

func (s *TestServer) Write(p []byte) (n int, err error) {
	s.t.Log(s.s.Config.Name + ": " + strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

func (s *TestServer) Get(name string, value float64, consistent bool) {
	s.t.Logf("%s: get %s %f", s.s.Config.Name, name, value)

	if v, c, err := s.a.Get(name); err != nil {
		s.t.Error(err)
	} else if v != value {
		s.t.Errorf("expected %f got %f", value, v)
	} else if c != consistent {
		s.t.Errorf("expected %v got %v", consistent, c)
	}

}

func (s *TestServer) Inc(name string, diff float64) {
	s.t.Logf("%s: inc %s %f", s.s.Config.Name, name, diff)

	if err := s.a.Inc(name, diff); err != nil {
		s.t.Error(err)
	}
}

func (s *TestServer) Set(name string, value float64) {
	s.t.Logf("%s: set %s %f", s.s.Config.Name, name, value)

	if err := s.a.Set(name, value); err != nil {
		s.t.Error(err)
	}
}

func (s *TestServer) Join(o *TestServer) {
	s.t.Logf("%s: join %s", s.s.Config.Name, o.s.Config.Name)

	if err := s.a.Join([]string{o.advertise}); err != nil {
		s.t.Error(err)
	}
}

func (s *TestServer) JoinOn(bind string) {
	s.t.Logf("%s: join %s", s.s.Config.Name, bind)

	if err := s.a.Join([]string{bind}); err != nil {
		s.t.Error(err)
	}
}
