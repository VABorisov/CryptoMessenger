package asymmetric

type AsymmetricCipher interface {
	Encrypt(data []byte) ([]byte, error)
	EncryptWithPeerKey(peerKey []byte, data []byte) ([]byte, error)
	Decrypt(cipher []byte) ([]byte, error)
	DeleteGuestKeys() error
	GetPublicKey() ([]byte, error)
	GetPrivateKey() ([]byte, error)
	SetPublicKey(key []byte) error
}
