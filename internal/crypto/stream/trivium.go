package stream

import (
	"crypto/rand"
	"fmt"
	"log/slog"

	"github.com/bmkessler/trivium"
)

type TriviumStreamCipher struct {
	logger *slog.Logger
	key    [10]byte
	nonce  [10]byte
}

func NewTriviumStreamCipher(logger *slog.Logger) StreamCipher {
	c := &TriviumStreamCipher{logger: logger}
	_ = c.generateKey()
	return c
}

func (c *TriviumStreamCipher) Encrypt(data []byte) ([]byte, error) {
	c.logger.Info("encrypting data with Trivium key")
	t := trivium.NewTrivium(c.key, c.nonce)

	keystream := t.NextBytes(uint(len(data)))

	cipher := make([]byte, len(data))
	for i := range data {
		cipher[i] = data[i] ^ keystream[i]
	}

	c.logger.Info("data successfully encrypted with Trivium")
	return cipher, nil
}

func (c *TriviumStreamCipher) Decrypt(cipher []byte) ([]byte, error) {
	c.logger.Info("decrypting data with Trivium key")

	res, err := c.Encrypt(cipher)
	if err != nil {
		c.logger.Error("failed to decrypt data with Trivium key", "error", err)
		return nil, fmt.Errorf("Trivium data decryption error: %w", err)
	}

	c.logger.Info("data successfully decrypted with Trivium")
	return res, nil
}

func (c *TriviumStreamCipher) GetKey() ([]byte, []byte) {
	return c.key[:], c.nonce[:]
}

func (c *TriviumStreamCipher) generateKey() error {
	_, err := rand.Read(c.key[:])
	if err != nil {
		return err
	}
	_, err = rand.Read(c.nonce[:])
	if err != nil {
		return err
	}
	return nil
}
