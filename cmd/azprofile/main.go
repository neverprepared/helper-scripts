package main

import (
	"errors"
	"fmt"
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
		Short: "Sync a profile to/from a remote directory or via Ably pub/sub",
		Long: `Sync a profile.

Two modes:
  rsync (default)   Copy a profile dir to/from a remote directory.
  ably (--ably)     Publish/receive an encrypted token bundle via Ably pub/sub.

Environment:
  AZPROFILE_HOME           Base directory (default: $HOME)
  AZPROFILE_SYNC           Default rsync directory (used if <dir> is omitted)
  AZPROFILE_MASTER_KEY     Hex-encoded master key (overrides OS keychain)`,
	}

	var pushAbly bool
	push := &cobra.Command{
		Use:   "push [dir] [profile]",
		Short: "Push a profile to a remote directory (or Ably with --ably)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if pushAbly {
				if len(args) > 1 {
					return fmt.Errorf("--ably accepts at most one positional [profile]")
				}
				profile := ""
				if len(args) == 1 {
					profile = args[0]
				}
				return azprofile.PublishProfile(cmd.Context(), profile)
			}
			if len(args) > 2 {
				return fmt.Errorf("too many arguments")
			}
			return runRsyncSync("push", args)
		},
	}
	push.Flags().BoolVar(&pushAbly, "ably", false, "Publish via Ably pub/sub instead of rsync")

	var pullAbly bool
	pull := &cobra.Command{
		Use:   "pull [dir] [profile]",
		Short: "Pull a profile from a remote directory (or Ably with --ably)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if pullAbly {
				if len(args) > 1 {
					return fmt.Errorf("--ably accepts at most one positional [profile]")
				}
				profile := ""
				if len(args) == 1 {
					profile = args[0]
				}
				return azprofile.PullOnce(cmd.Context(), profile)
			}
			if len(args) > 2 {
				return fmt.Errorf("too many arguments")
			}
			return runRsyncSync("pull", args)
		},
	}
	pull.Flags().BoolVar(&pullAbly, "ably", false, "Pull via Ably pub/sub instead of rsync")

	subscribe := &cobra.Command{
		Use:   "subscribe [profile]",
		Short: "Long-running daemon that applies inbound Ably token updates",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := ""
			if len(args) == 1 {
				profile = args[0]
			}
			return azprofile.Subscribe(cmd.Context(), profile)
		},
	}

	keygen := &cobra.Command{
		Use:   "keygen",
		Short: "Generate a master key, store it in the OS keychain, print the hex",
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				if _, err := azprofile.LoadMasterKey(); err == nil {
					return fmt.Errorf("master key already exists; pass --force to overwrite")
				}
			}
			k, err := azprofile.NewMasterKey()
			if err != nil {
				return err
			}
			if err := azprofile.SaveMasterKey(k); err != nil {
				return err
			}
			hexKey := azprofile.KeyToHex(k)
			fmt.Println(hexKey)
			fmt.Fprintln(os.Stderr, "Transfer this key to other machines via a secure channel, then run: azprofile sync import-key <hex>")
			return nil
		},
	}
	keygen.Flags().Bool("force", false, "Overwrite an existing key")

	importKey := &cobra.Command{
		Use:   "import-key <hex>",
		Short: "Import a master key into the OS keychain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			k, err := azprofile.KeyFromHex(args[0])
			if err != nil {
				return err
			}
			return azprofile.SaveMasterKey(k)
		},
	}

	exportKey := &cobra.Command{
		Use:   "export-key",
		Short: "Print the master key (requires --confirm)",
		RunE: func(cmd *cobra.Command, args []string) error {
			confirm, _ := cmd.Flags().GetBool("confirm")
			if !confirm {
				return fmt.Errorf("refusing to print master key without --confirm")
			}
			k, err := azprofile.LoadMasterKey()
			if err != nil {
				return err
			}
			fmt.Println(azprofile.KeyToHex(k))
			return nil
		},
	}
	exportKey.Flags().Bool("confirm", false, "Acknowledge that the key will be printed to stdout")

	configure := &cobra.Command{
		Use:   "configure",
		Short: "Write the encrypted Ably sync config (.config/azprofile/config.enc)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ablyKey, _ := cmd.Flags().GetString("ably-key")
			prefix, _ := cmd.Flags().GetString("channel-prefix")
			if ablyKey == "" {
				return fmt.Errorf("--ably-key is required")
			}
			cur, _ := azprofile.LoadConfig()
			cfg := &azprofile.SyncConfig{
				AblyAPIKey:    ablyKey,
				ChannelPrefix: prefix,
			}
			if cur != nil {
				cfg.SenderID = cur.SenderID
			}
			return azprofile.SaveConfig(cfg)
		},
	}
	configure.Flags().String("ably-key", "", "Ably API key (form: appId.keyId:secret)")
	configure.Flags().String("channel-prefix", "azprofile", "Channel prefix")

	status := &cobra.Command{
		Use:   "status",
		Short: "Show non-secret sync configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSyncStatus()
		},
	}

	c.AddCommand(push, pull, subscribe, keygen, importKey, exportKey, configure, status)
	return c
}

func runRsyncSync(action string, args []string) error {
	dir, profile := "", ""
	if len(args) >= 1 {
		dir = args[0]
	}
	if len(args) >= 2 {
		profile = args[1]
	}
	return azprofile.Sync(action, dir, profile)
}

func runSyncStatus() error {
	fmt.Printf("Config path:    %s\n", azprofile.ConfigPath())
	fmt.Printf("State path:     %s\n", azprofile.StatePath())
	fmt.Printf("Master key:     %s\n", azprofile.MasterKeySource())

	key, err := azprofile.LoadMasterKey()
	if err != nil {
		fmt.Printf("Status:         not configured (%s)\n", err.Error())
		return nil
	}
	fmt.Printf("Key fingerprint: %s\n", azprofile.KeyFingerprint(key))

	cfg, err := azprofile.LoadConfig()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Config:         not written yet (run: azprofile sync configure ...)")
			return nil
		}
		return err
	}
	fmt.Printf("Channel prefix: %s\n", cfg.ChannelPrefix)
	fmt.Printf("Sender ID:      %s\n", cfg.SenderID)
	if len(cfg.AblyAPIKey) > 8 {
		fmt.Printf("Ably API key:   %s…(%d chars)\n", cfg.AblyAPIKey[:8], len(cfg.AblyAPIKey))
	}
	return nil
}
