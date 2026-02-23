package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func TestNewAESGCMEncryptor_ValidKey(t *testing.T) {
	enc, err := NewAESGCMEncryptor(validKey(t))
	require.NoError(t, err)
	assert.NotNil(t, enc)
}

func TestNewAESGCMEncryptor_InvalidKeySize(t *testing.T) {
	_, err := NewAESGCMEncryptor([]byte("short"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	enc, err := NewAESGCMEncryptor(validKey(t))
	require.NoError(t, err)

	plaintext := "my-secret-token-value"
	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	enc, err := NewAESGCMEncryptor(validKey(t))
	require.NoError(t, err)

	ct, err := enc.Encrypt("")
	require.NoError(t, err)
	assert.Equal(t, "", ct)

	pt, err := enc.Decrypt("")
	require.NoError(t, err)
	assert.Equal(t, "", pt)
}

func TestEncrypt_DifferentCiphertexts(t *testing.T) {
	enc, err := NewAESGCMEncryptor(validKey(t))
	require.NoError(t, err)

	ct1, err := enc.Encrypt("same-input")
	require.NoError(t, err)
	ct2, err := enc.Encrypt("same-input")
	require.NoError(t, err)

	// Random nonce should produce different ciphertexts
	assert.NotEqual(t, ct1, ct2)
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	enc, err := NewAESGCMEncryptor(validKey(t))
	require.NoError(t, err)

	_, err = enc.Decrypt("not-valid-base64!!!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base64")
}

func TestDecrypt_TruncatedData(t *testing.T) {
	enc, err := NewAESGCMEncryptor(validKey(t))
	require.NoError(t, err)

	// Base64 of a very short byte slice (shorter than nonce)
	short := base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
	_, err = enc.Decrypt(short)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	enc, err := NewAESGCMEncryptor(validKey(t))
	require.NoError(t, err)

	ct, err := enc.Encrypt("secret")
	require.NoError(t, err)

	// Decode, flip a byte, re-encode
	data, err := base64.StdEncoding.DecodeString(ct)
	require.NoError(t, err)
	data[len(data)-1] ^= 0xFF
	tampered := base64.StdEncoding.EncodeToString(data)

	_, err = enc.Decrypt(tampered)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decrypt")
}

func TestDecrypt_WrongKey(t *testing.T) {
	enc1, err := NewAESGCMEncryptor(validKey(t))
	require.NoError(t, err)
	enc2, err := NewAESGCMEncryptor(validKey(t))
	require.NoError(t, err)

	ct, err := enc1.Encrypt("secret")
	require.NoError(t, err)

	_, err = enc2.Decrypt(ct)
	assert.Error(t, err)
}

func TestNewAESGCMEncryptor_16ByteKey(t *testing.T) {
	_, err := NewAESGCMEncryptor(make([]byte, 16))
	assert.Error(t, err)
}
