package magiclink

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

type EncryptionKey []byte

func (e EncryptionKey) String() {
	///
}

const keySize = 32 // AES-256

func GenerateKey() (EncryptionKey, error) {
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func Encrypt(plaintext string, key EncryptionKey) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// nonce || ciphertext || tag
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func Decrypt(ciphertextB64 string, key EncryptionKey) (string, error) {
	sealed, err := base64.RawURLEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(sealed) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := sealed[:nonceSize], sealed[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt failed: %w", err)
	}
	return string(plain), nil
}

func (m *MagicLink) Encrypt(v any) (string, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return Encrypt(string(jsonBytes), m.key)
}

func (m *MagicLink) Decrypt(value string, v any) error {
	data, err := Decrypt(value, m.key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), v)
}

type MagicLink struct {
	key EncryptionKey
}

func New(key EncryptionKey) *MagicLink {
	return &MagicLink{key: key}
}
