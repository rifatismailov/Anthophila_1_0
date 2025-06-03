package logging

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Logger представляє структуру лог-повідомлення.
type Logger struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// LoggerService керує логуванням через буферизований канал.
type LoggerService struct {
	logAddress string
	logChan    chan Logger
	wg         sync.WaitGroup
}

// NewLoggerService створює новий сервіс логування
func NewLoggerService(logAddress string) *LoggerService {
	service := &LoggerService{
		logAddress: logAddress,
		logChan:    make(chan Logger, 100), // буфер на 100 повідомлень
	}
	service.wg.Add(1)
	go service.run()
	return service
}

// run запускає горутину, яка обробляє лог-повідомлення з каналу
func (l *LoggerService) run() {
	defer l.wg.Done()
	for log := range l.logChan {
		if l.logAddress == "console" {
			logJson, _ := json.Marshal(log)
			fmt.Println(string(logJson))
		} else {
			fmt.Printf("Send to log server %s: %s - %s\n", l.logAddress, log.Message, log.Error)
			// Тут може бути логіка надсилання на сервер
		}
	}
}

// Log надсилає лог у канал
func (l *LoggerService) Log(message string, err string) {
	select {
	case l.logChan <- Logger{Message: message, Error: err}:
	default:
		// Канал заповнений — можна проігнорувати або зробити обробку
		fmt.Println("[WARN] Log channel full. Dropping log:", message)
	}
}

// Close завершує лог-сервіс і чекає, поки всі логи будуть оброблені
func (l *LoggerService) Close() {
	close(l.logChan)
	l.wg.Wait()
}
