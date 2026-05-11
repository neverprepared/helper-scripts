package azprofile

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/neverprepared/helper-scripts/internal/ui"
)

func requirePimCli() error {
	if _, err := exec.LookPath("az-pim-cli"); err != nil {
		return fmt.Errorf("az-pim-cli not found. Install: go install github.com/neverprepared/az-pim-cli@latest")
	}
	return nil
}

func pimHeader() {
	current := GetCurrent()
	fmt.Printf("%s%sProfile:%s %s\n", ui.Bold, ui.Blue, ui.NC, current)
	fmt.Printf("%s──────────────%s\n", ui.Dim, ui.NC)
}

func runPimCmd(args ...string) error {
	cmd := exec.Command("az-pim-cli", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func PimList() error {
	if err := requirePimCli(); err != nil {
		return err
	}
	pimHeader()
	fmt.Printf("\n%s%sEligible Role Assignments%s\n", ui.Bold, ui.Blue, ui.NC)
	return runPimCmd("list", "resource")
}

func PimActive() error {
	if err := requirePimCli(); err != nil {
		return err
	}
	pimHeader()
	fmt.Printf("\n%s%sActive Role Assignments%s\n", ui.Bold, ui.Blue, ui.NC)
	return runPimCmd("list", "active")
}

func PimActivate(names []string, role string, duration int, reason string, wait bool) error {
	if err := requirePimCli(); err != nil {
		return err
	}
	pimHeader()

	args := []string{"activate", "resource"}
	for _, n := range names {
		args = append(args, "--name", n)
	}
	if role != "" {
		args = append(args, "--role", role)
	}
	if duration > 0 {
		args = append(args, "--duration", fmt.Sprintf("%d", duration))
	}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	if wait {
		args = append(args, "--wait")
	}

	fmt.Printf("\n%s%s%s Activating: %s\n", ui.Cyan, ui.Arrow, ui.NC, strings.Join(names, ", "))
	if err := runPimCmd(args...); err != nil {
		return err
	}
	fmt.Printf("%s%s%s Activation complete\n", ui.Green, ui.Check, ui.NC)
	return nil
}

func PimDeactivate(names []string, role string) error {
	if err := requirePimCli(); err != nil {
		return err
	}
	pimHeader()

	args := []string{"deactivate", "resource"}
	for _, n := range names {
		args = append(args, "--name", n)
	}
	if role != "" {
		args = append(args, "--role", role)
	}

	fmt.Printf("\n%s%s%s Deactivating: %s\n", ui.Cyan, ui.Arrow, ui.NC, strings.Join(names, ", "))
	if err := runPimCmd(args...); err != nil {
		return err
	}
	fmt.Printf("%s%s%s Deactivation complete\n", ui.Green, ui.Check, ui.NC)
	return nil
}
