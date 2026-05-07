package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/neverprepared/helper-scripts/internal/azprofile"
	"github.com/neverprepared/helper-scripts/internal/ui"
)

func main() {
	root := &cobra.Command{
		Use:   "azprofile",
		Short: "Azure multi-identity manager",
		Long:  "azprofile — Azure multi-identity manager. Create, switch, refresh, and sync Azure CLI identities.",
	}
	root.SilenceUsage = true
	root.SilenceErrors = true

	root.AddCommand(
		listCmd(),
		useCmd(),
		currentCmd(),
		initCmd(),
		loginCmd(),
		whoamiCmd(),
		cronCmd(),
		refreshCmd(),
		syncCmd(),
	)

	if err := root.Execute(); err != nil {
		ui.Die("%s", err.Error())
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return azprofile.List()
		},
	}
}

func useCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Switch to a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return azprofile.Use(args[0])
		},
	}
}

func currentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show active profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			azprofile.Current()
			return nil
		},
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <name>",
		Short: "Create a profile and login",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return azprofile.Init(args[0])
		},
	}
}

func loginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login [name]",
		Short: "Re-authenticate a profile (default: active)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return azprofile.Login(name)
		},
	}
}

func whoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show active account details",
		RunE: func(cmd *cobra.Command, args []string) error {
			return azprofile.Whoami()
		},
	}
}

func cronCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "cron",
		Short: "Manage token-refresh cron entries",
	}

	install := &cobra.Command{
		Use:   "install [profile] [schedule]",
		Short: "Install a refresh cron (all profiles or one)",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, schedule := "", ""
			if len(args) >= 1 {
				profile = args[0]
			}
			if len(args) >= 2 {
				schedule = args[1]
			}
			return azprofile.CronInstall(profile, schedule)
		},
	}
	remove := &cobra.Command{
		Use:   "remove [profile]",
		Short: "Remove cron (specific profile, or all if omitted)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := ""
			if len(args) == 1 {
				profile = args[0]
			}
			return azprofile.CronRemove(profile)
		},
	}
	status := &cobra.Command{
		Use:   "status",
		Short: "Show installed crons",
		RunE: func(cmd *cobra.Command, args []string) error {
			azprofile.CronStatus()
			return nil
		},
	}

	c.AddCommand(install, remove, status)
	return c
}

func refreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh [profiles...]",
		Short: "Refresh Azure tokens (all profiles or specific ones)",
		RunE: func(cmd *cobra.Command, args []string) error {
			failures := azprofile.Refresh(args)
			if failures > 0 {
				os.Exit(failures)
			}
			return nil
		},
	}
}

func syncCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "sync",
		Short: "Sync a profile to/from a remote directory",
		Long: `Sync a profile to/from a remote directory.

Environment:
  AZPROFILE_HOME    Base directory (default: $HOME)
  AZPROFILE_SYNC    Default sync directory (used if <dir> is omitted)`,
	}

	push := &cobra.Command{
		Use:   "push [dir] [profile]",
		Short: "Push a profile to a remote directory",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync("push", args)
		},
	}
	pull := &cobra.Command{
		Use:   "pull [dir] [profile]",
		Short: "Pull a profile from a remote directory",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync("pull", args)
		},
	}

	c.AddCommand(push, pull)
	return c
}

func runSync(action string, args []string) error {
	dir, profile := "", ""
	if len(args) >= 1 {
		dir = args[0]
	}
	if len(args) >= 2 {
		profile = args[1]
	}
	return azprofile.Sync(action, dir, profile)
}
