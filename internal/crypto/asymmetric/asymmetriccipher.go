package asymmetric

type AsymetricCipher interface {
	Encrypt()
	Decrypt()
	GetPublicKey()
	GetPrivateKey()
}
