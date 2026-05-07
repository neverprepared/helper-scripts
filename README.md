# helper-scripts

A collection of CLI tools for common development tasks.

## azprofile

Manage multiple Azure CLI identities. Switch between accounts without re-authenticating, refresh tokens on a schedule, and sync profile credentials to a portable location.

```bash
azprofile init work        # Create a profile and login
azprofile init personal    # Create another profile
azprofile use work         # Switch to the 'work' profile
azprofile list             # Show all profiles
azprofile current          # Show active profile
azprofile whoami           # Show active account details
azprofile login            # Re-authenticate the active profile
azprofile login work       # Re-authenticate 'work'

# Token refresh
azprofile refresh                            # Refresh all profiles
azprofile refresh work dev                   # Refresh specific profiles

# Cron scheduling
azprofile cron install                       # Refresh all profiles hourly
azprofile cron install work "*/30 * * * *"   # Refresh 'work' every 30 min
azprofile cron remove work                   # Remove cron for 'work'
azprofile cron remove                        # Remove all azprofile crons
azprofile cron status                        # Show installed crons

# Sync to/from a remote directory (USB drive, NAS, network share, etc.)
azprofile sync push /Volumes/backup          # Push active profile
azprofile sync push /Volumes/backup work     # Push 'work' profile
azprofile sync pull /Volumes/backup work     # Pull 'work' profile
```

Profiles are stored in `~/.azure-profiles/`. The active profile is symlinked to `~/.azure`.

### Environment

- `AZPROFILE_HOME` — base directory (default `$HOME`).
- `AZPROFILE_SYNC` — default sync directory; lets you omit the directory argument:
  ```bash
  export AZPROFILE_SYNC=/Volumes/backup
  azprofile sync push
  azprofile sync pull work
  ```

## Install

### From release

Download the latest release tarball for your OS/arch from [Releases](https://github.com/neverprepared/helper-scripts/releases), extract it, and put `azprofile` on your `PATH`:

```bash
tar -xzf azprofile-vX.Y.Z-darwin-arm64.tar.gz
mv azprofile-vX.Y.Z-darwin-arm64/azprofile ~/.local/bin/
```

Make sure `~/.local/bin` is on your `PATH`.

### From source

```bash
git clone https://github.com/neverprepared/helper-scripts.git
cd helper-scripts
make install
```

`make install` builds `bin/azprofile` and symlinks it into `$PREFIX/bin` (default `~/.local/bin`).

To install to a different location:

```bash
make install PREFIX=/usr/local
```

## Uninstall

```bash
make uninstall
```

## Requirements

- Go 1.22+ (to build from source)
- Azure CLI (`az`)
- `rsync` (for `azprofile sync`)

## License

MIT
