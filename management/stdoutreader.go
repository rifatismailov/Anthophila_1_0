// stdoutreader.go
package management

import (
	"bufio"
	"fmt"
	"io"
)

// StdoutReader відповідає за читання виводу з stdout терміналу.
// Читає рядки в окремій горутині та надсилає їх у канал Output.
type StdoutReader struct {
	// Output — канал для передачі кожного рядка, зчитаного з stdout.
	// Інші компоненти можуть читати з цього каналу.
	Output chan string
}

// NewStdoutReader створює новий екземпляр StdoutReader
// і ініціалізує канал Output.
func NewStdoutReader() *StdoutReader {
	return &StdoutReader{
		Output: make(chan string),
	}
}

// Start запускає горутину, яка читає stdout постійно
// і передає кожен рядок у канал Output.
func (sr *StdoutReader) Start(stdout io.Reader) {
	go func() {
		defer close(sr.Output) // канал закриється після завершення читання

		reader := bufio.NewReader(stdout)
		for {
			// Читаємо до символу нового рядка
			line, err := reader.ReadString('\n')
			if err != nil {
				// Якщо сталася помилка, повідомляємо в канал
				sr.Output <- fmt.Sprintf("Error reading stdout: %v", err)
				break
			}
			// Надсилаємо прочитаний рядок у канал
			sr.Output <- line
		}
	}()
}
