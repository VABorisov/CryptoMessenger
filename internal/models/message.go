package models

type MessageType string

const (
	MessageTypeRegular      MessageType = "regular"
	MessageTypePublicKey    MessageType = "public_key"
	MessageTypeSymmetricKey MessageType = "symmetric_key"
	MessageTypeKeyAck       MessageType = "key_ack"
	MessagePing             MessageType = "ping"
	MessageTypeFragment                 = "fragment"
)

const (
	MaxUDPSize = 1400
)

type Message struct {
	Type             MessageType `json:"type"`
	Username         string      `json:"username"`
	HasImage         bool        `json:"has_image,omitempty"`
	PublicKeyData    []byte      `json:"public_key_data,omitempty"`
	SymmetricKeyData []byte      `json:"symmetric_key_data,omitempty"`
	EncryptionMode   string      `json:"encryption_mode,omitempty"`
	EncryptedData    []byte      `json:"encrypted_data,omitempty"`
	FragmentID       uint32      `json:"fragment_id,omitempty"`
	FragmentIndex    uint32      `json:"fragment_index,omitempty"`
	FragmentTotal    uint32      `json:"fragment_total,omitempty"`
	CRC              uint32      `json:"crc"`
}

type IncomingMessage struct {
	Msg  Message
	From string
}
