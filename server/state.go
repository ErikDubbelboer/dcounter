package server

type State struct {
	Active byte
	When   int64 // time.Now().Unix()
}
