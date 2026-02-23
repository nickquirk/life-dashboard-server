package mocks

// MockEncryptor implements crypto.TokenEncryptor.
// Default behaviour is passthrough (returns input unchanged).
type MockEncryptor struct {
	EncryptFunc func(plaintext string) (string, error)
	DecryptFunc func(ciphertext string) (string, error)
}

func (m *MockEncryptor) Encrypt(plaintext string) (string, error) {
	if m.EncryptFunc != nil {
		return m.EncryptFunc(plaintext)
	}
	return plaintext, nil
}

func (m *MockEncryptor) Decrypt(ciphertext string) (string, error) {
	if m.DecryptFunc != nil {
		return m.DecryptFunc(ciphertext)
	}
	return ciphertext, nil
}
