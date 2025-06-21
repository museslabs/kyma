package tui

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
)

type SyncServer struct {
	listener     net.Listener
	port         int
	clients      map[net.Conn]struct{}
	clientsMu    sync.Mutex
	running      bool
	currentSlide int
}

const port = 34622

func NewSyncServer() (*SyncServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port: %w", err)
	}

	server := &SyncServer{
		listener:     listener,
		port:         port,
		clients:      make(map[net.Conn]struct{}),
		currentSlide: 0,
	}

	return server, nil
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

	s.clientsMu.Lock()
	for client := range s.clients {
		client.Close()
	}
	s.clients = make(map[net.Conn]struct{})
	s.clientsMu.Unlock()

	slog.Info("Sync server stopped")
}

func (s *SyncServer) BroadcastSlideChange(slideNumber int) {
	s.currentSlide = slideNumber

	message := fmt.Sprintf("SLIDE:%d\n", slideNumber)

	s.clientsMu.Lock()
	for client := range s.clients {
		_, err := client.Write([]byte(message))
		if err != nil {
			delete(s.clients, client)
			client.Close()
		}
	}
	s.clientsMu.Unlock()
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

		s.clientsMu.Lock()
		s.clients[conn] = struct{}{}
		s.clientsMu.Unlock()

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
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sync server: %w", err)
	}

	client := &SyncClient{
		conn: conn,
		port: port,
	}

	return client, nil
}

func (c *SyncClient) ListenForSlideChanges(slideChangeChan chan<- int) {
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "SLIDE:") {
			slideNumStr := strings.TrimPrefix(line, "SLIDE:")
			if slideNum, err := strconv.Atoi(slideNumStr); err == nil {
				slideChangeChan <- slideNum
			}
		}
	}
}

func (c *SyncClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
