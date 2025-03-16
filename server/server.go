package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"log/slog"
	"net"
)

type Server struct {
	Port  int
	Host  string
	Extra map[string]any
}

func New(host string, port int) *Server {
	return &Server{
		Port:  port,
		Host:  host,
		Extra: map[string]any{},
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
	err := s.startUp(conn)
	if err != nil {
		slog.Error("Error while start up", "err", err)
		return
	}

	err = s.handleQuery(conn)
	if err != nil {
		slog.Error("Error while handling query", "err", err)
		return
	}
}

func (s *Server) handleQuery(conn net.Conn) error {

	buf := make([]byte, 1024)
	reader := bufio.NewReaderSize(conn, 8)

	n, err := reader.Read(buf)
	slog.Info("Bytes read", "len", n)
	if err != nil {
		slog.Error("Error while reading data", "err", err)
		return err
	}
	// scanner := bufio.NewScanner(conn)
	// for scanner.Scan() {
	// 	data := scanner.Bytes()
	//
	// 	slog.Info("Received query", "query", data)
	// 	conn.Write([]byte("ReadyForQuery"))
	// 	slog.Info("ReadyForQuery Sent inside handleQuery method")
	// }
	return nil
}

func (s *Server) startUp(conn net.Conn) error {
	err := s.performAuthentication(conn)
	if err != nil {
		return err
	}
	s.initializeSession(conn)
	return nil
}

func (s *Server) performAuthentication(conn net.Conn) error {
	buf := make([]byte, 1024)
	reader := bufio.NewReaderSize(conn, 5)
	n, err := reader.Read(buf)
	slog.Info("Bytes read", "len", n)
	if err != nil {
		slog.Error("Error while reading data", "err", err)
		return err
	}
	contentLength := binary.BigEndian.Uint32(buf[:4])
	protocolVersion := binary.BigEndian.Uint32(buf[4:8])

	slog.Info("Received message", "Content Length", contentLength)
	slog.Info("Received message", "Protocol Version", protocolVersion)
	slog.Info("Received message", "Protocol Version", buf[4:8])

	content := buf[8:n]
	slog.Info("Received message", "Content", content)
	slog.Info("Received message", "Content", buf[:n])

	pairs := bytes.Split(content, []byte("\x00"))

	for i := 0; i < len(pairs)-1; i += 2 {
		key, value := string(pairs[i]), string(pairs[i+1])
		s.Extra[key] = value
	}

	slog.Info("Added Extras", "extra", s.Extra)

	resBuf := new(bytes.Buffer)
	resBuf.WriteByte('R')
	// Message length (8 bytes total)
	binary.Write(resBuf, binary.BigEndian, int32(8))

	// Authentication type (0 = AuthenticationOk)
	binary.Write(resBuf, binary.BigEndian, int32(0))

	conn.Write(resBuf.Bytes())

	slog.Info("Authentication Ok sent")
	return nil
}

func (s *Server) initializeSession(conn net.Conn) error {
	resBuf := new(bytes.Buffer)
	resBuf.WriteByte('Z')
	binary.Write(resBuf, binary.BigEndian, int32(5))
	conn.Write(resBuf.Bytes())
	slog.Info("ReadyForQuery Sent")

	return nil
}
