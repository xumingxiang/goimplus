package main

import (
	"goim/libs/hash/cityhash"
	"time"
)

var (
	maxInt        = 1<<31 - 1
	emptyJSONBody = []byte("{}")
)

type ServerOptions struct {
	CliProto         int
	SvrProto         int
	HandshakeTimeout time.Duration
	TCPKeepalive     bool
	TCPRcvbuf        int
	TCPSndbuf        int
}

type Server struct {
	Stat      *Stat
	Buckets   []*Bucket // subkey bucket
	bucketIdx uint32
	round     *Round // accept round store
	operator  Operator
	Options   ServerOptions
}

// NewServer returns a new Server.
func NewServer(st *Stat, b []*Bucket, r *Round, o Operator, options ServerOptions) *Server {
	s := new(Server)
	s.Stat = st
	s.Buckets = b
	s.bucketIdx = uint32(len(b))
	s.round = r
	s.operator = o
	s.Options = options
	return s
}

func (server *Server) Bucket(subKey string) *Bucket {
	idx := cityhash.CityHash32([]byte(subKey), uint32(len(subKey))) % server.bucketIdx
	if Debug {
		guluLogger.Debug("\"%s\" hit channel bucket index: %d use cityhash", subKey, idx)
	}
	return server.Buckets[idx]
}
