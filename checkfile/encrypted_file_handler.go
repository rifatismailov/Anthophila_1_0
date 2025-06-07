package checkfile

import (
	"Anthophila/logging"
	"sync"
)

type EncryptedFileHandler struct {
	OutputChan    <-chan EncryptedFile
	PendingBuffer *PendingFilesBuffer
	Logger        *logging.LoggerService
	FileChan      chan<- string
	Mutex         *sync.Mutex
	ctx           <-chan struct{}
	wg            *sync.WaitGroup
}

func NewEncryptedFileHandler(
	outputChan <-chan EncryptedFile,
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
