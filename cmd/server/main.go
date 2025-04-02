package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"github.com/Barrioslopezfd/httpfromtcp/internal/request"
	"github.com/Barrioslopezfd/httpfromtcp/internal/response"
)

type Handler func(w *response.Writer, req *request.Request)

type HandlerError struct {
	Status_code response.Code
	Msg         string
}

type Server struct {
	listening atomic.Bool
	ln        net.Listener
	handler   Handler
}

func Serve(h Handler, port int) (*Server, error) {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create a listener, err=%s", err)
	}

	srv := &Server{
		ln:      ln,
		handler: h,
	}

	go srv.listen()

	return srv, nil
}

func (s *Server) listen() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if s.listening.Load() {
				return
			}
			fmt.Println("listen() error=", err)
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	req, err := request.RequestFromReader(conn)
	if err != nil {
		log.Fatal(err)
	}
	w := &response.Writer{
		Writer: conn,
	}

	s.handler(w, req)
}

func (s *Server) Close() error {
	s.listening.Store(true)
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}
