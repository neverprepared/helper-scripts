package azprofile

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/neverprepared/helper-scripts/internal/ui"
)

func Sync(action, remoteDir, profile string) error {
	if remoteDir == "" {
		remoteDir = os.Getenv("AZPROFILE_SYNC")
	}
	if remoteDir == "" {
		return fmt.Errorf("No sync directory specified. Pass a directory or set AZPROFILE_SYNC.")
	}

	home := Home()
	var localDir, remoteProfileDir string

	if profile != "" {
		localDir = filepath.Join(home, ".azure-profiles", profile) + "/"
		remoteProfileDir = filepath.Join(remoteDir, ".azure-profiles", profile) + "/"
	} else {
		cur := GetCurrent()
		if cur != "(none)" && cur != "(unmigrated directory)" {
			if fi, err := os.Stat(filepath.Join(home, ".azure-profiles", cur)); err == nil && fi.IsDir() {
				profile = cur
				localDir = filepath.Join(home, ".azure-profiles", cur) + "/"
				remoteProfileDir = filepath.Join(remoteDir, ".azure-profiles", cur) + "/"
			}
		}
		if localDir == "" {
			profile = "(default)"
			localDir = filepath.Join(home, ".azure") + "/"
			remoteProfileDir = filepath.Join(remoteDir, ".azure") + "/"
		}
	}

	switch action {
	case "push":
		if fi, err := os.Stat(localDir); err != nil || !fi.IsDir() {
			return fmt.Errorf("Local profile directory not found: %s", localDir)
		}
		return doSync(localDir, remoteProfileDir, "Pushing profile '"+profile+"'")
	case "pull":
		if fi, err := os.Stat(remoteProfileDir); err != nil || !fi.IsDir() {
			return fmt.Errorf("Remote profile directory not found: %s", remoteProfileDir)
		}
		return doSync(remoteProfileDir, localDir, "Pulling profile '"+profile+"'")
	default:
		return fmt.Errorf("Unknown action: %s. Use 'push' or 'pull'.", action)
	}
}

func doSync(src, dst, label string) error {
	fmt.Println()
	fmt.Printf("%s%s%s Azure Profile Sync%s\n", ui.Bold, ui.Blue, ui.SyncGlyph, ui.NC)
	fmt.Printf("%s─────────────────────%s\n", ui.Dim, ui.NC)
	fmt.Printf("%s%s%s %s\n", ui.Cyan, ui.Arrow, ui.NC, label)
	fmt.Printf("%s%s%s Source: %s%s%s\n", ui.Cyan, ui.Arrow, ui.NC, ui.Dim, src, ui.NC)
	fmt.Printf("%s%s%s Target: %s%s%s\n\n", ui.Cyan, ui.Arrow, ui.NC, ui.Dim, dst, ui.NC)

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	cmd := exec.Command("rsync", "-av", src, dst)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "sending") || strings.Contains(line, "sent") || strings.Contains(line, "total") {
			fmt.Printf("  %s%s%s\n", ui.Dim, line, ui.NC)
		} else if line != "" {
			fmt.Printf("  %s%s%s %s\n", ui.Green, ui.Check, ui.NC, line)
		}
	}

	if err := cmd.Wait(); err != nil {
		fmt.Println()
		fmt.Printf("%s%s Sync failed%s\n", ui.Red, ui.Cross, ui.NC)
		return err
	}
	fmt.Println()
	fmt.Printf("%s%s Sync complete%s\n", ui.Green, ui.Check, ui.NC)
	return nil
}
