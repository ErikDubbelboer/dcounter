package server

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/atomx/dcounter/api"
)

type TestServer struct {
	client string
	bind   string
	s      *Server
	t      *testing.T
}

func NewTestServer(t *testing.T, name string) *TestServer {
	s := &TestServer{
		client: "127.0.0.1:" + strconv.FormatInt(1000+rand.Int63n(10000), 10),
		bind:   "127.0.0.1:" + strconv.FormatInt(1000+rand.Int63n(10000), 10),
		t:      t,
	}

	s.s = New(s.bind, s.client)
	s.s.Config.Name = name
	s.s.Config.LogOutput = s

	s.t.Logf("%s: starting", s.s.Config.Name)
	if err := s.s.Start(); err != nil {
		t.Fatal(err)
	}

	return s
}

func (s *TestServer) Stop() {
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

func (s *TestServer) Debug() {
	s.s.logDebug()
}

func (s *TestServer) Get(name string, value float64, consistent bool) {
	s.t.Logf("%s: get %s %f", s.s.Config.Name, name, value)

	api, err := api.Dial("tcp", s.client)
	if err != nil {
		s.t.Fatal(err)
	}

	if v, c, err := api.Get(name); err != nil {
		s.t.Error(err)
	} else if v != value {
		s.t.Errorf("expected %f got %f", value, v)
	} else if c != consistent {
		s.t.Errorf("expected %v got %v", consistent, c)
	}

	if err := api.Close(); err != nil {
		s.t.Error(err)
	}
}

func (s *TestServer) Inc(name string, diff float64) {
	s.t.Logf("%s: inc %s %f", s.s.Config.Name, name, diff)

	api, err := api.Dial("tcp", s.client)
	if err != nil {
		s.t.Fatal(err)
	}

	if err := api.Inc(name, diff); err != nil {
		s.t.Error(err)
	}

	if err := api.Close(); err != nil {
		s.t.Error(err)
	}
}

func (s *TestServer) Reset(name string) {
	s.t.Logf("%s: reset %s", s.s.Config.Name, name)

	api, err := api.Dial("tcp", s.client)
	if err != nil {
		s.t.Fatal(err)
	}

	if err := api.Reset(name); err != nil {
		s.t.Error(err)
	}

	if err := api.Close(); err != nil {
		s.t.Error(err)
	}
}

func (s *TestServer) Join(o *TestServer) {
	s.t.Logf("%s: join %s", s.s.Config.Name, o.s.Config.Name)

	api, err := api.Dial("tcp", s.client)
	if err != nil {
		s.t.Fatal(err)
	}

	if err := api.Join([]string{o.bind}); err != nil {
		s.t.Error(err)
	}

	if err := api.Close(); err != nil {
		s.t.Error(err)
	}
}
