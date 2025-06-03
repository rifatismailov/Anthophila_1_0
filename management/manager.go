package management

import (
	"Anthophila/information"
	"Anthophila/logging"
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// Константа інтервалу перепідключення
const reconnectInterval = 5 * time.Second

// Manager представляє менеджер WebSocket-зʼєднання
type Manager struct {
	Logger     *logging.LoggerService // додано
	ServerAddr string
	Key        string
	ctx        context.CancelFunc // для завершення Reader

}

// NewManager — конструктор (фабрика) для створення нового менеджера
func NewManager(logger *logging.LoggerService, serverAddr, key string) *Manager {
	return &Manager{
		Logger:     logger,
		ServerAddr: serverAddr,
		Key:        key,
	}
}

// Start ініціалізує WebSocket-зʼєднання і запускає логіку обміну повідомленнями
func (m *Manager) Start() {
	for {
		if err := m.run(); err != nil {
			m.Logger.Log("Connection error: %v. Retrying in %v...", err.Error())
		}
		time.Sleep(reconnectInterval)
	}
}

func (m *Manager) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = cancel // зберігаємо cancel для завершення Reader

	cryptoManager, err := NewCryptoManager(m.Logger, m.Key)
	if err != nil || cryptoManager == nil {
		return fmt.Errorf("failed to init CryptoManager: %v", err)
	}

	nickname := information.NewInfo().InfoJson()
	ws, _, err := websocket.DefaultDialer.Dial(m.ServerAddr, nil)
	if err != nil {
		cancel() // скасовуємо контекст, якщо не вдалося підключитись

		return fmt.Errorf("failed to connect: %w", err)
	}
	defer ws.Close()

	encryptName := cryptoManager.EncryptText(nickname)
	if encryptName == "" {
		cancel() // скасовуємо контекст, якщо не вдалося підключитись
		err := fmt.Errorf("failed to encrypt nickname")
		m.Logger.Log("Crypto error", err.Error())
		return err
	}

	if err := ws.WriteMessage(websocket.TextMessage, []byte("nick:"+encryptName)); err != nil {
		cancel() // скасовуємо контекст, якщо не вдалося підключитись
		m.Logger.Log("WebSocket send error", err.Error())
		return err
	}

	reader := NewReader(ctx, m.Logger, cryptoManager, ws)
	go reader.ReadMessageCommand(ws)

	// Пінг-сервер кожні N секунд
	ticker := time.NewTicker(reconnectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ws.WriteMessage(websocket.TextMessage, []byte("Ping")); err != nil {
				cancel() // скасовуємо контекст, якщо не вдалося підключитись
				m.Logger.Log("ping failed", err.Error())
				return err
			}
		case <-ctx.Done():
			return nil // якщо контекст скасовано, виходимо

		}
	}
}
