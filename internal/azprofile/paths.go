package azprofile

import (
	"os"
	"path/filepath"
)

func Home() string {
	if v := os.Getenv("AZPROFILE_HOME"); v != "" {
		return v
	}
	h, _ := os.UserHomeDir()
	return h
}

func ProfilesDir() string {
	return filepath.Join(Home(), ".azure-profiles")
}

func ActiveLink() string {
	return filepath.Join(Home(), ".azure")
}

func ProfilePath(name string) string {
	return filepath.Join(ProfilesDir(), name)
}

func EnsureProfilesDir() error {
	return os.MkdirAll(ProfilesDir(), 0o755)
}
