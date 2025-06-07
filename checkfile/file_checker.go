package checkfile

import (
	"Anthophila/information"
	"Anthophila/logging"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileHasher interface {
	CheckAndWriteHash(path, hashFile string) (bool, error)
}

type FileChecker struct {
	File_server         string
	Logger              *logging.LoggerService
	Key                 string
	Directories         []string
	SupportedExtensions []string
	Hour                int8
	Minute              int8
	Info                *information.Info
	Hasher              FileHasher

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	pendingMu sync.Mutex
}

func NewFileChecker(file_server string, logger *logging.LoggerService, key string, directories []string, se []string, h int8, m int8, info *information.Info) *FileChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &FileChecker{
		File_server:         file_server,
		Logger:              logger,
		Key:                 key,
		Directories:         directories,
		SupportedExtensions: se,
		Hour:                h,
		Minute:              m,
		Info:                info,
		ctx:                 ctx,
		cancel:              cancel,
	}
}

func (fc *FileChecker) Start() {
	fc.Logger.LogInfo("FileChecker запуск", "")
	inputEnc := make(chan Verify)
	outputEnc := make(chan EncryptedFile)

	encryptor, err := NewFILEEncryptor([]byte(fc.Key), inputEnc, outputEnc)
	if err != nil {
		fc.Logger.LogError("FILEEncryptor помилка:", err.Error())
		return
	}
	go encryptor.Run()

	vb := &VerifyBuffer{}
	_ = vb.LoadFromFile("verified_files.json")

	pendingBuffer := &PendingFilesBuffer{}
	_ = pendingBuffer.LoadFromFile("pending_files.json")

	fs := NewFileSender("http://" + fc.File_server + "/api/files/upload")
	fs.Start()

	// 🔄 Обробка результатів
	fc.wg.Add(1)
	go func() {
		defer fc.wg.Done()
		for {
			select {
			case <-fc.ctx.Done():
				return
			case result := <-fs.ResultChan:
				if result.Status == "201" {
					fc.Logger.LogInfo("✅ Надіслано", result.Path)
					fc.pendingMu.Lock()
					pendingBuffer.RemoveFromBuffer(result.Path)
					fc.pendingMu.Unlock()
					_ = os.Remove(result.Path)
				} else {
					fc.Logger.LogError("❌ Надсилання помилка", result.Path+" → "+result.Error.Error())
				}
			}
		}
	}()

	// 🔄 Сканування
	fc.wg.Add(1)
	go func() {
		defer fc.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-fc.ctx.Done():
				return
			case <-ticker.C:
				for _, dir := range fc.Directories {
					_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
						if err != nil || info.IsDir() || !isSupportedFileType(path, fc.SupportedExtensions) {
							return nil
						}
						changed, verify, err := vb.SaveToBuffer(path)
						if err != nil {
							fc.Logger.LogError("Помилка буфера", err.Error())
							return nil
						}
						if changed {
							_ = os.Remove(verify.Path + ".enc")
							inputEnc <- verify
						}
						return nil
					})
				}
				fc.pendingMu.Lock()
				_ = vb.SaveToFile("verified_files.json")
				_ = pendingBuffer.SaveToFile("pending_files.json")
				fc.pendingMu.Unlock()
			}
		}
	}()

	// 🔄 Обробка шифрування
	fc.wg.Add(1)
	go func() {
		defer fc.wg.Done()
		for {
			select {
			case <-fc.ctx.Done():
				return
			case encryptedFile := <-outputEnc:
				fc.pendingMu.Lock()
				pendingBuffer.AddToBuffer(encryptedFile)
				fc.pendingMu.Unlock()
				fs.FileChan <- encryptedFile.EncryptedPath
			}
		}
	}()
}

func (fc *FileChecker) Stop() {
	fc.cancel()
	fc.wg.Wait()
	fc.Logger.LogInfo("FileChecker зупинено", "")
}

func isSupportedFileType(file string, supportedExtensions []string) bool {
	for _, ext := range supportedExtensions {
		if strings.HasSuffix(strings.ToLower(file), strings.ToLower(ext)) {
			return true
		}
	}
	return false
}
