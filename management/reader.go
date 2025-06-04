package management

import (
	"Anthophila/information"
	"Anthophila/logging"

	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// Reader - структура, яка забезпечує обробку отриманих повідомлень через WebSocket.
type Reader struct {
	Ctx           context.Context // для завершення Reader
	Logger        *logging.LoggerService
	Encryptor     *CryptoManager    // Менеджер шифрування для захисту даних
	Terminal      TerminalInterface // Інтерфейс для роботи з терміналом
	CurrentClient string            // Додається, щоб зберігати поточного клієнта для відповіді
	cancelOutput  chan struct{}     // Канал для зупинки старої горутини
	mu            sync.Mutex        // М'ютекс для потокобезпечного доступу
	Sender        *Sender
}

// NewReader створює новий екземпляр `Reader` з параметрами
func NewReader(ctx context.Context, logger *logging.LoggerService, encryptor *CryptoManager, ws *websocket.Conn) *Reader {
	r := &Reader{
		Ctx:          ctx,
		Logger:       logger,
		Encryptor:    encryptor,
		cancelOutput: make(chan struct{}),
		Sender:       NewSender(ws, logger), // ← правильна змінна тут!,
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
	handler := NewTerminalHandler(r.Logger, ws, r.Encryptor, &r.CurrentClient, r.Sender)

	if err := handler.Start(); err != nil {
		r.Logger.LogError("Failed to start terminal: ", err.Error())
		return
	}

	// Призначаємо інтерфейсне посилання
	r.Terminal = handler
}

// ReadMessage читає повідомлення з WebSocket і логгує їх
func (r *Reader) ReadMessage(ws *websocket.Conn) {
	for {
		//_, message, err := ws.ReadMessage()
		_, _, err := ws.ReadMessage()
		if err != nil {
			r.Logger.LogError("Error reading message: %v", err.Error())
		}
		//r.Logger.LogInfo("Received: ", string(message))
	}
}

// ReadMessageCommand читає повідомлення з командою, обробляє її та надсилає відповідь або передає в термінал
func (r *Reader) ReadMessageCommand(wSocket *websocket.Conn) {
	for {
		select {
		case <-r.Ctx.Done(): //знищуємо горутину якщо в нас викликається Done() то ми повертаємо  return нічого тим самим знищуємо горутину
			r.Sender.Close()
			if r.Terminal != nil {
				r.Terminal.Stop()
			}
			return

		default:
			_, message, err := wSocket.ReadMessage()
			if err != nil {
				r.Logger.LogError("Error reading message: ", err.Error())
				return
			}

			// 1. Розшифрування
			decrypted := r.Encryptor.DecryptText(string(message))
			// 2. Перевірка, чи дійсно щось розшифрувалося
			if decrypted == "" {
				r.Logger.LogError("Failed to decrypt message", string(message))
				continue
			}

			// 3. Логування розшифрованого тексту
			var raw map[string]json.RawMessage
			if err := json.Unmarshal([]byte(decrypted), &raw); err != nil {
				r.Logger.LogError("JSON parsing failed", err.Error())
				return
			}

			// 🔍 Визначаємо формат:
			if _, ok := raw["clientInfo"]; ok {
				var reg Registration
				if err := json.Unmarshal([]byte(decrypted), &reg); err != nil {
					r.Logger.LogError("Failed to parse registration", err.Error())
					return
				}

				var info ClientInfo
				if err := json.Unmarshal([]byte(reg.ClientInfo), &info); err != nil {
					r.Logger.LogError("Failed to parse clientInfo", reg.ClientInfo)
					return
				}

				r.Logger.LogInfo(reg.Message, reg.Status)

			} else if _, ok := raw["sClient"]; ok {
				var cmd Message
				if err := json.Unmarshal([]byte(decrypted), &cmd); err != nil {
					r.Logger.LogError("Failed to unmarshal decrypted JSON:", decrypted)
					continue
				}

				r.mu.Lock()
				r.CurrentClient = cmd.SClient
				r.mu.Unlock()
				//fmt.Println("Received text Command: ", string(cmd.Message))

				if cmd.Message == "help" {
					msg := Message{
						SClient: information.NewInfo().GetMACAddress(),
						RClient: cmd.SClient,
						Message: EscapeTerminalMessage("terminal", cmd.Message),
					}

					jsonData, err := json.Marshal(msg)
					if err != nil {
						r.Logger.LogError("Error marshalling JSON:", err.Error())
						continue
					}
					encrypted := r.Encryptor.EncryptText(string(jsonData))

					r.Sender.Send([]byte(encrypted))

				} else {
					if strings.TrimSpace(cmd.Message) == "restart" || strings.TrimSpace(cmd.Message) == "exit_cli" {
						r.initTerminal(wSocket)
						//r.Logger.LogInfo("Terminal restart: ", "restart")
						continue
					}

					if r.Terminal != nil {
						r.Terminal.SendCommand(cmd.Message)
					} else {
						r.Logger.LogInfo("Terminal not initialized command ignored: ", cmd.Message)
					}
				}
			}
		}
	}
}
