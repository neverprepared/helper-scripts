package azprofile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// utf8BOM appears at the start of azureProfile.json.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type azureProfileFile struct {
	Subscriptions []struct {
		User struct {
			Name string `json:"name"`
		} `json:"user"`
	} `json:"subscriptions"`
}

// AzureUPNFromProfile reads <profileDir>/azureProfile.json and returns the
// first non-empty user.name. Returns an error if the file is missing or
// contains no user identity (e.g., not yet logged in).
func AzureUPNFromProfile(profileDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(profileDir, "azureProfile.json"))
	if err != nil {
		return "", err
	}
	b = bytes.TrimPrefix(b, utf8BOM)
	var f azureProfileFile
	if err := json.Unmarshal(b, &f); err != nil {
		return "", fmt.Errorf("parse azureProfile.json: %w", err)
	}
	for _, s := range f.Subscriptions {
		if s.User.Name != "" {
			return s.User.Name, nil
		}
	}
	return "", errors.New("no user identity in azureProfile.json (not logged in?)")
}

// AzureUPNHash returns the first 12 hex chars of SHA-256(upn). Used in the
// Ably channel name so the channel doesn't leak the user identity.
func AzureUPNHash(upn string) string {
	sum := sha256.Sum256([]byte(upn))
	return hex.EncodeToString(sum[:])[:12]
}
