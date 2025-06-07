package checkfile

import (
	"Anthophila/logging"
	"os"
	"sync"
)

type ResultListener struct {
	ResultChan    <-chan Result
	PendingBuffer *PendingFilesBuffer
	Logger        *logging.LoggerService
	Mutex         *sync.Mutex
	ctx           <-chan struct{}
	wg            *sync.WaitGroup
}

func NewResultListener(
	resultChan <-chan Result,
	pendingBuffer *PendingFilesBuffer,
	logger *logging.LoggerService,
	mutex *sync.Mutex,
	ctx <-chan struct{},
	wg *sync.WaitGroup,
) *ResultListener {
	return &ResultListener{
		ResultChan:    resultChan,
		PendingBuffer: pendingBuffer,
		Logger:        logger,
		Mutex:         mutex,
		ctx:           ctx,
		wg:            wg,
	}
}

func (r *ResultListener) Start() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-r.ctx:
				return
			case result := <-r.ResultChan:
				if result.Status == "201" {
					r.Mutex.Lock()
					r.PendingBuffer.RemoveFromBuffer(result.Path)
					r.Mutex.Unlock()
					_ = os.Remove(result.Path)
				} else {
					r.Logger.LogError("Помилка відправлення файлу", result.Error.Error())
				}
			}
		}
	}()
}
