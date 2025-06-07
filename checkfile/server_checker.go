// server_checker.go

package checkfile

import (
	"time"
)

type ServerChecker struct {
	PendingBuffer *PendingFilesBuffer
	ServerAddress string
	CheckInterval time.Duration
}

func NewServerChecker(buffer *PendingFilesBuffer, serverAddress string, interval time.Duration) *ServerChecker {
	return &ServerChecker{
		PendingBuffer: buffer,
		ServerAddress: serverAddress,
		CheckInterval: interval,
	}
}

func (sc *ServerChecker) Start() {
	for {
		if sc.isServerAvailable() {
			sc.processBuffer()
		}
		time.Sleep(sc.CheckInterval)
	}
}

func (sc *ServerChecker) isServerAvailable() bool {
	// Логіка перевірки доступності сервера
	// Наприклад, можна виконати HTTP-запит до сервера
	return true // Поки що повертаємо true для прикладу
}

func (sc *ServerChecker) processBuffer() {
	for path, file := range sc.PendingBuffer.buffer {
		if sendFile(file) {
			sc.PendingBuffer.RemoveFromBuffer(path)
		}
	}
}
