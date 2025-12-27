package stream

type StreamCipher interface {
	Encrypt()
	Decrypt()
	GetKey()
}
