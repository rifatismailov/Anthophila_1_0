// stderrreader.go
package management

import (
	"bufio"
	"fmt"
	"io"
)

// StderrReader відповідає за читання помилок з stderr терміналу.
// Він працює в окремій горутині і надсилає кожен зчитаний рядок у канал Output.
type StderrReader struct {
	// Output — канал, через який передаються зчитані рядки або повідомлення про помилки.
	Output chan string
}

// NewStderrReader створює новий екземпляр StderrReader
// і ініціалізує канал Output для подальшої передачі даних.
func NewStderrReader() *StderrReader {
	return &StderrReader{
		Output: make(chan string),
	}
}

// Start запускає горутину для постійного читання stderr.
// Кожен зчитаний рядок передається у канал Output.
func (sr *StderrReader) Start(stderr io.Reader) {
	go func() {
		reader := bufio.NewReader(stderr)
		for {
			// Зчитуємо stderr по рядках
			line, err := reader.ReadString('\n')
			if err != nil {
				// У разі помилки передаємо повідомлення про помилку у канал
				sr.Output <- fmt.Sprintf("Error reading stderr: %v", err)
				break
			}
			// Відправляємо рядок у канал Output
			sr.Output <- line
		}
	}()
}
