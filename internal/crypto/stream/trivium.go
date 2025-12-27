package stream

import "log/slog"

type TriviumStreamCipher struct {
	logger *slog.Logger
	key    string
}

func NewTriviumStreamCipher(logger *slog.Logger) StreamCipher {
	return nil
}

func (c *TriviumStreamCipher) Encrypt() error {
	return nil
}

func (c *TriviumStreamCipher) Decrypt() error {
	return nil
}

func (c *TriviumStreamCipher) GetKey() error {
	return nil
}

func (c *TriviumStreamCipher) generateKey() error {
	return nil
}
