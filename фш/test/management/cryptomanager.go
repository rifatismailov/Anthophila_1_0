package management

import "Anthophila/logging"

type CryptoManager struct {
	Encryptor  *Encryptor
	LogStatus  bool
	LogAddress string
}

func NewCryptoManager(logStatus bool, logAddress, key string) (*CryptoManager, error) {
	encryptor, err := NewEncryptor(key)
	if err != nil {
		return nil, err
	}
	return &CryptoManager{
		Encryptor:  encryptor,
		LogStatus:  logStatus,
		LogAddress: logAddress,
	}, nil
}

func (cm *CryptoManager) EncryptText(text string) string {
	encrypted, err := cm.Encryptor.EncryptText(text)
	if err != nil {
		if cm.LogStatus {
			logging.Log(cm.LogAddress, "Encryption error: ", err.Error())
		}
		return ""
	}
	return encrypted
}

func (cm *CryptoManager) DecryptText(text string) string {
	decrypted, err := cm.Encryptor.DecryptText(text)
	if err != nil {
		if cm.LogStatus {
			logging.Log(cm.LogAddress, "Decryption error: ", err.Error())
		}
		return ""
	}
	return decrypted
}
