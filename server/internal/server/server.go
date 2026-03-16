package server

import (
	"context"
	"net"

	"github.com/ARCoder181105/kvstore/internal/store"
)

type Server struct {
	addr   string
	store  *store.Store
	ln     net.Listener
	ctx    context.Context
	cancel context.CancelFunc
}

func New(addr string, store *store.Store) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		addr:   addr,
		store:  store,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.ln = ln

	go func() {
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				select {
				case <-s.ctx.Done():
					return
				default:
					continue
				}
			}
			go s.handleConn(conn)
		}
	}()
	return nil
}

func (s *Server) Stop() {
	s.cancel()
	if s.ln != nil {
		s.ln.Close()
	}
}
