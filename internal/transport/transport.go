package transport

import "github.com/VABorisov/CryptoMessenger/internal/models"

type Transport interface {
	Send(to string, msg models.Message) error
	ReceiveChannel() <-chan models.IncomingMessage
	ListenPort() int
	Close() error
}
