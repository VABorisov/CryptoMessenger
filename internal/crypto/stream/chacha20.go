package stream

import (
	"crypto/rand"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/chacha20"
)

const (
	NonceSize = chacha20.NonceSizeX
	KeySize   = 32
)

type ChaCha20StreamCipher struct {
	logger *slog.Logger
	key    [KeySize]byte
	nonce  [NonceSize]byte
}

func NewChaCha20StreamCipher(logger *slog.Logger) StreamCipher {
	c := &ChaCha20StreamCipher{logger: logger}
	_ = c.generateKeyAndNonce()
	return c
}

func NewChaCha20StreamCipherWithKey(logger *slog.Logger, key []byte, nonce []byte) StreamCipher {
	if len(key) != KeySize {
		panic("ChaCha20 key must be 32 bytes")
	}
	if len(nonce) != NonceSize {
		panic("ChaCha20 nonce must be 24 bytes (XChaCha20)")
	}

	c := &ChaCha20StreamCipher{logger: logger}
	copy(c.key[:], key)
	copy(c.nonce[:], nonce)
	return c
}

func (c *ChaCha20StreamCipher) Encrypt(data []byte) ([]byte, error) {
	c.logger.Info("encrypting data with ChaCha20", "size", len(data))

	cipher, err := chacha20.NewUnauthenticatedCipher(c.key[:], c.nonce[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20 cipher: %w", err)
	}

	out := make([]byte, len(data))
	cipher.XORKeyStream(out, data)

	c.logger.Info("data successfully encrypted with ChaCha20")
	return out, nil
}

func (c *ChaCha20StreamCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	return c.Encrypt(ciphertext)
}

func (c *ChaCha20StreamCipher) GetKey() ([]byte, []byte) {
	return c.key[:], c.nonce[:]
}

func (c *ChaCha20StreamCipher) generateKeyAndNonce() error {
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
