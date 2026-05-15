package azprofile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var profileNameRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateProfileName rejects names containing characters that could break
// shell-quoted cron lines, path traversal, or surprising filesystem behavior.
// Allowed: alphanumeric, dot, underscore, hyphen. Disallowed: spaces, slashes,
// quotes, `..`, and shell metacharacters.
func ValidateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name is empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("profile name %q is reserved", name)
	}
	if !profileNameRE.MatchString(name) {
		return fmt.Errorf("profile name %q has invalid characters; use only [A-Za-z0-9._-]", name)
	}
	return nil
}

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
