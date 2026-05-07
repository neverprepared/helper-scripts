package azprofile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/neverprepared/helper-scripts/internal/ui"
)

func GetCurrent() string {
	link := ActiveLink()
	fi, err := os.Lstat(link)
	if err != nil {
		return "(none)"
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(link)
		if err != nil {
			return "(none)"
		}
		return filepath.Base(target)
	}
	if fi.IsDir() {
		return "(unmigrated directory)"
	}
	return "(none)"
}

func MigrateIfNeeded() error {
	link := ActiveLink()
	fi, err := os.Lstat(link)
	if err != nil {
		return nil
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return nil
	}
	if err := EnsureProfilesDir(); err != nil {
		return err
	}
	target := ProfilePath("default")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("Cannot migrate: %s already exists", target)
	}
	fmt.Printf("%s%s%s Migrating existing .azure/ to profile 'default'...\n", ui.Yellow, ui.Arrow, ui.NC)
	if err := os.Rename(link, target); err != nil {
		return err
	}
	if err := os.Symlink(target, link); err != nil {
		return err
	}
	fmt.Printf("%s%s%s Migrated. Active profile is now 'default'.\n\n", ui.Green, ui.Check, ui.NC)
	return nil
}

func Current() {
	fmt.Printf("%s%sActive profile:%s %s\n", ui.Bold, ui.Blue, ui.NC, GetCurrent())
}

func List() error {
	if err := EnsureProfilesDir(); err != nil {
		return err
	}
	current := GetCurrent()

	fmt.Printf("%s%sAzure Profiles%s\n", ui.Bold, ui.Blue, ui.NC)
	fmt.Printf("%s──────────────%s\n", ui.Dim, ui.NC)

	entries, err := os.ReadDir(ProfilesDir())
	if err != nil {
		return err
	}
	found := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		found++
		name := e.Name()
		if name == current {
			fmt.Printf("  %s%s%s %s%s%s %s(active)%s\n",
				ui.Green, ui.Check, ui.NC, ui.Bold, name, ui.NC, ui.Dim, ui.NC)
		} else {
			fmt.Printf("  %s-%s %s\n", ui.Dim, ui.NC, name)
		}
	}
	if found == 0 {
		fmt.Printf("  %sNo profiles. Run: azprofile init <name>%s\n", ui.Dim, ui.NC)
	}
	return nil
}

func Use(name string) error {
	if name == "" {
		return fmt.Errorf("Usage: azprofile use <name>")
	}
	if err := MigrateIfNeeded(); err != nil {
		return err
	}
	target := ProfilePath(name)
	if fi, err := os.Stat(target); err != nil || !fi.IsDir() {
		return fmt.Errorf("Profile '%s' not found. Run: azprofile init %s", name, name)
	}
	link := ActiveLink()
	if fi, err := os.Lstat(link); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(link); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%s exists and is not a symlink. Run azprofile again to migrate.", link)
		}
	}
	if err := os.Symlink(target, link); err != nil {
		return err
	}
	fmt.Printf("%s%s%s Switched to %s%s%s\n", ui.Green, ui.Check, ui.NC, ui.Bold, name, ui.NC)

	if _, err := exec.LookPath("az"); err == nil {
		cmd := exec.Command("az", "account", "show",
			"--query", "{user:user.name, subscription:name, tenant:tenantId}",
			"-o", "tsv")
		cmd.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+target)
		out, err := cmd.Output()
		if err == nil {
			line := strings.TrimSpace(string(out))
			if line != "" {
				if fields := strings.Fields(line); len(fields) > 0 {
					fmt.Printf("%s  %s%s\n", ui.Dim, fields[0], ui.NC)
				}
			}
		}
	}
	return nil
}

func Init(name string) error {
	if name == "" {
		return fmt.Errorf("Usage: azprofile init <name>")
	}
	if err := MigrateIfNeeded(); err != nil {
		return err
	}
	if err := EnsureProfilesDir(); err != nil {
		return err
	}
	target := ProfilePath(name)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}

	fmt.Printf("%s%sInitializing profile '%s'%s\n", ui.Bold, ui.Blue, name, ui.NC)
	fmt.Printf("%s%s%s Config dir: %s%s%s\n\n", ui.Cyan, ui.Arrow, ui.NC, ui.Dim, target, ui.NC)

	if err := runAzLogin(target); err != nil {
		return err
	}

	fmt.Printf("\n%s%s%s Profile '%s' initialized.\n", ui.Green, ui.Check, ui.NC, name)
	fmt.Printf("%s  Switch to it with: azprofile use %s%s\n", ui.Dim, name, ui.NC)
	return nil
}

func Login(name string) error {
	if name == "" {
		name = GetCurrent()
		if name == "(none)" || name == "(unmigrated directory)" {
			return fmt.Errorf("No active profile. Specify one: azprofile login <name>")
		}
	}
	target := ProfilePath(name)
	if fi, err := os.Stat(target); err != nil || !fi.IsDir() {
		return fmt.Errorf("Profile '%s' not found. Run: azprofile init %s", name, name)
	}

	fmt.Printf("%s%sRe-authenticating profile '%s'%s\n", ui.Bold, ui.Blue, name, ui.NC)
	fmt.Printf("%s%s%s Config dir: %s%s%s\n\n", ui.Cyan, ui.Arrow, ui.NC, ui.Dim, target, ui.NC)

	if err := runAzLogin(target); err != nil {
		return err
	}

	fmt.Printf("\n%s%s%s Profile '%s' re-authenticated.\n", ui.Green, ui.Check, ui.NC, name)
	return nil
}

func runAzLogin(configDir string) error {
	cmd := exec.Command("az", "login")
	cmd.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+configDir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Whoami() error {
	if _, err := exec.LookPath("az"); err != nil {
		return fmt.Errorf("az CLI not found")
	}
	current := GetCurrent()
	fmt.Printf("%s%sProfile:%s %s\n", ui.Bold, ui.Blue, ui.NC, current)
	fmt.Printf("%s──────────────%s\n", ui.Dim, ui.NC)
	cmd := exec.Command("az", "account", "show", "-o", "table")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
