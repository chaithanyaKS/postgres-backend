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
func sendCommandComplete(conn net.Conn, commandTag string) {
	// 'C' + length + commandTag
	message := make([]byte, 5+len(commandTag)+1)
	message[0] = 'C' // CommandComplete identifier

	// Message length (including itself)
	binary.BigEndian.PutUint32(message[1:], uint32(4+len(commandTag)+1))

	// Copy the commandTag string (e.g., "SET") + null terminator
	copy(message[5:], commandTag)
	message[5+len(commandTag)] = 0

	// Send message
	_, err := conn.Write(message)
	if err != nil {
		slog.Error("Failed to send CommandComplete", "err", err)
	}
	slog.Info("Sent CommandComplete", "command", commandTag)
}

func (s *Server) handleQuery(conn net.Conn) error {
	slog.Info("Inside handleQuery, waiting for message")

	for {
		buf := make([]byte, 1024)
		reader := bufio.NewReaderSize(conn, 8)

		n, err := reader.Read(buf)
		slog.Info("Bytes read", "len", n)
		slog.Info("Bytes read", "data", buf[:n])

		sendCommandComplete(conn, "SET")

		// Send ReadyForQuery ('Z')
		conn.Write([]byte{'Z', 0, 0, 0, 5, 'I'})
		if err != nil {
			slog.Error("Error while reading data", "err", err)
			return err
		}
	}
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

func sendParameterStatus(conn net.Conn, key, value string) {
	message := make([]byte, 5+len(key)+1+len(value)+1) // S + length + key + 0 + value + 0
	message[0] = 'S'                                   // Message type 'S'

	// Message length (big-endian)
	binary.BigEndian.PutUint32(message[1:], uint32(len(message)))

	// Copy key=value into message
	copy(message[5:], key)
	message[5+len(key)] = 0
	copy(message[6+len(key):], value)
	message[len(message)-1] = 0

	slog.Info("Parameter Status", "message", message)

	// Send message
	conn.Write(message)
}

func sendBackendKeyData(conn net.Conn, processID, secretKey int32) error {
	// Create the message buffer (1 byte for 'K' + 4 bytes for length + 4 bytes PID + 4 bytes SecretKey)
	message := make([]byte, 13)
	message[0] = 'K' // Message type 'K'

	// Set message length (12 bytes: 4-byte PID + 4-byte secretKey + 4-byte length field itself)
	binary.BigEndian.PutUint32(message[1:], uint32(12))

	// Set process ID (PID)
	binary.BigEndian.PutUint32(message[5:], uint32(processID))

	// Set secret key
	binary.BigEndian.PutUint32(message[9:], uint32(secretKey))

	// Send the message
	_, err := conn.Write(message)
	if err != nil {
		slog.Error("Failed to send BackendKeyData", "err", err)
		return err
	}

	slog.Info("Sent BackendKeyData", "PID", processID, "SecretKey", secretKey)
	return nil
}

func (s *Server) initializeSession(conn net.Conn) error {

	sendParameterStatus(conn, "server_version", "15.0")
	sendParameterStatus(conn, "client_encoding", "UTF8")
	sendParameterStatus(conn, "server_encoding", "UTF8")
	sendParameterStatus(conn, "is_superuser", "false")
	sendParameterStatus(conn, "session_authorization", "postgres")

	slog.Info("ParameterStatus Sent")
	processID := int32(12345) // Fake Process ID
	secretKey := int32(67890)

	sendBackendKeyData(conn, processID, secretKey)

	slog.Info("BackendKeyData Sent")

	resBuf := new(bytes.Buffer)
	resBuf.WriteByte('Z')
	binary.Write(resBuf, binary.BigEndian, uint32(5))
	resBuf.WriteByte('I')
	conn.Write(resBuf.Bytes())

	slog.Info("ReadyForQuery Sent")

	return nil
}
