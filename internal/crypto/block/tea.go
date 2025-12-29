package block

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
)

type TEABlockCipher struct {
	logger *slog.Logger
	key    [16]byte
}

const delta = 0x9E3779B9

func NewTEABlockCipher(logger *slog.Logger) BlockCipher {
	c := &TEABlockCipher{logger: logger}
	_ = c.generateKey()
	return c
}

func NewTEABlockCipherWithKey(logger *slog.Logger, key []byte) BlockCipher {
	c := &TEABlockCipher{
		logger: logger,
	}
	copy(c.key[:], key)
	return c
}

func (c *TEABlockCipher) Encrypt(data []byte) ([]byte, error) {
	c.logger.Info("encrypting data with TEA key")

	// Добавляем padding
	padded := addPKCS7Padding(data, 8)

	out := make([]byte, len(padded))
	for i := 0; i < len(padded); i += 8 {
		v0 := bytesToUint32(padded[i : i+4])
		v1 := bytesToUint32(padded[i+4 : i+8])
		e0, e1 := teaEncryptBlock(v0, v1, c.key)
		copy(out[i:i+4], uint32ToBytes(e0))
		copy(out[i+4:i+8], uint32ToBytes(e1))
	}

	c.logger.Info("data successfully encrypted with TEA")
	return out, nil
}

func (c *TEABlockCipher) Decrypt(cipher []byte) ([]byte, error) {
	c.logger.Info("decrypting data with TEA key")

	if len(cipher)%8 != 0 {
		err := errors.New("ciphertext length must be multiple of 8 bytes")
		c.logger.Error("failed to decrypt data with TEA key", "error", err)
		return nil, fmt.Errorf("TEA data decryption error: %w", err)
	}

	out := make([]byte, len(cipher))
	for i := 0; i < len(cipher); i += 8 {
		v0 := bytesToUint32(cipher[i : i+4])
		v1 := bytesToUint32(cipher[i+4 : i+8])
		d0, d1 := teaDecryptBlock(v0, v1, c.key)
		copy(out[i:i+4], uint32ToBytes(d0))
		copy(out[i+4:i+8], uint32ToBytes(d1))
	}

	// Убираем padding
	plaintext, err := removePKCS7Padding(out)
	if err != nil {
		c.logger.Error("invalid padding during TEA decryption", "error", err)
		return nil, fmt.Errorf("TEA padding error: %w", err)
	}

	c.logger.Info("data successfully decrypted with TEA")
	return plaintext, nil
}

func (c *TEABlockCipher) GetKey() []byte {
	return c.key[:]
}

func (c *TEABlockCipher) generateKey() error {
	_, err := rand.Read(c.key[:])
	return err
}

func bytesToUint32(b []byte) uint32 {
	return uint32(b[0])<<24 |
		uint32(b[1])<<16 |
		uint32(b[2])<<8 |
		uint32(b[3])
}

func uint32ToBytes(v uint32) []byte {
	return []byte{
		byte(v >> 24),
		byte(v >> 16),
		byte(v >> 8),
		byte(v),
	}
}

func teaEncryptBlock(v0, v1 uint32, key [16]byte) (uint32, uint32) {
	k := []uint32{
		bytesToUint32(key[0:4]),
		bytesToUint32(key[4:8]),
		bytesToUint32(key[8:12]),
		bytesToUint32(key[12:16]),
	}

	var sum uint32 = 0
	d := uint32(delta)

	for i := 0; i < 32; i++ {
		sum += d
		v0 += ((v1 << 4) + k[0]) ^ (v1 + sum) ^ ((v1 >> 5) + k[1])
		v1 += ((v0 << 4) + k[2]) ^ (v0 + sum) ^ ((v0 >> 5) + k[3])
	}
	return v0, v1
}

func teaDecryptBlock(v0, v1 uint32, key [16]byte) (uint32, uint32) {
	k := []uint32{
		bytesToUint32(key[0:4]),
		bytesToUint32(key[4:8]),
		bytesToUint32(key[8:12]),
		bytesToUint32(key[12:16]),
	}

	d := uint32(delta)
	sum := d * 32

	for i := 0; i < 32; i++ {
		v1 -= ((v0 << 4) + k[2]) ^ (v0 + sum) ^ ((v0 >> 5) + k[3])
		v0 -= ((v1 << 4) + k[0]) ^ (v1 + sum) ^ ((v1 >> 5) + k[1])
		sum -= d
	}
	return v0, v1
}

func addPKCS7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

func removePKCS7Padding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("invalid padding: data is empty")
	}
	padding := int(data[len(data)-1])
	if padding < 1 || padding > 16 {
		return nil, errors.New("invalid padding value")
	}
	for _, b := range data[len(data)-padding:] {
		if b != byte(padding) {
			return nil, errors.New("invalid padding bytes")
		}
	}
	return data[:len(data)-padding], nil
}
