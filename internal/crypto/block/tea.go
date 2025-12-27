package block

import "log/slog"

type TEABlockCipher struct {
	logger *slog.Logger
	key    []byte
}

func NewTEABlockCipher(logger *slog.Logger) BlockCipher {
	return nil
}

func (c *TEABlockCipher) Encrypt() error {
	return nil
}

func (c *TEABlockCipher) Decrypt() error {
	return nil
}

func (c *TEABlockCipher) GetKey() error {
	return nil
}

func (c *TEABlockCipher) generateKey() error {
	return nil
}
