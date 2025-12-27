package udp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/VABorisov/CryptoMessenger/internal/models"
)

const (
	DefaultPort = 12345
	BuffSize    = 65536
)

type UDPHandler struct {
	listenPort int
	conn       *net.UDPConn
	receiveCh  chan models.IncomingMessage
	logger     *slog.Logger
	mu         sync.Mutex
	running    bool
}

func NewUDPHandler(logger *slog.Logger, preferredPort int) (*UDPHandler, error) {
	port := preferredPort
	if port == 0 {
		port = DefaultPort
	}

	h := &UDPHandler{
		logger:    logger,
		receiveCh: make(chan models.IncomingMessage, 100),
	}

	// Слушаем на всех интерфейсах с указанным или динамическим портом
	addr := net.UDPAddr{Port: port, IP: net.IPv4zero}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %w", err)
	}

	// Получаем реальный назначенный порт
	actualPort := conn.LocalAddr().(*net.UDPAddr).Port

	h.conn = conn
	h.listenPort = actualPort
	h.running = true

	go h.receiveLoop()

	logger.Info("UDP handler started", "local_port", actualPort)

	return h, nil
}

func (h *UDPHandler) ListenPort() int {
	return h.listenPort
}

func (h *UDPHandler) ReceiveChannel() <-chan models.IncomingMessage {
	return h.receiveCh
}

func (h *UDPHandler) Send(to string, msg models.Message) error {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return fmt.Errorf("transport stopped")
	}
	h.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	addr, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		return err
	}

	_, err = h.conn.WriteToUDP(data, addr)
	if err != nil {
		return err
	}

	h.logger.Info("UDP message sent", "to", to, "msg", msg)
	return nil
}

func (h *UDPHandler) receiveLoop() {
	buf := make([]byte, BuffSize)

	for {
		h.mu.Lock()
		if !h.running {
			h.mu.Unlock()
			return
		}
		h.mu.Unlock()

		h.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, addr, err := h.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if h.running {
				h.logger.Error("UDP read error", "error", err)
			}
			continue
		}

		var msg models.Message
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			h.logger.Warn("Failed to unmarshal UDP message", "error", err)
			continue
		}

		incoming := models.IncomingMessage{
			Msg:  msg,
			From: addr.String(),
		}

		select {
		case h.receiveCh <- incoming:
		default:
			h.logger.Warn("Receive channel full, dropping message")
		}
	}
}

func (h *UDPHandler) Close() error {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return nil
	}
	h.running = false
	h.mu.Unlock()

	err := h.conn.Close()
	close(h.receiveCh)
	h.logger.Info("UDP transport stopped")
	return err
}
