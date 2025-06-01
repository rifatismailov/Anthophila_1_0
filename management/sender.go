package management

import (
	"Anthophila/logging"

	"github.com/gorilla/websocket"
)

// Sender структура для відправки повідомлень через WebSocket
type Sender struct{}

// NewSender створює новий екземпляр Sender
func NewSender() *Sender {
	return &Sender{}
}

// sendMessageWith надсилає повідомлення через WebSocket і логує помилки, якщо вони виникають
func (*Sender) sendMessageWith(logAddress string, wSocket *websocket.Conn, text []byte) error {
	err := wSocket.WriteMessage(websocket.TextMessage, text)
	if err != nil {
		logging.Log(logAddress, "Error sending message:", err.Error())
		return err
	}
	return nil
}
