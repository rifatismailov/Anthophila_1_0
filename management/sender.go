package management

import (
	"Anthophila/logging"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

// Sender асинхронно надсилає повідомлення через WebSocket
type Sender struct {
	sendChan chan []byte
	ws       *websocket.Conn
	logger   *logging.LoggerService
	wg       sync.WaitGroup
}

// NewSender створює новий асинхронний Sender
func NewSender(ws *websocket.Conn, logger *logging.LoggerService) *Sender {
	s := &Sender{
		sendChan: make(chan []byte, 1000), // буфер 100 повідомлень
		ws:       ws,
		logger:   logger,
	}
	s.wg.Add(1)
	go s.run()
	return s
}

// run слухає канал і надсилає повідомлення
func (s *Sender) run() {
	defer s.wg.Done()
	for msg := range s.sendChan {
		err := s.ws.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			//s.logger.Log("Send error", err.Error())
			//fmt.Println("Size message", len(msg))
			//fmt.Println("Send error", err.Error())

			fmt.Println("Send error:", err)

			// Спробуємо перепідключитись
			newWS, reconnectErr := reconnectWebSocket()
			if reconnectErr != nil {
				fmt.Println("Failed to reconnect WebSocket:", reconnectErr)
				continue // або break, якщо критично
			}

			fmt.Println("WebSocket перепідключено.")
			s.ws = newWS

			// Повторна відправка останнього повідомлення
			retryErr := s.ws.WriteMessage(websocket.TextMessage, msg)
			if retryErr != nil {
				fmt.Println("Повторна відправка не вдалася:", retryErr)
			}
		}
	}
}

// Send надсилає повідомлення через канал
func (s *Sender) Send(msg []byte) {
	select {
	case s.sendChan <- msg:
	default:
		// якщо канал заповнений
		s.logger.Log("[WARN] Send channel full. Dropping message", string(msg))
	}
}

// Close закриває канал і чекає завершення обробки
func (s *Sender) Close() {
	close(s.sendChan)
	s.wg.Wait()
}

func reconnectWebSocket() (*websocket.Conn, error) {
	// Залежить від твого клієнта: адреса, TLS, хедери
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
