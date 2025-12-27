package block

type BlockCipher interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(cipher []byte) ([]byte, error)
	GetKey() []byte
}
