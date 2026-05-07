package azprofile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/neverprepared/helper-scripts/internal/ui"
)

const CronTagPrefix = "# azprofile-refresh"

const DefaultCronSchedule = "13 * * * *"

func cronTag(profile string) string {
	if profile == "" {
		profile = "all"
	}
	return CronTagPrefix + ":" + profile
}

func readCrontab() string {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func writeCrontab(content string) error {
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func filterOut(content, substr string) string {
	if content == "" {
		return ""
	}
	var kept []string
	for _, line := range strings.Split(content, "\n") {
		if !strings.Contains(line, substr) {
			kept = append(kept, line)
		}
	}
	return strings.TrimRight(strings.Join(kept, "\n"), "\n")
}

func CronInstall(profile, schedule string) error {
	if schedule == "" {
		schedule = DefaultCronSchedule
	}
	tag := cronTag(profile)

	refreshArgs := ""
	label := "all profiles"
	if profile != "" {
		if fi, err := os.Stat(ProfilePath(profile)); err != nil || !fi.IsDir() {
			return fmt.Errorf("Profile '%s' not found. Run: azprofile init %s", profile, profile)
		}
		refreshArgs = " " + profile
		label = "profile '" + profile + "'"
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	home := Home()
	logPath := filepath.Join(home, ".azure-profiles", "refresh.log")

	cronLine := fmt.Sprintf("AZPROFILE_HOME=%s %s %s refresh%s >> %s 2>&1 %s",
		home, schedule, exe, refreshArgs, logPath, tag)

	filtered := filterOut(readCrontab(), tag)
	var content string
	if filtered == "" {
		content = cronLine + "\n"
	} else {
		content = filtered + "\n" + cronLine + "\n"
	}
	if err := writeCrontab(content); err != nil {
		return err
	}

	fmt.Printf("%s%s%s Cron installed for %s%s%s: %s%s%s\n",
		ui.Green, ui.Check, ui.NC, ui.Bold, label, ui.NC, ui.Dim, schedule, ui.NC)
	fmt.Printf("%s  Log: %s%s\n", ui.Dim, logPath, ui.NC)
	return nil
}

func CronRemove(profile string) error {
	tag := CronTagPrefix
	if profile != "" {
		tag = cronTag(profile)
	}
	filtered := filterOut(readCrontab(), tag)
	if filtered == "" {
		_ = exec.Command("crontab", "-r").Run()
	} else {
		if err := writeCrontab(filtered + "\n"); err != nil {
			return err
		}
	}
	if profile != "" {
		fmt.Printf("%s%s%s Cron removed for profile '%s'\n", ui.Green, ui.Check, ui.NC, profile)
	} else {
		fmt.Printf("%s%s%s All azprofile crons removed\n", ui.Green, ui.Check, ui.NC)
	}
	return nil
}

func CronStatus() {
	var lines []string
	for _, line := range strings.Split(readCrontab(), "\n") {
		if strings.Contains(line, CronTagPrefix) {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		fmt.Printf("%s-%s No crons installed. Run: azprofile cron install [profile] [schedule]\n", ui.Dim, ui.NC)
		return
	}
	fmt.Printf("%s%s%s Installed crons:\n", ui.Green, ui.Check, ui.NC)
	for _, l := range lines {
		fmt.Printf("  %s%s%s\n", ui.Dim, l, ui.NC)
	}
}
