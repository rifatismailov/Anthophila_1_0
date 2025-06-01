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

// Log надсилає повідомлення до лог-сервера або виводить у консоль.
// Параметри:
// - logAddress: адреса сервера логування або "console"
// - message: текст повідомлення
// - err: повідомлення про помилку (може бути пустим)
func Log(logAddress string, message string, err string) {
	var wg sync.WaitGroup
	done := make(chan struct{})

	logger := Logger{
		Message: message,
		Error:   err,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Тут можна реалізувати відправку на сервер, зараз просто друк
		if logAddress == "console" {
			// Вивід у консоль у форматі JSON
			logJson, _ := json.Marshal(logger)
			fmt.Println(string(logJson))
		} else {
			// Тут ти можеш реалізувати відправку logger на сервер logAddress
			// sendToServer(logAddress, logger)
			fmt.Printf("Send to log server %s: %s - %s\n", logAddress, message, err)
		}
		close(done)
	}()

	wg.Wait()
}
