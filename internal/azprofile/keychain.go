package azprofile

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keychainService = "azprofile"
	keychainAccount = "master-key"
	envMasterKey    = "AZPROFILE_MASTER_KEY"
)

// LoadMasterKey returns the 32-byte master key.
// Order: AZPROFILE_MASTER_KEY env var (hex) > OS keychain.
func LoadMasterKey() ([]byte, error) {
	if v := strings.TrimSpace(os.Getenv(envMasterKey)); v != "" {
		k, err := KeyFromHex(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", envMasterKey, err)
		}
		return k, nil
	}
	s, err := keyring.Get(keychainService, keychainAccount)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, fmt.Errorf("master key not found. Run: azprofile sync keygen (or set %s)", envMasterKey)
		}
		return nil, fmt.Errorf("keychain: %w", err)
	}
	return KeyFromHex(strings.TrimSpace(s))
}

func SaveMasterKey(key []byte) error {
	if len(key) != cryptoKeySize {
		return fmt.Errorf("key must be %d bytes", cryptoKeySize)
	}
	return keyring.Set(keychainService, keychainAccount, KeyToHex(key))
}

func DeleteMasterKey() error {
	return keyring.Delete(keychainService, keychainAccount)
}

// MasterKeySource describes where LoadMasterKey would fetch the key.
func MasterKeySource() string {
	if strings.TrimSpace(os.Getenv(envMasterKey)) != "" {
		return "env:" + envMasterKey
	}
	return "keychain"
}
