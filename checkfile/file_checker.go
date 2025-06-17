package checkfile

import (
	"Anthophila/information"
	"Anthophila/logging"
	sm "Anthophila/struct_modul"

	"context"
	"sync"
)

// FileHasher - інтерфейс для перевірки та запису хешів файлів.
// Метод CheckAndWriteHash(path, hashFile) повертає true, якщо файл змінено (новий хеш), інакше false.
type FileHasher interface {
	CheckAndWriteHash(path, hashFile string) (bool, error)
}

// FileChecker - головна структура для керування процесом перевірки та надсилання файлів.
// Відповідає за:
// - Сканування директорій на наявність нових/змінених файлів.
// - Шифрування знайдених файлів.
// - Надсилання зашифрованих файлів на сервер.
// - Логування подій.
// - Обробку результатів відправки.
type FileChecker struct {
	File_server         string                 // Адреса сервера, куди надсилатимуться зашифровані файли (наприклад, 192.168.0.10:8020)
	Logger              *logging.LoggerService // Сервіс логування подій (інформаційних, помилок тощо)
	Key                 string                 // Ключ шифрування (повинен бути 32-байтний для AES-256)
	Directories         []string               // Список директорій, які потрібно сканувати
	SupportedExtensions []string               // Дозволені типи файлів за розширенням (наприклад, .doc, .pdf)
	Hour                int8                   // Час запуску (опціонально, наразі не використовується)
	Minute              int8                   // Хвилина запуску (опціонально, наразі не використовується)
	Info                *information.Info      // Інформація про клієнта (hostname, ip, mac тощо)
	Hasher              FileHasher             // Інтерфейс для перевірки хешу файлів (для визначення змін)

	ctx       context.Context    // Контекст завершення роботи (для управління горутинами)
	cancel    context.CancelFunc // Функція для скасування контексту (зупинка всіх процесів)
	wg        sync.WaitGroup     // Група для синхронного очікування завершення всіх горутин
	pendingMu sync.Mutex         // М'ютекс для потокобезпечного доступу до буферів (Pending, Verify)
}

// NewFileChecker - конструктор FileChecker. Ініціалізує контекст завершення та встановлює всі залежності.
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

// Start - основний метод запуску всіх підпроцесів: сканування, шифрування, відправка та обробка результатів.
func (fc *FileChecker) Start() {
	fc.Logger.LogInfo("🚀 Запуск FileChecker", "")

	input_to_enc_file, output_enc_file, vb, pb, encryptor, sender, err := fc.initComponents()
	if err != nil {
		fc.Logger.LogError("❌ Encryptor init error", err.Error())
		return
	}

	fc.startEncryptor(encryptor)
	fc.startSender(sender)
	fc.startResultHandler(sender, pb)
	fc.startEncryptedHandler(output_enc_file, pb, sender)
	fc.startPendingFileFlusher(pb, sender.Iutput_to_send_enc_file)
	fc.startScanner(vb, pb, input_to_enc_file)
}

// Stop - завершує всі процеси, викликаючи cancel() і очікуючи завершення горутин через WaitGroup.
func (fc *FileChecker) Stop() {
	fc.cancel()
	fc.wg.Wait()
	fc.Logger.LogInfo("🛑 FileChecker зупинено", "")
}

// initComponents - створює всі потрібні компоненти: буфери, канали, енкриптор, відправник.
func (fc *FileChecker) initComponents() (
	chan sm.Verify,
	chan sm.EncryptedFile,
	*VerifyBuffer,
	*PendingFilesBuffer,
	*FILEEncryptor,
	*FileSender,
	error,
) {
	input_to_enc_file := make(chan sm.Verify, 100)
	output_enc_file := make(chan sm.EncryptedFile, 100)

	vb := &VerifyBuffer{}
	_ = vb.LoadFromFile("verified_files.json")

	pb := &PendingFilesBuffer{}
	_ = pb.LoadFromFile("pending_files.json")

	encryptor, err := NewFILEEncryptor([]byte(fc.Key), input_to_enc_file, output_enc_file)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	sender := NewFileSender("http://" + fc.File_server + "/api/files/upload")

	return input_to_enc_file, output_enc_file, vb, pb, encryptor, sender, nil
}

// startEncryptor - запускає процес шифрування (енкриптор).
func (fc *FileChecker) startEncryptor(encryptor *FILEEncryptor) {
	fc.Logger.LogInfo("▶️ Запуск Encryptor", "")
	encryptor.Start(&fc.wg)
}

// startSender - запускає процес надсилання зашифрованих файлів.
func (fc *FileChecker) startSender(sender *FileSender) {
	fc.Logger.LogInfo("▶️ Запуск Sender", "")
	sender.Start()
}

// startResultHandler - запускає слухача результатів відправки (видаляє успішно відправлені файли з буфера).
func (fc *FileChecker) startResultHandler(sender *FileSender, pb *PendingFilesBuffer) {
	handler := NewResultListener(sender.ResultChan, pb, fc.Logger, &fc.pendingMu, fc.ctx.Done(), &fc.wg)
	handler.Start()
}

// startEncryptedHandler - слухає канал вихідних зашифрованих файлів і передає їх у буфер та відправку.
func (fc *FileChecker) startEncryptedHandler(input_enc_file <-chan sm.EncryptedFile, pb *PendingFilesBuffer, sender *FileSender) {
	h := NewEncryptedFileHandler(input_enc_file, pb, fc.Logger, sender.Iutput_to_send_enc_file, &fc.pendingMu, fc.ctx.Done(), &fc.wg)
	h.Start()
}

// startScanner - запускає сканер директорій, який перевіряє нові або змінені файли.
func (fc *FileChecker) startScanner(vb *VerifyBuffer, pb *PendingFilesBuffer, input_to_enc_file chan<- sm.Verify) {
	scanner := NewScanner(fc.Directories, fc.SupportedExtensions, vb, pb, input_to_enc_file, fc.Logger, &fc.pendingMu, fc.ctx.Done(), &fc.wg)
	scanner.Start()
}

// startPendingFileFlusher - запускає механізм перевірки доступності сервера та надсилання файлів із буфера.
func (fc *FileChecker) startPendingFileFlusher(pb *PendingFilesBuffer, fileChan chan<- string) {
	flusher := NewPendingFlusher("http://"+fc.File_server+"/api/files", pb, fileChan, fc.Logger, &fc.pendingMu, fc.ctx.Done(), &fc.wg)
	flusher.Start()
}
