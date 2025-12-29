package udp

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"log/slog"
	"math/rand/v2"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/VABorisov/CryptoMessenger/internal/models"
	"github.com/VABorisov/CryptoMessenger/internal/transport"
)

const (
	DefaultPort     = 12345
	BuffSize        = 65536
	SafePayloadSize = 7000
	MaxUDPSize      = 65507
)

type UDPTransport struct {
	listenPort int
	conn       *net.UDPConn
	receiveCh  chan models.IncomingMessage
	logger     *slog.Logger
	mu         sync.Mutex
	running    bool

	pendingFragments map[string]map[uint32][]byte
	fragmentsMu      sync.Mutex
}

func NewUDPTransport(logger *slog.Logger, preferredPort int) (transport.Transport, error) {
	port := preferredPort
	if port == 0 {
		port = DefaultPort
	}

	t := &UDPTransport{
		logger:           logger,
		receiveCh:        make(chan models.IncomingMessage, 100),
		pendingFragments: make(map[string]map[uint32][]byte),
	}

	addr := net.UDPAddr{Port: port, IP: net.IPv4zero}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		logger.Error("failed to listen UDP port", "error", err)
		return nil, fmt.Errorf("UDP listening error: %w", err)
	}

	conn.SetWriteBuffer(1024 * 1024)
	conn.SetReadBuffer(1024 * 1024)

	actualPort := conn.LocalAddr().(*net.UDPAddr).Port
	t.conn = conn
	t.listenPort = actualPort
	t.running = true

	go t.receiveLoop()

	logger.Info("UDP transport started", "local port", actualPort)
	return t, nil
}

func (t *UDPTransport) ListenPort() int {
	return t.listenPort
}

func (t *UDPTransport) ReceiveChannel() <-chan models.IncomingMessage {
	return t.receiveCh
}

func (t *UDPTransport) Send(to string, msg models.Message) error {
	t.logger.Info("transmitting message via UDP", "to", to)

	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return errors.New("UDP transport stopped")
	}
	t.mu.Unlock()

	msg.CRC = 0
	dataWithoutCRC, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message for CRC: %w", err)
	}
	msg.CRC = crc32.ChecksumIEEE(dataWithoutCRC)

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	addr, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		return fmt.Errorf("resolve address: %w", err)
	}

	if len(data) <= SafePayloadSize+1000 {
		_, err = t.conn.WriteToUDP(data, addr)
		return err
	}

	fragmentID := rand.Uint32()
	totalFragments := (len(data) + SafePayloadSize - 1) / SafePayloadSize

	t.logger.Info("fragmenting large message", "size", len(data), "fragments", totalFragments, "to", to)

	for i := 0; i < totalFragments; i++ {
		start := i * SafePayloadSize
		end := start + SafePayloadSize
		if end > len(data) {
			end = len(data)
		}
		payload := data[start:end]

		fragMsg := models.Message{
			Type:          models.MessageTypeFragment,
			FragmentID:    fragmentID,
			FragmentIndex: uint32(i),
			FragmentTotal: uint32(totalFragments),
			EncryptedData: payload,
		}

		fragMsg.CRC = 0
		noCRC, err := json.Marshal(fragMsg)
		if err != nil {
			return err
		}
		fragMsg.CRC = crc32.ChecksumIEEE(noCRC)

		fragData, err := json.Marshal(fragMsg)
		if err != nil {
			return err
		}

		if len(fragData) > MaxUDPSize {
			t.logger.Error("fragment too large", "size", len(fragData))
			return fmt.Errorf("fragment too large: %d bytes", len(fragData))
		}

		_, err = t.conn.WriteToUDP(fragData, addr)
		if err != nil {
			t.logger.Error("failed to send fragment", "index", i, "error", err)
			return err
		}
	}

	t.logger.Info("large message sent successfully in fragments", "fragments", totalFragments)
	return nil
}

func (t *UDPTransport) receiveLoop() {
	t.logger.Info("starting UDP receiver")
	buf := make([]byte, BuffSize)

	for {
		t.mu.Lock()
		if !t.running {
			t.mu.Unlock()
			return
		}
		t.mu.Unlock()

		t.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, fromAddr, err := t.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if t.running {
				t.logger.Error("UDP read error", "error", err)
			}
			continue
		}

		var msg models.Message
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			t.logger.Error("unmarshal error", "error", err, "from", fromAddr.String())
			continue
		}

		receivedCRC := msg.CRC
		msg.CRC = 0
		noCRCData, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		if crc32.ChecksumIEEE(noCRCData) != receivedCRC {
			t.logger.Warn("CRC mismatch", "from", fromAddr.String())
			continue
		}

		if msg.Type == models.MessageTypeFragment {
			if t.processFragment(fromAddr.String(), msg) {

			}
			continue
		}

		t.sendToChannel(models.IncomingMessage{
			Msg:  msg,
			From: fromAddr.String(),
		})
	}
}

func (t *UDPTransport) processFragment(from string, frag models.Message) bool {
	key := from + "_" + strconv.FormatUint(uint64(frag.FragmentID), 10)

	t.fragmentsMu.Lock()
	defer t.fragmentsMu.Unlock()

	if t.pendingFragments[key] == nil {
		t.pendingFragments[key] = make(map[uint32][]byte)
	}

	t.pendingFragments[key][frag.FragmentIndex] = frag.EncryptedData

	if len(t.pendingFragments[key]) == int(frag.FragmentTotal) {
		var reassembled []byte
		for i := uint32(0); i < frag.FragmentTotal; i++ {
			reassembled = append(reassembled, t.pendingFragments[key][i]...)
		}
		delete(t.pendingFragments, key)

		var fullMsg models.Message
		if err := json.Unmarshal(reassembled, &fullMsg); err != nil {
			t.logger.Error("reassemble unmarshal error", "error", err)
			return false
		}

		t.logger.Info("message reassembled", "size", len(reassembled), "from", from)

		t.sendToChannel(models.IncomingMessage{
			Msg:  fullMsg,
			From: from,
		})
		return true
	}
	return false
}

func (t *UDPTransport) sendToChannel(msg models.IncomingMessage) {
	select {
	case t.receiveCh <- msg:
	default:
		t.logger.Error("receive channel full, dropping message")
	}
}

func (t *UDPTransport) Close() error {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = false
	t.mu.Unlock()

	err := t.conn.Close()
	close(t.receiveCh)

	t.fragmentsMu.Lock()
	t.pendingFragments = make(map[string]map[uint32][]byte)
	t.fragmentsMu.Unlock()

	t.logger.Info("UDP transport stopped")
	return err
}
