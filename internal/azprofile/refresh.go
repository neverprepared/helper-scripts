package azprofile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/neverprepared/helper-scripts/internal/ui"
)

func ensureCronPath() {
	sep := string(os.PathListSeparator)
	current := os.Getenv("PATH")
	parts := strings.Split(current, sep)
	have := make(map[string]bool, len(parts))
	for _, p := range parts {
		have[p] = true
	}
	home, _ := os.UserHomeDir()
	candidates := []string{"/usr/local/bin", "/opt/homebrew/bin", filepath.Join(home, ".local/bin")}
	for _, c := range candidates {
		if have[c] {
			continue
		}
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			parts = append([]string{c}, parts...)
			have[c] = true
		}
	}
	os.Setenv("PATH", strings.Join(parts, sep))
}

func refreshOne(name string) bool {
	dir := ProfilePath(name)
	getToken := exec.Command("az", "account", "get-access-token", "--output", "none")
	getToken.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+dir)
	if err := getToken.Run(); err != nil {
		fmt.Printf("  %s%s%s %s%s%s %s— token refresh failed, may need: azprofile init %s%s\n",
			ui.Red, ui.Cross, ui.NC, ui.Bold, name, ui.NC, ui.Dim, name, ui.NC)
		return false
	}
	show := exec.Command("az", "account", "show", "--query", "user.name", "-o", "tsv")
	show.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+dir)
	user := "unknown"
	if out, err := show.Output(); err == nil {
		if u := strings.TrimSpace(string(out)); u != "" {
			user = u
		}
	}
	fmt.Printf("  %s%s%s %s%s%s %s(%s)%s\n",
		ui.Green, ui.Check, ui.NC, ui.Bold, name, ui.NC, ui.Dim, user, ui.NC)
	return true
}

// Refresh refreshes tokens for the given profiles, or all profiles if names is empty.
// Returns the number of failures (intended to be used as the process exit code).
func Refresh(names []string) int {
	ensureCronPath()

	fmt.Printf("[%s] %s%s%s Azure Token Refresh%s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		ui.Bold, ui.Blue, ui.Arrow, ui.NC)
	fmt.Printf("%s────────────────────%s\n", ui.Dim, ui.NC)

	if fi, err := os.Stat(ProfilesDir()); err != nil || !fi.IsDir() {
		fmt.Printf("  %sNo profiles found.%s\n", ui.Yellow, ui.NC)
		return 0
	}

	failures := 0
	if len(names) > 0 {
		for _, name := range names {
			if fi, err := os.Stat(ProfilePath(name)); err != nil || !fi.IsDir() {
				fmt.Printf("  %s%s%s %s%s%s %s— profile not found%s\n",
					ui.Red, ui.Cross, ui.NC, ui.Bold, name, ui.NC, ui.Dim, ui.NC)
				failures++
				continue
			}
			if !refreshOne(name) {
				failures++
			}
		}
		return failures
	}

	entries, _ := os.ReadDir(ProfilesDir())
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !refreshOne(e.Name()) {
			failures++
		}
	}
	return failures
}
