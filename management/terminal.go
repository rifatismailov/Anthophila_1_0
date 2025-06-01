package management

import (
	"Anthophila/information"
	"Anthophila/logging"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

type TerminalHandler struct {
	LogStatus     bool            //статус логування
	LogAddress    string          //адреса для логування
	WSocket       *websocket.Conn // WebSocket-зʼєднання для обміну повідомленнями
	Term          *TManager       // Менеджер терміналу для обробки команд
	CurrentClient *string         // Поточний клієнт, який надсилає команди
	Cancel        chan struct{}   // Канал для зупинки горутини
	Encryptor     *CryptoManager  // Менеджер шифрування для захисту даних
}

func NewTerminalHandler(logStatus bool, logAddress string, ws *websocket.Conn, encryptor *CryptoManager, client *string) *TerminalHandler {
	return &TerminalHandler{
		LogStatus:     logStatus,
		LogAddress:    logAddress,
		WSocket:       ws,
		Encryptor:     encryptor,
		CurrentClient: client,
		Cancel:        make(chan struct{}),
	}
}

func (t *TerminalHandler) Start() error {
	t.Term = NewTerminalManager()
	if err := t.Term.Start(); err != nil {
		return err
	}
	go t.listen()
	return nil
}

func (t *TerminalHandler) Stop() {
	if t.Term != nil {
		t.Term.Stop()
	}
	close(t.Cancel)
}

// SendCommand implements TerminalInterface.
func (t *TerminalHandler) SendCommand(command string) {
	if t.Term != nil {
		t.Term.SendCommand(command)
	}
}

func (t *TerminalHandler) listen() {
	for {
		select {
		case <-t.Cancel:
			return
		case line := <-t.Term.GetOutput():
			fmt.Printf("User who send command " + (*t.CurrentClient) + "\n")
			msg := Message{
				SClient: information.NewInfo().GetMACAddress(),
				RClient: *t.CurrentClient,
				Message: "{\"terminal\":\"" + strings.Trim(line, "\n") + "\"}",
			}
			jsonData, _ := json.Marshal(msg)
			encrypted := t.Encryptor.EncryptText(string(jsonData))
			if err := NewSender().sendMessageWith(t.LogAddress, t.WSocket, []byte(encrypted)); err != nil {
				if t.LogStatus {
					logging.Log(t.LogAddress, "Error sending encrypted message:", err.Error())
				}
			}
		}
	}
}
