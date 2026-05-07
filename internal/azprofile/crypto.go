package azprofile

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	cryptoMagic     = "AZP1"
	cryptoNonceSize = 12
	cryptoKeySize   = 32
)

// EncryptGCM encrypts plaintext with key (32 bytes) using AES-256-GCM.
// Output layout: magic(4) | nonce(12) | ciphertext+tag.
func EncryptGCM(key, plaintext []byte) ([]byte, error) {
	if len(key) != cryptoKeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", cryptoKeySize, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, cryptoNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(cryptoMagic)+cryptoNonceSize+len(plaintext)+gcm.Overhead())
	out = append(out, cryptoMagic...)
	out = append(out, nonce...)
	out = gcm.Seal(out, nonce, plaintext, nil)
	return out, nil
}

// DecryptGCM reverses EncryptGCM.
func DecryptGCM(key, blob []byte) ([]byte, error) {
	if len(key) != cryptoKeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", cryptoKeySize, len(key))
	}
	if len(blob) < len(cryptoMagic)+cryptoNonceSize+16 {
		return nil, errors.New("ciphertext too short")
	}
	if string(blob[:len(cryptoMagic)]) != cryptoMagic {
		return nil, errors.New("bad magic; not an azprofile encrypted blob")
	}
	nonce := blob[len(cryptoMagic) : len(cryptoMagic)+cryptoNonceSize]
	ct := blob[len(cryptoMagic)+cryptoNonceSize:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ct, nil)
}

func NewMasterKey() ([]byte, error) {
	k := make([]byte, cryptoKeySize)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	return k, nil
}

func KeyToHex(key []byte) string {
	return hex.EncodeToString(key)
}

func KeyFromHex(s string) ([]byte, error) {
	k, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}
	if len(k) != cryptoKeySize {
		return nil, fmt.Errorf("key must decode to %d bytes, got %d", cryptoKeySize, len(k))
	}
	return k, nil
}

// KeyFingerprint returns the first 8 hex chars of SHA-256(key) for display.
func KeyFingerprint(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:])[:8]
}
