# ai-token-checker

`ai-token-checker` is a small, read-only CLI that shows the 5-hour and weekly
quota usage available to Claude Code and Codex on macOS and Linux.

Claude usage is fetched live from Anthropic. Codex usage comes from the newest
rate-limit snapshot in the local Codex session cache, so its output always shows
an `as of` timestamp and can be stale. Checking Codex this way does not consume
quota.

## Install

With Go 1.22 or newer:

```sh
go install github.com/kter/ai-token-checker@latest
```

Prebuilt macOS and Linux binaries for amd64 and arm64 are also available from
GitHub Releases.

## Usage

```text
Usage: ai-token-checker [flags]

Flags:
  -h, --help
        show help and exit
  -json
        emit machine-readable JSON
  -no-color
        disable ANSI colors
  -tool string
        show only claude or codex
  -version
        print version and exit
```

Both one-dash and two-dash flag forms accepted by Go's standard flag parser are
supported, including `--json`, `--no-color`, and `--tool codex`. Color is also
disabled when stdout is not a terminal or when `NO_COLOR` is set.

The command exits successfully when at least one selected tool returns quota
data. Missing credentials or cached data for one tool does not prevent the
other from rendering. It exits non-zero when none of the selected tools can be
read, or when flag usage is invalid.

## Data and security notes

- On macOS, Claude's OAuth credential is read from the `Claude Code-credentials`
  Keychain item. On Linux, it is read from `~/.claude/.credentials.json`.
- Credentials and local session files are only read. This tool never writes them
  back and never refreshes OAuth tokens.
- Claude quota uses an unofficial Anthropic endpoint. The endpoint and response
  format may change without notice.
- Codex quota is read from the newest
  `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` file and may not reflect usage
  since that snapshot was recorded.

## Development

```sh
go test ./...
go vet ./...
go build ./...
```

The core uses only the Go standard library. TTY detection uses a small
platform-specific ioctl check, so there are no third-party dependencies.
