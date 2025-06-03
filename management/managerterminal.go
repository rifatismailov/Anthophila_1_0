package management

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// TManager - керує запуском терміналу, обробкою вводу/виводу, та його зупинкою.
type TManager struct {
	cmd     *exec.Cmd       // об'єкт exec.Cmd для запуску процесу терміналу
	input   chan string     // канал для надсилання команд у stdin
	output  chan string     // канал для читання stdout/stderr
	wg      *sync.WaitGroup // синхронізація завершення горутин
	pid     int             // PID процесу терміналу
	mu      sync.Mutex      // м'ютекс для потокобезпечного доступу
	stopped bool            // прапорець, чи термінал зупинено
}

// NewTerminalManager створює екземпляр TManager з базовими налаштуваннями
func NewTerminalManager() *TManager {
	tm := &TManager{
		input:   make(chan string),
		output:  make(chan string),
		wg:      &sync.WaitGroup{},
		cmd:     nil,
		stopped: false,
	}

	// Визначає команду терміналу залежно від ОС
	if runtime.GOOS == "windows" {
		tm.cmd = exec.Command("cmd.exe")
	} else {
		tm.cmd = exec.Command("bash")
	}
	return tm
}

// Start запускає процес терміналу, створює пайпи, та стартує горутину для обробки
func (tm *TManager) Start() error {
	tm.mu.Lock()         // Блокуємо доступ до змінних, щоб уникнути race condition
	defer tm.mu.Unlock() // Розблокуємо після завершення функції

	// Якщо термінал вже запущено (процес існує, але ще не завершився)
	if tm.cmd != nil && tm.cmd.Process != nil && tm.cmd.ProcessState == nil {
		tm.output <- "Terminal already running"       // Повідомляємо через output
		return errors.New("terminal already running") // Повертаємо помилку
	}

	// Якщо термінал ще не ініціалізовано — створюємо нову bash-команду
	if tm.cmd == nil {
		if runtime.GOOS == "windows" {
			tm.cmd = exec.Command("cmd.exe")
		} else {
			tm.cmd = exec.Command("bash")
		}
	} else {
		tm.cmd = exec.Command(tm.cmd.Path)
	}

	// Отримуємо stdin пайп (для передачі команд у термінал)
	stdin, err := tm.cmd.StdinPipe()
	if err != nil {
		tm.output <- fmt.Sprintf("Error creating stdin pipe: %v", err)
		return err
	}

	// Отримуємо stdout пайп (для читання виводу терміналу)
	stdout, err := tm.cmd.StdoutPipe()
	if err != nil {
		tm.output <- fmt.Sprintf("Error creating stdout pipe: %v", err)
		return err
	}

	// Отримуємо stderr пайп (для читання помилок з терміналу)
	stderr, err := tm.cmd.StderrPipe()
	if err != nil {
		tm.output <- fmt.Sprintf("Error creating stderr pipe: %v", err)
		return err
	}

	// Запускаємо процес терміналу
	if err := tm.cmd.Start(); err != nil {
		tm.output <- fmt.Sprintf("Error starting command: %v", err)
		return err
	}

	tm.pid = tm.cmd.Process.Pid // Зберігаємо PID процесу
	tm.stopped = false          // Позначаємо, що термінал активний

	tm.wg.Add(1)                             // Додаємо 1 до групи очікування для runTerminal
	go tm.runTerminal(stdin, stdout, stderr) // Запускаємо горутину для обробки терміналу

	return nil // Все добре — повертаємо nil
}

// Stop безпечно зупиняє термінал, завершує горутину та закриває ресурси
func (tm *TManager) Stop() {
	tm.mu.Lock()         // Блокуємо доступ до полів, бо будемо змінювати стан
	defer tm.mu.Unlock() // Розблокуємо після завершення функції

	// Якщо термінал ще не запущено або вже зупинено — нічого не робимо
	if tm.cmd == nil || tm.cmd.Process == nil {
		return
	}

	// Якщо термінал ще не зупинений
	if !tm.stopped {
		tm.stopped = true // Позначаємо, що зупинено
		close(tm.input)   // Закриваємо канал input, щоб зупинити runTerminal()
	}

	_ = tm.cmd.Process.Kill() // Насильно зупиняємо процес терміналу
	tm.cmd = nil              // Очищаємо посилання на процес

	tm.wg.Wait() // Очікуємо завершення runTerminal() перед повним виходом
}

// SendCommand надсилає команду у термінал, якщо він активний
func (tm *TManager) SendCommand(command string) {
	tm.mu.Lock()         // Захист від одночасного доступу (конкурентності)
	defer tm.mu.Unlock() // Розблокування після завершення

	// Якщо термінал уже зупинено — повідомляємо у канал output
	if tm.stopped {
		tm.output <- "Cannot send command: terminal is stopped"
		return
	}

	// Захист від panic, якщо канал вже закритий або команда некоректна
	defer func() {
		if r := recover(); r != nil {
			tm.output <- fmt.Sprintf("Recovered in SendCommand: %v", r)
		}
	}()

	// Надсилаємо команду у термінал (канал input слухає runTerminal)
	tm.input <- command
}

// GetOutput повертає канал для читання виводу з терміналу
func (tm *TManager) GetOutput() <-chan string {
	return tm.output // Канал, в який runTerminal пише всі stdout/stderr повідомлення
}

// Restart викликає Stop і Start для перезапуску терміналу
func (tm *TManager) Restart() {
	tm.Stop()
	time.Sleep(1 * time.Second)
	if err := tm.Start(); err != nil {
		tm.output <- fmt.Sprintf("Failed to start terminal: %v", err)
	}
}

// runExpectSudoSu запускає expect-скрипт для автоматичного входу під root через sudo su.
// Приймає пароль як аргумент і повертає помилку у випадку невдачі.
func runExpectSudoSu(password string) error {
	// Формуємо expect-скрипт із вбудованим паролем користувача.
	expectScript := fmt.Sprintf(
		`#!/usr/bin/expect
	set timeout -1                    ;# Очікувати без тайм-ауту
	spawn sudo su                     ;# Запускаємо команду "sudo su"
	expect "Password:"                ;# Чекаємо на появу запиту пароля
	send "%s\r"                       ;# Вводимо пароль + carriage return
	interact                          ;# Передаємо керування користувачу (інтерактивна сесія)`,
		password,
	)

	// Шлях до тимчасового файлу для скрипта (автоматично у /tmp/)
	tmpFile := filepath.Join(os.TempDir(), "sudo_su_script.exp")

	// Записуємо expect-скрипт у тимчасовий файл
	if err := os.WriteFile(tmpFile, []byte(expectScript), 0700); err != nil {
		return err // Якщо не вдалося записати — повертаємо помилку
	}

	// Формуємо команду для запуску expect з нашим скриптом
	cmd := exec.Command("expect", tmpFile)

	// Привʼязуємо stdin, stdout та stderr до поточних,
	// щоб користувач міг взаємодіяти з процесом (як у терміналі)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Запускаємо expect-скрипт
	return cmd.Run()
}

// runTerminal читає stdout/stderr від процесу терміналу та обробляє команди з каналу input.
// stdin — канал запису в термінал (введення користувача)
// stdout, stderr — вихідні потоки з терміналу (відповіді)
func (tm *TManager) runTerminal(stdin io.WriteCloser, stdout io.Reader, stderr io.Reader) {
	// Після завершення методу повідомляємо wg, що горутина завершена
	defer tm.wg.Done()

	// Горутина для читання стандартного виводу (stdout) процесу
	go func() {
		defer close(tm.output) // Закриваємо канал output, коли закінчимо читання
		reader := bufio.NewReader(stdout)
		for {
			// Читаємо рядок до символу нового рядка
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Процес завершився — виходимо з циклу
					break
				}
				// Помилка читання stdout — відправляємо її у канал output
				tm.output <- fmt.Sprintf("Error reading stdout: %v", err)
				break
			}
			// Надсилаємо прочитаний рядок у output-канал
			tm.output <- line
		}
	}()

	// Горутина для читання стандартного потоку помилок (stderr)
	go func() {
		reader := bufio.NewReader(stderr)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				tm.output <- fmt.Sprintf("Error reading stderr: %v", err)
				break
			}
			tm.output <- line
		}
	}()

	// Основний цикл — читає команди з каналу input
	for command := range tm.input {
		trimmed := strings.TrimSpace(command)

		// Команда "exit" — закриваємо stdin і виходимо
		if trimmed == "exit" {
			stdin.Close()
			return
		}

		// Команда "stop" — нічого не робимо (пропускаємо)
		if trimmed == "stop" {
			continue
		}

		// Обробка спеціальної команди sudo su:<пароль>
		if strings.HasPrefix(trimmed, "sudo su:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				password := strings.TrimSpace(parts[1])
				if err := runExpectSudoSu(password); err != nil {
					tm.output <- fmt.Sprintf("Expect sudo su failed: %v", err)
				} else {
					tm.output <- "Sudo su command executed successfully"
				}
			} else {
				tm.output <- "Invalid sudo su command format"
			}
			continue
		}

		// Якщо команда — ping без параметру `-c`, додаємо обмеження
		if strings.HasPrefix(trimmed, "ping ") && !strings.Contains(trimmed, "-c") {
			parts := strings.Split(trimmed, " ")
			if len(parts) > 1 {
				command = fmt.Sprintf("ping -c 4 %s", strings.Join(parts[1:], " "))
			}
		}

		// Надсилаємо команду в stdin термінала
		_, err := io.WriteString(stdin, command+"\n")
		if err != nil {
			// У разі помилки повідомляємо користувача та намагаємося перезапустити термінал
			tm.output <- fmt.Sprintf("Error writing to stdin: %v", err)
			tm.Restart()
		}
	}

	// Очікуємо завершення процесу після закриття каналу input
	if tm.cmd != nil {
		if err := tm.cmd.Wait(); err != nil {
			tm.output <- fmt.Sprintf("Error waiting for command: %v", err)
		}
	}
}
