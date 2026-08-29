# Installation

The supported installer is a transactional Bubble Tea application. It targets
the Mango compositor integration in this port and installs the Quickshell
payload without taking ownership of unrelated user files.

## Guided install

```bash
git clone https://github.com/ItzWithTails/illogical-mango.git
cd illogical-mango
./install
```

`./install` builds the installer locally when Go is available. Otherwise it
downloads the matching release binary and verifies its published SHA-256 before
executing it. No script from the network is piped into a shell.

The default `recommended` preset installs the shell, launcher, desktop entries,
user service, and a small marked Mango include. It does **not** copy the broad
GTK, terminal, KDE, and Niri preference set. Choose `full` only if you have
reviewed those dotfiles and want them.

## Review before applying

```bash
./install install --dry-run --yes --no-packages
```

The review reports exact create, replace, stale-removal, preserved-modification,
and unchanged counts. Existing foreign files and locally edited installed files
are preserved by default. To deliberately replace conflicts, select `replace`
in the UI or pass `--conflict replace`.

Important defaults:

- a full system upgrade is off;
- starting the shell is left to the user/session;
- SDDM and other display managers are never changed;
- uninstall never removes packages;
- the installer never performs a hidden `git pull`;
- `NO_COLOR` and `--no-color` are supported;
- English and Russian UI text are selected with `--language` or `LANG`.

## Presets

| Preset | Files | Intended use |
|---|---|---|
| `minimal` | Quickshell payload, launcher, desktop/service files | Existing hand-managed Mango setup |
| `recommended` | Minimal plus reversible Mango integration | Most users |
| `full` | Recommended plus the repository's broad dotfile tree | Dedicated desktop install after review |

## Dependencies

Automatic dependency installation is enabled only on Arch-family systems,
where the repository has a maintained Mango/Quickshell recipe. The exact list
is displayed before confirmation and every requested package is verified after
the package manager exits. A partial package result fails the operation before
home files are changed.

Package-manager transactions cannot be rolled back safely by a dotfiles
installer because packages may be shared. This boundary is stated in review,
recorded in output, and packages are left installed on uninstall.

Fedora, Debian, Ubuntu, NixOS, and other distributions use filesystem-only
installation. Install the equivalents described in [PACKAGES.md](PACKAGES.md)
yourself; the installer does not claim those recipes are verified.

## Lifecycle commands

```bash
./install status
./install update
./install uninstall
./install rollback
```

- `status` checks regular-file hashes and symlink targets without writing.
- `update` installs the current checkout and removes files that disappeared
  upstream, while preserving local edits.
- `uninstall` removes only unchanged installer-owned files and its marked Mango
  include. User modifications remain and are reported.
- `rollback` restores the complete filesystem state before the last committed
  operation, including symlinks and the installation manifest.

Each mutation is preceded by a durable write-ahead journal entry. Failure,
Ctrl+C, or an interrupted previous run triggers reverse-order restoration. A
legacy v1 manifest is migrated inside the same transaction. Every applied
operation also keeps a timestamped transcript under
`~/.local/state/ilmango-v2/logs/`; failure output prints its exact path.

## Safe sandbox and CI

`--root` redirects **all reads and writes** to a filesystem tree and disables
packages and every host command; it is not a pretend chroot.

```bash
root=$(mktemp -d)
./install install --repo "$PWD" --home /home/tester --root "$root" --yes
./install status --home /home/tester --root "$root" --yes
./install uninstall --home /home/tester --root "$root" --yes
```

Non-terminal execution refuses to start unless `--yes` is present. Headless
output is plain text and caps the action listing unless `--verbose` is used.

## After installation

Log out and back into Mango, or start the installed shell explicitly:

```bash
~/.local/bin/ilmango run --daemon
~/.local/bin/ilmango doctor
~/.local/bin/ilmango logs
```
