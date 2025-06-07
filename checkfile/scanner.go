package checkfile

import (
	"Anthophila/logging"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Scanner struct {
	Directories         []string
	SupportedExtensions []string
	VerifyBuffer        *VerifyBuffer
	PendingBuffer       *PendingFilesBuffer
	InputChan           chan<- Verify
	Logger              *logging.LoggerService
	Mutex               *sync.Mutex
	ctx                 <-chan struct{}
	wg                  *sync.WaitGroup
}

func NewScanner(
	directories []string,
	supportedExtensions []string,
	verifyBuffer *VerifyBuffer,
	pendingBuffer *PendingFilesBuffer,
	inputChan chan<- Verify,
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

func (s *Scanner) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.Logger.LogInfo("🔁 Сканування директорій", "Start")

		for {
			select {
			case <-s.ctx:
				s.Logger.LogInfo("⛔ Сканування зупинено", "")
				return
			default:
				s.Logger.LogInfo("🔁 Сканування директорій", "Start")
				for _, dir := range s.Directories {
					err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
						if err != nil || info.IsDir() || !isSupportedFileType(path, s.SupportedExtensions) {
							return nil
						}
						changed, verify, err := s.VerifyBuffer.SaveToBuffer(path)
						if err != nil {
							s.Logger.LogError("Помилка буфера", err.Error())
							return nil
						}
						if changed {
							s.Logger.LogInfo("Знайдено змінений файл", verify.Path)
							_ = os.Remove(verify.Path + ".enc")
							s.InputChan <- verify
						}
						return nil
					})
					if err != nil {
						s.Logger.LogError("Помилка обходу директорії", err.Error())
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

func isSupportedFileType(file string, supportedExtensions []string) bool {
	for _, ext := range supportedExtensions {
		if strings.HasSuffix(strings.ToLower(file), strings.ToLower(ext)) {
			return true
		}
	}
	return false
}
