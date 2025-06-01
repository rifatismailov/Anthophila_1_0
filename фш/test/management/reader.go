package management

import (
	"Anthophila/information"
	"Anthophila/logging"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// Reader - структура, яка забезпечує обробку отриманих повідомлень через WebSocket.
type Reader struct {
	LogStatus     bool              // Статус логування``
	LogAddress    string            // Адреса для логування
	Encryptor     *CryptoManager    // Менеджер шифрування для захисту даних
	Terminal      TerminalInterface // Інтерфейс для роботи з терміналом
	CurrentClient string            // Додається, щоб зберігати поточного клієнта для відповіді
	cancelOutput  chan struct{}     // Канал для зупинки старої горутини
	mu            sync.Mutex        // М'ютекс для потокобезпечного доступу
}

// NewReader створює новий екземпляр `Reader` з параметрами
func NewReader(logStatus bool, logAddress string, encryptor *CryptoManager, ws *websocket.Conn) *Reader {
	r := &Reader{
		LogStatus:    logStatus,
		LogAddress:   logAddress,
		Encryptor:    encryptor,
		cancelOutput: make(chan struct{}),
	}
	r.initTerminal(ws)
	return r
}

// initTerminal запускає новий термінал і запускає горутину, яка читає вихід з нього
func (r *Reader) initTerminal(ws *websocket.Conn) {
	// Якщо попередній термінал існує — зупиняємо
	if r.Terminal != nil {
		r.Terminal.Stop()
	}

	// Ініціалізація нового TerminalHandler
	handler := NewTerminalHandler(r.LogStatus, r.LogAddress, ws, r.Encryptor, &r.CurrentClient)

	if err := handler.Start(); err != nil {
		if r.LogStatus {
			logging.Log(r.LogAddress, "Failed to start terminal: ", err.Error())
		}
		return
	}

	// Призначаємо інтерфейсне посилання
	r.Terminal = handler
}

// ReadMessage читає повідомлення з WebSocket і логгує їх
func (r *Reader) ReadMessage(ws *websocket.Conn) {
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			logging.Log(r.LogAddress, "Error reading message: %v", err.Error())
		}
		logging.Log(r.LogAddress, "Received: ", string(message))
	}
}

// ReadMessageCommand читає повідомлення з командою, обробляє її та надсилає відповідь або передає в термінал
func (r *Reader) ReadMessageCommand(wSocket *websocket.Conn) {
	for {
		_, message, err := wSocket.ReadMessage()
		if err != nil {
			if r.LogStatus {
				logging.Log(r.LogAddress, "Error reading message: ", err.Error())
			}
			return
		}
		// 1. Розшифрування
		decrypted := r.Encryptor.DecryptText(string(message))

		// 2. Перевірка, чи дійсно щось розшифрувалося
		if decrypted == "" {
			if r.LogStatus {
				logging.Log(r.LogAddress, "Failed to decrypt message", string(message))
			}
			continue
		}

		// 3. Логування розшифрованого тексту
		logging.Log(r.LogAddress, "Received decrypted message: ", decrypted)

		// 4. Парсинг JSON
		var cmd Message
		if err := json.Unmarshal([]byte(decrypted), &cmd); err != nil {
			if r.LogStatus {
				logging.Log(r.LogAddress, "Failed to unmarshal decrypted JSON:", decrypted)
			}
			continue
		}

		r.mu.Lock()
		r.CurrentClient = cmd.SClient
		r.mu.Unlock()
		logging.Log(r.LogAddress, "Received text Command: ", string(cmd.Message))

		if cmd.Message == "help" {
			fmt.Println("Available commands: help, restart, exit, terminal")
			msg := Message{
				SClient: information.NewInfo().GetMACAddress(),
				RClient: cmd.SClient,
				Message: "{\"terminal\":\"" + cmd.Message + "\"}",
			}

			jsonData, err := json.Marshal(msg)
			if err != nil {
				if r.LogStatus {
					logging.Log(r.LogAddress, "Error marshalling JSON:", err.Error())
				}
				continue
			}
			encrypted := r.Encryptor.EncryptText(string(jsonData))

			if err := NewSender().sendMessageWith(r.LogAddress, wSocket, []byte(encrypted)); err != nil {
				if r.LogStatus {
					logging.Log(r.LogAddress, "Error sending encrypted message:", err.Error())
				}
			}

		} else {
			if strings.TrimSpace(cmd.Message) == "restart" || strings.TrimSpace(cmd.Message) == "exit" {
				r.initTerminal(wSocket)
				continue
			}

			if r.Terminal != nil {
				r.Terminal.SendCommand(cmd.Message)
			} else if r.LogStatus {
				logging.Log(r.LogAddress, "Terminal not initialized command ignored: ", cmd.Message)
			}

		}
	}
}
