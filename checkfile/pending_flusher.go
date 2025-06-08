///////////////////////////////////////////////////////////////////////////////
// Package: checkfile
// Клас: PendingFlusher
// Опис:
//   Клас відповідає за надсилання зашифрованих файлів із буфера
//   (PendingFilesBuffer), коли сервер доступний. Перевіряє доступність
//   сервера за допомогою HTTP-запиту до /ping. Якщо сервер доступний —
//   надсилає файли у FileSender.
//
//   Запускається у фоновій горутині і завершується, коли context закривається.
///////////////////////////////////////////////////////////////////////////////

package checkfile

import (
	"Anthophila/logging"
	sm "Anthophila/struct_modul"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// /////////////////////////////////////////////////////////////////////////////
// Структура: PendingFlusher
//
// Поля:
// - ServerURL: повна адреса сервера, включно з портом, без "/ping"
// - PendingBuf: буфер файлів, які ще не були відправлені
// - FileChan: канал, у який передаються шляхи файлів для FileSender
// - Logger: сервіс для логування
// - Mutex: використовується для безпечного доступу до буфера в багатьох потоках
// - ContextDone: сигнал від context.Context про завершення (зупинка горутини)
// - WaitGroup: дозволяє дочекатися завершення цієї горутини
// /////////////////////////////////////////////////////////////////////////////
type PendingFlusher struct {
	ServerURL   string                 // URL до сервера без "/ping"
	PendingBuf  *PendingFilesBuffer    // Буфер зашифрованих файлів для надсилання
	FileChan    chan<- string          // Канал для передачі файлів до FileSender
	Logger      *logging.LoggerService // Сервіс логування
	Mutex       *sync.Mutex            // Мʼютекс для захисту буфера
	ContextDone <-chan struct{}        // Канал завершення (від context)
	WaitGroup   *sync.WaitGroup        // Синхронізація горутин
}

// /////////////////////////////////////////////////////////////////////////////
// Функція: NewPendingFlusher
// Створює і повертає новий об'єкт PendingFlusher.
//
// Параметри:
// - serverURL: адреса сервера, наприклад "http://192.168.1.10:8020"
// - pb: вказівник на буфер з файлами
// - fileChan: канал, через який передаються файли для надсилання
// - logger: сервіс логування
// - mutex: мʼютекс для синхронізації доступу до буфера
// - ctxDone: канал завершення (зазвичай ctx.Done())
// - wg: вказівник на загальний WaitGroup
// /////////////////////////////////////////////////////////////////////////////
func NewPendingFlusher(
	serverURL string,
	pb *PendingFilesBuffer,
	fileChan chan<- string,
	logger *logging.LoggerService,
	mutex *sync.Mutex,
	ctxDone <-chan struct{},
	wg *sync.WaitGroup,
) *PendingFlusher {
	return &PendingFlusher{
		ServerURL:   serverURL,
		PendingBuf:  pb,
		FileChan:    fileChan,
		Logger:      logger,
		Mutex:       mutex,
		ContextDone: ctxDone,
		WaitGroup:   wg,
	}
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: Start
// Запускає горутину, яка кожні 15 секунд перевіряє:
// 1. Чи є файли у буфері.
// 2. Чи сервер доступний (HTTP GET на /ping).
// Якщо так — надсилає файли з буфера в FileSender через FileChan.
// Завершується, коли ContextDone закриється.
// /////////////////////////////////////////////////////////////////////////////
func (pf *PendingFlusher) Start() {
	pf.WaitGroup.Add(1)
	go func() {
		defer pf.WaitGroup.Done()

		pf.Logger.LogInfo("🌐 Running Pending Flusher", "Start Pending")

		for {
			select {
			case <-pf.ContextDone:
				pf.Logger.LogInfo("🌐 Pending Flusher Completion", "Complet Pending")
				return

			default:
				// Отримуємо список файлів з буфера (з блокуванням)
				pf.Mutex.Lock()
				files := pf.PendingBuf.GetAllFiles()
				pf.Mutex.Unlock()

				if len(files) > 0 {
					// Перевіряємо доступність сервера
					resp, err := http.Get(pf.ServerURL + "/ping")
					if err == nil {
						defer resp.Body.Close() // закриваємо відповідь
					}

					if err == nil && resp.StatusCode == 200 {
						// Копіюємо список файлів, щоб уникнути блокування під час надсилання
						pf.Mutex.Lock()
						pendingFiles := make([]sm.EncryptedFile, len(files))
						copy(pendingFiles, files)
						pf.Mutex.Unlock()

						// Відправляємо файли один за одним
						for _, file := range pendingFiles {
							pf.Logger.LogInfo("➡️ Sending from buffer to FileSender", file.EncryptedPath)
							pf.FileChan <- file.EncryptedPath
						}

					} else {
						// Лог помилки
						if err != nil {
							pf.Logger.LogError("🌐 Server unavailable", err.Error())
						} else {
							pf.Logger.LogError("🌐 Server not responding", "error code "+strconv.Itoa(resp.StatusCode))
						}
					}

					time.Sleep(15 * time.Second) // Затримка між перевірками

				} else {
					time.Sleep(5 * time.Second) // Менший інтервал, якщо файлів немає
				}
			}
		}
	}()
}
