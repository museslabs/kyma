package tui

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
)

type SyncServer struct {
	listener     net.Listener
	port         int
	clients      map[net.Conn]bool
	running      bool
	currentSlide int
}

const startPort = 3000

func NewSyncServer() (*SyncServer, error) {
	maxAttempts := 100

	for port := startPort; port < startPort+maxAttempts; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			// Server not found on this port, try next one
			continue
		}

		server := &SyncServer{
			listener:     listener,
			port:         port,
			clients:      make(map[net.Conn]bool),
			currentSlide: 0,
		}

		return server, nil
	}

	return nil, fmt.Errorf("no available ports in range %d-%d", startPort, startPort+maxAttempts-1)
}

func (s *SyncServer) GetPort() int {
	return s.port
}

func (s *SyncServer) Start() {
	s.running = true

	go s.acceptConnections()
}

func (s *SyncServer) Stop() {
	s.running = false

	if s.listener != nil {
		s.listener.Close()
	}

	for client := range s.clients {
		client.Close()
	}
	s.clients = make(map[net.Conn]bool)

	slog.Info("Sync server stopped")
}

func (s *SyncServer) BroadcastSlideChange(slideNumber int) {
	s.currentSlide = slideNumber

	message := fmt.Sprintf("SLIDE:%d\n", slideNumber)

	for client := range s.clients {
		_, err := client.Write([]byte(message))
		if err != nil {
			delete(s.clients, client)
			client.Close()
		}
	}
}

func (s *SyncServer) acceptConnections() error {
	for {
		running := s.running

		if !running {
			break
		}

		conn, err := s.listener.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept connection: %w", err)
		}

		s.clients[conn] = true

		currentSlide := s.currentSlide

		message := fmt.Sprintf("SLIDE:%d\n", currentSlide)
		_, err = conn.Write([]byte(message))
		if err != nil {
			return fmt.Errorf("failed to send current slide to new client: %w", err)
		}

	}
	return nil
}

type SyncClient struct {
	conn net.Conn
	port int
}

func NewSyncClient() (*SyncClient, error) {
	maxAttempts := 100

	for port := startPort; port < startPort+maxAttempts; port++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}

		client := &SyncClient{
			conn: conn,
			port: port,
		}

		return client, nil
	}

	return nil, fmt.Errorf("no sync server found in port range %d-%d", startPort, startPort+maxAttempts-1)
}

func (c *SyncClient) ListenForSlideChanges(slideChangeChan chan<- int) {
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "SLIDE:") {
			slideNumStr := strings.TrimPrefix(line, "SLIDE:")
			if slideNum, err := strconv.Atoi(slideNumStr); err == nil {
				select {
				case slideChangeChan <- slideNum:
				default:
					// Channel is full, skip this update
				}
			}
		}
	}
}

func (c *SyncClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
