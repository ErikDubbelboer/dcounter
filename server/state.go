package server

type state struct {
	Active byte
	When   int64 // time.Now().Unix()
}
