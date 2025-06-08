///////////////////////////////////////////////////////////////////////////////
// Package: checkfile
// Клас: ResultListener
// Опис: Слухає результати від FileSender. Якщо файл успішно відправлено (код 201),
//       то видаляє файл з PendingBuffer та фізично з диска.
//       Якщо сталася помилка — логувати її.
///////////////////////////////////////////////////////////////////////////////

package checkfile

import (
	"Anthophila/logging"
	r "Anthophila/struct_modul"
	"os"
	"sync"
)

// /////////////////////////////////////////////////////////////////////////////
// ResultListener
// Поля:
// - ResultChan: канал, у який FileSender надсилає результат відправки файлу.
// - PendingBuffer: буфер очікування, з якого видаляються успішно передані файли.
// - Logger: сервіс логування для фіксації помилок та дій.
// - Mutex: м’ютекс для безпечної синхронізації доступу до PendingBuffer.
// - ctx: сигнал для завершення горутини (наприклад, при зупинці програми).
// - wg: синхронізація завершення горутин (WaitGroup).
// /////////////////////////////////////////////////////////////////////////////
type ResultListener struct {
	ResultChan    <-chan r.Result        // Канал результатів відправлення файлів
	PendingBuffer *PendingFilesBuffer    // Буфер файлів, які ще не відправлені
	Logger        *logging.LoggerService // Сервіс логування
	Mutex         *sync.Mutex            // М’ютекс для захисту буфера
	ctx           <-chan struct{}        // Канал завершення
	wg            *sync.WaitGroup        // Група очікування завершення
}

// /////////////////////////////////////////////////////////////////////////////
// NewResultListener
// Конструктор для створення нового ResultListener.
//
// Параметри:
// - resultChan: канал результатів від FileSender
// - pendingBuffer: буфер з файлами для відправки
// - logger: сервіс для логування
// - mutex: м’ютекс для захисту буфера
// - ctx: канал для завершення
// - wg: WaitGroup для синхронізації
// /////////////////////////////////////////////////////////////////////////////
func NewResultListener(
	resultChan <-chan r.Result,
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

// /////////////////////////////////////////////////////////////////////////////
// Start
// Запускає горутину, яка постійно слухає канал результатів.
// Якщо отримано статус "201" — файл було успішно відправлено:
// - видаляємо файл з PendingBuffer
// - видаляємо фізично з файлової системи
//
// Якщо статус інший — лог помилки.
// /////////////////////////////////////////////////////////////////////////////
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
					// Блокуємо буфер перед модифікацією
					r.Mutex.Lock()
					r.PendingBuffer.RemoveFromBuffer(result.Path)
					r.Mutex.Unlock()

					// Видаляємо фізично файл
					_ = os.Remove(result.Path)
				} else {
					// Лог помилки
					r.Logger.LogError("Помилка відправлення файлу", result.Error.Error())
				}
			}
		}
	}()
}
