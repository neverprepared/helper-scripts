# helper-scripts

A collection of standalone CLI tools for common development tasks.

## Tools

### azprofile

Manage multiple Azure CLI identities. Switch between accounts without re-authenticating.

```bash
azprofile init work        # Create a profile and login
azprofile init personal    # Create another profile
azprofile use work         # Switch to the 'work' profile
azprofile list             # Show all profiles
azprofile whoami           # Show active account details
azprofile login                              # Re-authenticate the active profile
azprofile login work                         # Re-authenticate 'work'
azprofile cron install                       # Refresh all profiles hourly
azprofile cron install work "*/30 * * * *"   # Refresh 'work' every 30 min
azprofile cron remove work                   # Remove cron for 'work'
azprofile cron status                        # Show installed crons
```

Profiles are stored in `~/.azure-profiles/`. The active profile is symlinked to `~/.azure`.

Set `AZPROFILE_HOME` to change the base directory (defaults to `$HOME`).

### azprofile-refresh

Refresh Azure tokens for all profiles (or specific ones). Designed to run from cron.

```bash
azprofile-refresh              # Refresh all profiles
azprofile-refresh work dev     # Refresh specific profiles
```

### azprofile-sync

Sync Azure profile credentials to or from a remote directory (USB drive, NAS, network share, etc.).

```bash
azprofile-sync push /Volumes/backup          # Push active profile
azprofile-sync push /Volumes/backup work     # Push 'work' profile
azprofile-sync pull /Volumes/backup work     # Pull 'work' profile
```

Set `AZPROFILE_SYNC` to skip the directory argument:

```bash
export AZPROFILE_SYNC=/Volumes/backup
azprofile-sync push
azprofile-sync pull work
```

## Install

### From release

Download the latest release zip from [Releases](https://github.com/neverprepared/helper-scripts/releases), extract it, and run:

```bash
make install
```

This symlinks all tools into `~/.local/bin`. Make sure `~/.local/bin` is on your `PATH`.

To install to a different location:

```bash
make install PREFIX=/usr/local
```

### From source

```bash
git clone https://github.com/neverprepared/helper-scripts.git
cd helper-scripts
make install
```

## Uninstall

```bash
make uninstall
```

## Requirements

- bash 4+
- Azure CLI (`az`) — for azprofile tools

## License

MIT
