package management

import (
	"encoding/json"
	"log"
)

// ToJSON приймає довільну структуру або map і повертає JSON-рядок.
// У разі помилки повертає порожній рядок і лог повідомлення.
func ToJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[ToJSON Error] %v\n", err)
		return ""
	}
	return string(jsonData)
}

// EscapeTerminalMessage формує JSON-повідомлення з полем terminal
// і гарантує правильне екранування спецсимволів.
func EscapeTerminalMessage(body_command string, command string) string {
	payload := map[string]string{
		body_command: command,
	}
	return ToJSON(payload)
}
