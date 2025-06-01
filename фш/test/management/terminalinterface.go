package management

// TerminalInterface визначає загальні методи для роботи з терміналом
type TerminalInterface interface {
	Start() error
	Stop()
	SendCommand(command string)
}
