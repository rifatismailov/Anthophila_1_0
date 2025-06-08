///////////////////////////////////////////////////////////////////////////////
// Package: checkfile
// Клас: EncryptedFileHandler
// Опис: Обробляє зашифровані файли, зберігає їх у буфер та передає їх шлях для надсилання.
///////////////////////////////////////////////////////////////////////////////

package checkfile

import (
	"Anthophila/logging"
	sm "Anthophila/struct_modul"
	"sync"
)

// /////////////////////////////////////////////////////////////////////////////
// EncryptedFileHandler
// Поля:
// - OutputChan: канал для отримання зашифрованих файлів.
// - PendingBuffer: буфер файлів, які ще не були відправлені на сервер.
// - Logger: сервіс для логування подій.
// - FileChan: канал, через який передається шлях файлу у FileSender.
// - Mutex: м’ютекс для синхронізації доступу до PendingBuffer.
// - ctx: канал сигналу завершення виконання горутини.
// - wg: вказівник на sync.WaitGroup, використовується для очікування завершення горутин.
// /////////////////////////////////////////////////////////////////////////////
type EncryptedFileHandler struct {
	OutputChan    <-chan sm.EncryptedFile // Канал отримання зашифрованих файлів
	PendingBuffer *PendingFilesBuffer     // Буфер очікування
	Logger        *logging.LoggerService  // Логер
	FileChan      chan<- string           // Канал для відправки шляху файлу
	Mutex         *sync.Mutex             // М’ютекс для синхронізації буфера
	ctx           <-chan struct{}         // Контекст завершення
	wg            *sync.WaitGroup         // Синхронізація горутин
}

// /////////////////////////////////////////////////////////////////////////////
// NewEncryptedFileHandler
// Опис: Конструктор, створює новий об'єкт EncryptedFileHandler
// /////////////////////////////////////////////////////////////////////////////
func NewEncryptedFileHandler(
	outputChan <-chan sm.EncryptedFile,
	pendingBuffer *PendingFilesBuffer,
	logger *logging.LoggerService,
	fileChan chan<- string,
	mutex *sync.Mutex,
	ctx <-chan struct{},
	wg *sync.WaitGroup,
) *EncryptedFileHandler {
	return &EncryptedFileHandler{
		OutputChan:    outputChan,
		PendingBuffer: pendingBuffer,
		Logger:        logger,
		FileChan:      fileChan,
		Mutex:         mutex,
		ctx:           ctx,
		wg:            wg,
	}
}

// /////////////////////////////////////////////////////////////////////////////
// Start
// Опис: Запускає горутину, яка обробляє вхідні зашифровані файли:
//   - зберігає їх у PendingBuffer
//   - передає шлях у FileSender через FileChan
//
// /////////////////////////////////////////////////////////////////////////////
func (h *EncryptedFileHandler) Start() {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-h.ctx:
				return
			case encryptedFile := <-h.OutputChan:
				h.Mutex.Lock()
				h.PendingBuffer.AddToBuffer(encryptedFile)
				h.Mutex.Unlock()
				h.FileChan <- encryptedFile.EncryptedPath
			}
		}
	}()
}
