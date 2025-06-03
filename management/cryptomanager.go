package management

import "Anthophila/logging"

type CryptoManager struct {
	Logger    *logging.LoggerService
	Encryptor *Encryptor
}

func NewCryptoManager(logger *logging.LoggerService, key string) (*CryptoManager, error) {
	encryptor, err := NewEncryptor(key)
	if err != nil {
		return nil, err
	}
	return &CryptoManager{
		Logger:    logger,
		Encryptor: encryptor,
	}, nil
}

func (cm *CryptoManager) EncryptText(text string) string {
	encrypted, err := cm.Encryptor.EncryptText(text)
	if err != nil {
		cm.Logger.Log("Encryption error: ", err.Error())
		return ""
	}
	return encrypted
}

func (cm *CryptoManager) DecryptText(text string) string {
	decrypted, err := cm.Encryptor.DecryptText(text)
	if err != nil {
		cm.Logger.Log("Decryption error: ", err.Error())
		return ""
	}
	return decrypted
}
