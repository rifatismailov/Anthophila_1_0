package management

// Message — загальна структура для обміну повідомленнями між клієнтами через WebSocket.
type Message struct {
	SClient string `json:"sClient"`
	RClient string `json:"rClient"`
	Message string `json:"message"`
}
