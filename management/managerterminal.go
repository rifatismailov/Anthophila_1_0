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

type TManager struct {
	cmd     *exec.Cmd
	input   chan string
	output  chan string
	wg      *sync.WaitGroup
	pid     int
	mu      sync.Mutex
	stopped bool
}

func NewTerminalManager() *TManager {
	tm := &TManager{
		input:   make(chan string),
		output:  make(chan string),
		wg:      &sync.WaitGroup{},
		cmd:     nil,
		stopped: false,
	}

	if runtime.GOOS == "windows" {
		tm.cmd = exec.Command("cmd.exe")
	} else {
		tm.cmd = exec.Command("bash")
	}

	return tm
}

func (tm *TManager) Start() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.cmd != nil && tm.cmd.Process != nil && tm.cmd.ProcessState == nil {
		tm.output <- "Terminal already running"
		return errors.New("terminal already running")
	}

	if tm.cmd == nil {
		tm.cmd = exec.Command("bash")
	} else {
		tm.cmd = exec.Command(tm.cmd.Path)
	}

	stdin, err := tm.cmd.StdinPipe()
	if err != nil {
		tm.output <- fmt.Sprintf("Error creating stdin pipe: %v", err)
		return err
	}
	stdout, err := tm.cmd.StdoutPipe()
	if err != nil {
		tm.output <- fmt.Sprintf("Error creating stdout pipe: %v", err)
		return err
	}
	stderr, err := tm.cmd.StderrPipe()
	if err != nil {
		tm.output <- fmt.Sprintf("Error creating stderr pipe: %v", err)
		return err
	}

	if err := tm.cmd.Start(); err != nil {
		tm.output <- fmt.Sprintf("Error starting command: %v", err)
		return err
	}

	tm.pid = tm.cmd.Process.Pid
	tm.stopped = false

	tm.wg.Add(1)
	go tm.runTerminal(stdin, stdout, stderr)

	return nil
}

func (tm *TManager) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.cmd == nil || tm.cmd.Process == nil {
		return
	}

	if !tm.stopped {
		tm.stopped = true
		close(tm.input)
	}

	_ = tm.cmd.Process.Kill()
	tm.cmd = nil
	tm.wg.Wait()
}

func (tm *TManager) SendCommand(command string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.stopped {
		tm.output <- "Cannot send command: terminal is stopped"
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tm.output <- fmt.Sprintf("Recovered in SendCommand: %v", r)
		}
	}()

	tm.input <- command
}

func (tm *TManager) GetOutput() <-chan string {
	return tm.output
}

func (tm *TManager) Restart() {
	tm.Stop()
	time.Sleep(1 * time.Second)
	if err := tm.Start(); err != nil {
		tm.output <- fmt.Sprintf("Failed to start terminal: %v", err)
	}
}

func runExpectSudoSu(password string) error {
	expectScript := fmt.Sprintf(
		`#!/usr/bin/expect
	set timeout -1
	spawn sudo su
	expect "Password:"
	send "%s\r"
	interact`, password)

	tmpFile := filepath.Join(os.TempDir(), "sudo_su_script.exp")
	if err := os.WriteFile(tmpFile, []byte(expectScript), 0700); err != nil {
		return err
	}

	cmd := exec.Command("expect", tmpFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (tm *TManager) runTerminal(stdin io.WriteCloser, stdout io.Reader, stderr io.Reader) {
	defer tm.wg.Done()

	go func() {
		defer close(tm.output)
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				tm.output <- fmt.Sprintf("Error reading stdout: %v", err)
				break
			}
			tm.output <- line
		}
	}()

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

	for command := range tm.input {
		if strings.TrimSpace(command) == "exit" {
			stdin.Close()
			return
		}

		if strings.TrimSpace(command) == "stop" {
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(command), "sudo su:") {
			parts := strings.SplitN(command, ":", 2)
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

		if strings.HasPrefix(command, "ping ") && !strings.Contains(command, "-c") {
			parts := strings.Split(command, " ")
			if len(parts) > 1 {
				command = fmt.Sprintf("ping -c 4 %s", strings.Join(parts[1:], " "))
			}
		}

		_, err := io.WriteString(stdin, command+"\n")
		if err != nil {
			tm.output <- fmt.Sprintf("Error writing to stdin: %v", err)
			tm.Restart()
		}
	}

	if tm.cmd != nil {
		if err := tm.cmd.Wait(); err != nil {
			tm.output <- fmt.Sprintf("Error waiting for command: %v", err)
		}
	}
}
