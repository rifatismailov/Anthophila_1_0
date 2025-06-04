package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

// Logger — структура лог-повідомлення
type Logger struct {
	MacAddress string    `json:"mac-address"`
	HostName   string    `json:"host-name"`
	Message    string    `json:"message"`
	Event      string    `json:"event,omitempty"`
	Level      string    `json:"level"`      // "info" або "error"
	EventTime  time.Time `json:"event_time"` // Час логування
}

// LoggerService — сервіс логування
type LoggerService struct {
	logAddress string
	logChan    chan Logger
	wg         sync.WaitGroup
	esClient   *elasticsearch.Client

	macAddress string
	hostName   string
}

// NewLoggerService — створення лог-сервісу
func NewLoggerService(macAddress, hostName, logAddress string, esClient *elasticsearch.Client) *LoggerService {
	service := &LoggerService{
		logAddress: logAddress,
		logChan:    make(chan Logger, 100),
		esClient:   esClient,
		macAddress: macAddress,
		hostName:   hostName,
	}
	service.wg.Add(1)
	go service.run()
	return service
}

// run — горутина, що обробляє лог-повідомлення
func (l *LoggerService) run() {
	defer l.wg.Done()
	for log := range l.logChan {
		log.EventTime = time.Now()
		log.HostName = l.hostName
		log.MacAddress = l.macAddress

		if l.logAddress == "console" || l.esClient == nil {
			logJson, _ := json.Marshal(log)
			fmt.Println(string(logJson))
		} else {
			body, err := json.Marshal(log)
			if err != nil {
				fmt.Println("Marshal error:", err)
				continue
			}
			_, err = l.esClient.Index("logs-go-app", bytes.NewReader(body))
			if err != nil {
				fmt.Println("Error sending to Elasticsearch:", err)
			}
		}
	}
}

// LogError — логування помилки
func (l *LoggerService) LogError(message, eventStr string) {
	log := Logger{
		Message: message,
		Event:   eventStr,
		Level:   "error",
	}
	select {
	case l.logChan <- log:
	default:
		fmt.Println("[WARN] Log channel full. Dropping error log:", message)
	}
}

// LogInfo — логування інформаційного повідомлення
func (l *LoggerService) LogInfo(message, eventStr string) {
	log := Logger{
		Message: message,
		Event:   eventStr,
		Level:   "info",
	}
	select {
	case l.logChan <- log:
	default:
		fmt.Println("[WARN] Log channel full. Dropping info log:", message)
	}
}

// Close — завершення лог-сервісу
func (l *LoggerService) Close() {
	close(l.logChan)
	l.wg.Wait()
}
