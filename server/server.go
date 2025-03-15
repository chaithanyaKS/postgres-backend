package server

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"net"
)

type Server struct {
	Port int
	Host string
}

func New(host string, port int) *Server {
	return &Server{
		Port: port,
		Host: host,
	}
}

func (s *Server) getAddress() string {
	address := fmt.Sprintf("%s:%d", s.Host, s.Port)
	return address
}

func (s *Server) ListenAndServe() {
	listener, err := net.Listen("tcp", s.getAddress())
	if err != nil {
		log.Fatal("Something went wrong when listening on ", "port", s.Port, "err", err)
	}

	slog.Info("Listening on ", "port", s.Port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("something went wrong when accepting connection", "err", err)
		}

		go s.handleConnections(conn)
	}
}

func (s *Server) handleConnections(conn net.Conn) {
	slog.Info("Connected to ", "address", conn.RemoteAddr())
	defer slog.Info("Closing Connection to ", "address", conn.RemoteAddr())
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		data := scanner.Bytes()
		slog.Info("received message", "data", data)
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Something went wrong", "err", err)
	}
}
