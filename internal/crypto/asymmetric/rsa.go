package asymmetric

import "log/slog"

const keyPath = "~/.cm/"

type RSAAsymmetricCipher struct {
	Logger     *slog.Logger
	PrivateKey []byte
	PublicKey  []byte
	Username   string
	IsGuest    string
}

func NewRSAAsymmetricCipher(logger *slog.Logger, username string, isGuest bool) *AsymetricCipher {
	return nil
}

func (c *RSAAsymmetricCipher) Encrypt() error {
	return nil
}

func (c *RSAAsymmetricCipher) Decrypt() error {
	return nil
}

func (c *RSAAsymmetricCipher) GetPublicKey() error {
	return nil
}

func (c *RSAAsymmetricCipher) GetPrivteKey() error {
	return nil
}

func (c *RSAAsymmetricCipher) loadOrGenerateKeys() error {
	return nil
}

func (c *RSAAsymmetricCipher) generateKeyPair() error {
	return nil
}

func (c *RSAAsymmetricCipher) generatePublicKeyFromPrivateKey() error {
	return nil
}
