package checkfile

import (
	"Anthophila/logging"
	v "Anthophila/struct_modul"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// /////////////////////////////////////////////////////////////////////////////
// Структура: Scanner
// Відповідає за сканування директорій на наявність змінених або нових файлів,
// перевірку їх хешів і надсилання на шифрування.
// /////////////////////////////////////////////////////////////////////////////
type Scanner struct {
	Directories         []string               // Список директорій для сканування
	SupportedExtensions []string               // Підтримувані розширення файлів
	VerifyBuffer        *VerifyBuffer          // Буфер перевірених файлів і хешів
	PendingBuffer       *PendingFilesBuffer    // Буфер файлів, що очікують надсилання
	InputChan           chan<- v.Verify        // Канал для передачі файлів на шифрування
	Logger              *logging.LoggerService // Сервіс логування
	Mutex               *sync.Mutex            // М'ютекс для синхронізації доступу до буферів
	ctx                 <-chan struct{}        // Контекст для завершення роботи горутини
	wg                  *sync.WaitGroup        // Очікування завершення горутин
}

// /////////////////////////////////////////////////////////////////////////////
// Функція: NewScanner
// Конструктор для створення нового екземпляра Scanner
// /////////////////////////////////////////////////////////////////////////////
func NewScanner(
	directories []string,
	supportedExtensions []string,
	verifyBuffer *VerifyBuffer,
	pendingBuffer *PendingFilesBuffer,
	inputChan chan<- v.Verify,
	logger *logging.LoggerService,
	mutex *sync.Mutex,
	ctx <-chan struct{},
	wg *sync.WaitGroup,
) *Scanner {
	return &Scanner{
		Directories:         directories,
		SupportedExtensions: supportedExtensions,
		VerifyBuffer:        verifyBuffer,
		PendingBuffer:       pendingBuffer,
		InputChan:           inputChan,
		Logger:              logger,
		Mutex:               mutex,
		ctx:                 ctx,
		wg:                  wg,
	}
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: Start
// Запускає нескінченний цикл сканування директорій у окремій горутині.
// Знаходить нові або змінені файли, надсилає їх на шифрування,
// зберігає буфери у JSON-файли.
// /////////////////////////////////////////////////////////////////////////////
func (s *Scanner) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx:
				s.Logger.LogInfo("Scanning stopped", "End")
				return
			default:
				s.Logger.LogInfo("🔁 Directory scanning", "Start")
				for _, dir := range s.Directories {
					err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
						if err != nil || info.IsDir() || !isSupportedFileType(path, s.SupportedExtensions) {
							return nil
						}
						changed, verify, err := s.VerifyBuffer.SaveToBuffer(path)
						if err != nil {
							s.Logger.LogError("Buffer error", err.Error())
							return nil
						}
						if changed {
							s.Logger.LogInfo("Modified file found", verify.Path)
							_ = os.Remove(verify.Path + ".enc")
							s.InputChan <- verify
						}
						return nil
					})
					if err != nil {
						s.Logger.LogError("Directory traversal error", err.Error())
					}
				}

				s.Mutex.Lock()
				_ = s.VerifyBuffer.SaveToFile("verified_files.json")
				_ = s.PendingBuffer.SaveToFile("pending_files.json")
				s.Mutex.Unlock()

				time.Sleep(10 * time.Second)
			}
		}
	}()
}

// /////////////////////////////////////////////////////////////////////////////
// Функція: isSupportedFileType
// Перевіряє, чи файл має одне з підтримуваних розширень
// /////////////////////////////////////////////////////////////////////////////
func isSupportedFileType(file string, supportedExtensions []string) bool {
	for _, ext := range supportedExtensions {
		if strings.HasSuffix(strings.ToLower(file), strings.ToLower(ext)) {
			return true
		}
	}
	return false
}
