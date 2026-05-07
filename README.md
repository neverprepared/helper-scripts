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
azprofile sync push /Volumes/backup          # Push active profile (rsync)
azprofile sync push /Volumes/backup work     # Push 'work' profile (rsync)
azprofile sync pull /Volumes/backup work     # Pull 'work' profile (rsync)

# Sync between machines via Ably pub/sub (encrypted, ~30 KB per push)
azprofile sync keygen                        # Generate master key, store in OS keychain
azprofile sync configure --ably-key APP.KEY:SECRET --channel-prefix mygroup
azprofile sync push --ably work              # Encrypt + publish 'work' tokens
azprofile sync pull --ably work              # Read latest message and apply
azprofile sync subscribe work                # Long-running daemon (apply as they arrive)
azprofile sync status                        # Show non-secret config + key fingerprint
```

Profiles are stored in `~/.azure-profiles/`. The active profile is symlinked to `~/.azure`.

### Environment

- `AZPROFILE_HOME` — base directory (default `$HOME`).
- `AZPROFILE_SYNC` — default rsync directory; lets you omit the directory argument:
  ```bash
  export AZPROFILE_SYNC=/Volumes/backup
  azprofile sync push
  azprofile sync pull work
  ```
- `AZPROFILE_MASTER_KEY` — hex-encoded 32-byte master key. When set, overrides the OS keychain. Use this for headless Linux boxes without a secret-service daemon.

## Ably pub/sub sync

Use this mode when you want to keep multiple personal machines on the same Azure tokens without carrying a USB stick or running rsync over a VPN. Only the four files Azure CLI needs to authenticate (`msal_token_cache.json`, `msal_http_cache.bin`, `azureProfile.json`, `clouds.config`) travel — telemetry, command indexes, and logs stay local.

The published payload is encrypted with AES-256-GCM **before** it leaves your machine. Ably never sees plaintext. The same key encrypts the local `~/.config/azprofile/config.enc`, so the Ably API key is also unreadable at rest.

### Initial setup

On the first machine:

```bash
azprofile sync keygen                                       # generates + stores key, prints hex
KEY=$(azprofile sync export-key --confirm)                  # copy hex out-of-band
azprofile sync configure --ably-key "APP.KEY:SECRET" \
                         --channel-prefix mygroup
azprofile sync push --ably work
```

On each additional machine:

```bash
azprofile sync import-key "<paste-hex-from-machine-1>"
azprofile sync configure --ably-key "APP.KEY:SECRET" \
                         --channel-prefix mygroup
azprofile sync pull --ably work
```

The channel name is `<prefix>.<sha256(upn)[:12]>.<profile>` — derived from the Azure UPN inside `azureProfile.json` so the channel doesn't leak your identity.

### Auto-publish

`azprofile refresh`, `azprofile init`, and `azprofile login` automatically publish to Ably when `~/.config/azprofile/config.enc` exists. The hook is best-effort — it never fails the parent command. Combined with the existing cron entry, this keeps every subscribed machine on the latest refresh token without any extra wiring.

### Subscribe daemon (launchd)

`azprofile sync subscribe <profile>` is a long-running process. Wire it under launchd with something like (adjust paths):

```xml
<plist version="1.0"><dict>
  <key>Label</key>           <string>local.azprofile.subscribe.work</string>
  <key>ProgramArguments</key><array>
    <string>/Users/you/.local/bin/azprofile</string>
    <string>sync</string><string>subscribe</string><string>work</string>
  </array>
  <key>KeepAlive</key>       <true/>
  <key>RunAtLoad</key>       <true/>
  <key>StandardOutPath</key> <string>/Users/you/.azure-profiles/subscribe.log</string>
  <key>StandardErrorPath</key><string>/Users/you/.azure-profiles/subscribe.log</string>
</dict></plist>
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
- `rsync` (for the rsync sync mode)
- An Ably account + API key (only for the Ably sync mode — free tier is plenty for personal use)
- macOS Keychain or Linux secret-service (or set `AZPROFILE_MASTER_KEY` to bypass)

## License

MIT
