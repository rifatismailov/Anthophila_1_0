package checkfile

import (
	"Anthophila/information"
	"Anthophila/logging"
	"context"
	"sync"
)

// FileHasher - інтерфейс для перевірки та запису хешів файлів
// (використовується для визначення, чи змінився файл).
type FileHasher interface {
	CheckAndWriteHash(path, hashFile string) (bool, error)
}

// FileChecker - головний координаційний клас для:
// - сканування директорій
// - перевірки змін
// - шифрування
// - відправки файлів на сервер
// - логування подій
type FileChecker struct {
	File_server         string                 // Адреса сервера для відправлення файлів
	Logger              *logging.LoggerService // Сервіс логування
	Key                 string                 // Ключ AES-256 для шифрування
	Directories         []string               // Директорії для сканування
	SupportedExtensions []string               // Підтримувані типи файлів
	Hour                int8                   // Час запуску (не використовується в поточному коді)
	Minute              int8                   // Хвилина запуску
	Info                *information.Info      // Додаткова інформація про клієнта
	Hasher              FileHasher             // Компонент для перевірки хешів

	ctx       context.Context    // Контекст для завершення всіх процесів
	cancel    context.CancelFunc // Функція для завершення всіх горутин
	wg        sync.WaitGroup     // Група для синхронного завершення всіх запущених процесів
	pendingMu sync.Mutex         // М'ютекс для синхронного доступу до буферів
}

// NewFileChecker - конструктор FileChecker
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
	fc.Logger.LogInfo("🚀 Запуск FileChecker", "")

	inputEnc, outputEnc, vb, pb, encryptor, sender, err := fc.initComponents()
	if err != nil {
		fc.Logger.LogError("❌ Encryptor init error", err.Error())
		return
	}

	fc.startEncryptor(encryptor)
	fc.startSender(sender)
	fc.startResultHandler(sender, pb)
	fc.startEncryptedHandler(outputEnc, pb, sender)
	fc.startScanner(vb, pb, inputEnc)
}

func (fc *FileChecker) Stop() {
	fc.cancel()
	fc.wg.Wait()
	fc.Logger.LogInfo("🛑 FileChecker зупинено", "")
}

// --- ПІДМЕТОДИ ---

func (fc *FileChecker) initComponents() (
	chan Verify,
	chan EncryptedFile,
	*VerifyBuffer,
	*PendingFilesBuffer,
	*FILEEncryptor,
	*FileSender,
	error,
) {
	inputEnc := make(chan Verify, 100)
	outputEnc := make(chan EncryptedFile, 100)

	vb := &VerifyBuffer{}
	_ = vb.LoadFromFile("verified_files.json")

	pb := &PendingFilesBuffer{}
	_ = pb.LoadFromFile("pending_files.json")

	encryptor, err := NewFILEEncryptor([]byte(fc.Key), inputEnc, outputEnc)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	sender := NewFileSender("http://" + fc.File_server + "/api/files/upload")

	return inputEnc, outputEnc, vb, pb, encryptor, sender, nil
}

func (fc *FileChecker) startEncryptor(encryptor *FILEEncryptor) {
	fc.Logger.LogInfo("▶️ Запуск Encryptor", "")
	encryptor.Start(&fc.wg)
}

func (fc *FileChecker) startSender(sender *FileSender) {
	fc.Logger.LogInfo("▶️ Запуск Sender", "")
	sender.Start()
}

func (fc *FileChecker) startResultHandler(sender *FileSender, pb *PendingFilesBuffer) {
	handler := NewResultListener(sender.ResultChan, pb, fc.Logger, &fc.pendingMu, fc.ctx.Done(), &fc.wg)
	handler.Start()
}

func (fc *FileChecker) startEncryptedHandler(output <-chan EncryptedFile, pb *PendingFilesBuffer, sender *FileSender) {
	h := NewEncryptedFileHandler(output, pb, fc.Logger, sender.FileChan, &fc.pendingMu, fc.ctx.Done(), &fc.wg)
	h.Start()
}

func (fc *FileChecker) startScanner(vb *VerifyBuffer, pb *PendingFilesBuffer, input chan<- Verify) {
	scanner := NewScanner(fc.Directories, fc.SupportedExtensions, vb, pb, input, fc.Logger, &fc.pendingMu, fc.ctx.Done(), &fc.wg)
	scanner.Start()
}
