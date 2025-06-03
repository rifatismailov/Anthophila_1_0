package management

import (
	"Anthophila/logging"
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
		sendChan: make(chan []byte, 100), // буфер 100 повідомлень
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
			s.logger.Log("Send error", err.Error())
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
