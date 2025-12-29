package asymmetric

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const keyPath = "~/.cm/"

type RSAAsymmetricCipher struct {
	logger     *slog.Logger
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	username   string
}

func NewRSAAsymmetricCipher(logger *slog.Logger, username string) (AsymmetricCipher, error) {
	c := &RSAAsymmetricCipher{
		logger:   logger,
		username: username,
	}

	if err := c.loadOrGenerateKeys(); err != nil {
		logger.Error("failed to load or generate RSA keys", "username", username, "error", err)
		return nil, fmt.Errorf("RSA initialization error: %w", err)
	}

	return c, nil
}

func (c *RSAAsymmetricCipher) Encrypt(data []byte) ([]byte, error) {
	c.logger.Info("encrypting data with RSA public key", "username", c.username)
	if c.publicKey == nil {
		err := errors.New("RSA public key not loaded")
		c.logger.Error("failed to encrypt data with RSA public key", "username", c.username, "error", err)
		return nil, fmt.Errorf("RSA data encryption error: %w", err)
	}

	res, err := rsa.EncryptPKCS1v15(rand.Reader, c.publicKey, data)
	if err != nil {
		c.logger.Error("failed to encrypt data with RSA public key", "username", c.username, "error", err)
		return nil, fmt.Errorf("RSA data encryption error: %w", err)
	}

	c.logger.Info("data successfully encrypted with RSA", "username", c.username)

	return res, nil
}

func (c *RSAAsymmetricCipher) EncryptWithPeerKey(peerKey []byte, data []byte) ([]byte, error) {
	block, _ := pem.Decode(peerKey)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("key is not an RSA public key")
	}

	return rsa.EncryptPKCS1v15(rand.Reader, rsaPub, data)
}

func (c *RSAAsymmetricCipher) Decrypt(cipher []byte) ([]byte, error) {
	c.logger.Info("decrypting data with RSA private key", "username", c.username)
	if c.privateKey == nil {
		err := errors.New("RSA private key not loaded")
		c.logger.Error("failed to decrypt data with RSA public key", "username", c.username, "error", err)
		return nil, fmt.Errorf("RSA data decryption error: %w", err)
	}

	res, err := rsa.DecryptPKCS1v15(rand.Reader, c.privateKey, cipher)
	if err != nil {
		c.logger.Error("failed to decrypt data with RSA private key", "username", c.username, "error", err)
		return nil, fmt.Errorf("RSA data decryption error: %w", err)
	}

	c.logger.Info("data successfully decrypted with RSA", "username", c.username)

	return res, nil
}

func (c *RSAAsymmetricCipher) GetPublicKey() ([]byte, error) {
	if c.publicKey == nil {
		err := errors.New("RSA public key not loaded")
		c.logger.Error("failed to obtain RSA public key", "username", c.username, "error", err)
		return nil, fmt.Errorf("obtaining RSA public key error: %w", err)
	}
	return c.encodePublicKey(c.publicKey), nil
}

func (c *RSAAsymmetricCipher) GetPrivateKey() ([]byte, error) {
	if c.privateKey == nil {
		err := errors.New("RSA private key not loaded")
		c.logger.Error("failed to obtain RSA private key", "username", c.username, "error", err)
		return nil, fmt.Errorf("obtaining RSA private key error: %w", err)
	}
	return c.encodePrivateKey(c.privateKey), nil
}

func (c *RSAAsymmetricCipher) SetPublicKey(key []byte) error {
	c.logger.Info("setting RSA public key", "username", c.username)
	block, _ := pem.Decode(key)
	if block == nil {
		err := errors.New("invalid RSA public key error")
		c.logger.Error("failed to decode RSA puclic key", "username", c.username, "error", err)
		return fmt.Errorf("setting RSA public error: %w", err)
	}

	c.logger.Debug("parsing RSA public key", "username", c.username)
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		c.logger.Error("failed to parse RSA public key", "username", c.username, "error", err)
		return fmt.Errorf("setting RSA public error: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		err := errors.New("invalid RSA public key type")
		c.logger.Error("failed to check RSA public key type", "username", c.username, "error", err)
		return fmt.Errorf("loading RSA keys error: %w", err)
	}

	c.publicKey = rsaPub
	return nil
}

func (c *RSAAsymmetricCipher) DeleteGuestKeys() error {
	path := resolvePath(keyPath)

	privPath := filepath.Join(path, c.username+"_private.pem")
	pubPath := filepath.Join(path, c.username+"_public.pem")

	files := []string{privPath, pubPath}

	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			if err := os.Remove(f); err != nil {
				c.logger.Error("failed to delete guest RSA key file", "username", c.username, "error", err)
				return fmt.Errorf("deleting guest RSA keys error: %w", err)
			}
			c.logger.Info("guest key successfully deleted", "username", c.username)
		}
	}

	c.privateKey = nil
	c.publicKey = nil

	return nil
}

func (c *RSAAsymmetricCipher) loadOrGenerateKeys() error {
	c.logger.Info("trying to load RSA keys", "username", c.username)
	path := resolvePath(keyPath)

	if err := os.MkdirAll(path, 0700); err != nil {
		c.logger.Error("failed to create RSA keys path", "username", c.username, "error", err)
		return fmt.Errorf("obtaining RSA keys error: %w", err)
	}

	privPath := filepath.Join(path, c.username+"_private.pem")
	pubPath := filepath.Join(path, c.username+"_public.pem")

	c.logger.Debug("checking if RSA keys exists", "username", c.username)
	if _, err := os.Stat(privPath); err == nil {
		c.logger.Info("RSA keys exists, trying to load", "username", c.username)
		return c.loadKeys(privPath, pubPath)
	}

	c.logger.Warn("RSA keys does not exists, trying to create new", "username", c.username)
	if err := c.generateKeyPair(); err != nil {
		return fmt.Errorf("obtaining RSA keys error: %w", err)
	}

	if err := os.WriteFile(privPath, c.encodePrivateKey(c.privateKey), 0600); err != nil {
		c.logger.Error("failed to write RSA private key", "username", c.username, "error", err)
		return fmt.Errorf("obtaining RSA keys error: %w", err)
	}
	if err := os.WriteFile(pubPath, c.encodePublicKey(c.publicKey), 0644); err != nil {
		c.logger.Error("failed to write RSA public key", "username", c.username, "error", err)
		return fmt.Errorf("obtaining RSA keys error: %w", err)
	}

	c.logger.Info("RSA keys successfully generated", "username", c.username)

	return nil
}

func (c *RSAAsymmetricCipher) loadKeys(privPath, pubPath string) error {
	c.logger.Info("loading RSA keys", "username", c.username)
	privBytes, err := os.ReadFile(privPath)
	if err != nil {
		c.logger.Error("failed to load RSA private key", "username", c.username, "error", err)
		return fmt.Errorf("loading RSA keys error: %w", err)
	}

	c.logger.Debug("decoding RSA private key", "username", c.username)
	block, _ := pem.Decode(privBytes)
	if block == nil {
		err := errors.New("invalid RSA private key ")
		c.logger.Error("failed to decode RSA private key", "username", c.username, "error", err)
		return fmt.Errorf("loading RSA keys error: %w", err)
	}

	c.logger.Debug("parsing RSA private key", "username", c.username)
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		c.logger.Error("failed to parse RSA private key", "username", c.username, "error", err)
		return fmt.Errorf("loading RSA keys error: %w", err)
	}

	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		c.logger.Error("failed to load RSA public key", "username", c.username, "error", err)
		return fmt.Errorf("loading RSA keys error: %w", err)
	}

	c.logger.Debug("decoding RSA public key", "username", c.username)
	blockPub, _ := pem.Decode(pubBytes)
	if blockPub == nil {
		err := errors.New("invalid RSA public key error")
		c.logger.Error("failed to decode RSA public key", "username", c.username, "error", err)
		return fmt.Errorf("loading RSA keys error: %w", err)
	}

	c.logger.Debug("parsing RSA public key", "username", c.username)
	pub, err := x509.ParsePKIXPublicKey(blockPub.Bytes)
	if err != nil {
		c.logger.Error("failed to parse RSA public key", "username", c.username, "error", err)
		return fmt.Errorf("loading RSA keys error: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		err := errors.New("invalid RSA public key type")
		c.logger.Error("failed to check RSA public key type", "username", c.username, "error", err)
		return fmt.Errorf("loading RSA keys error: %w", err)
	}

	c.privateKey = priv
	c.publicKey = rsaPub
	return nil
}

func (c *RSAAsymmetricCipher) generateKeyPair() error {
	c.logger.Info("genereating RSA key pair", "username", c.username)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		c.logger.Error("failed to genrate RSA key pair", "username", c.username, "error", err)
		return err
	}

	c.privateKey = key
	c.publicKey = &key.PublicKey
	return nil
}

func (c *RSAAsymmetricCipher) encodePrivateKey(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func (c *RSAAsymmetricCipher) encodePublicKey(key *rsa.PublicKey) []byte {
	pubBytes, _ := x509.MarshalPKIXPublicKey(key)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
}

func resolvePath(path string) string {
	if path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
