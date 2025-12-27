package models

type Message struct {
	Username string `json:"username"`
	Text     string `json:"text,omitempty"`
	HasImage bool   `json:"has_image,omitempty"`
}

type IncomingMessage struct {
	Msg  Message
	From string
}
