package block

type BlockCipher interface {
	Encrypt()
	Decrypt()
	GetKey()
}
