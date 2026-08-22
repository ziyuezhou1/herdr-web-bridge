# Environment Audit

Audit date: 2026-08-20 (Asia/Hong_Kong)  
Project: `D:\herdr-web-bridge`

No software was installed, upgraded, removed, or reconfigured during this audit.

## Required Command Results

| Command | Result |
| --- | --- |
| `herdr --version` | Passed: `herdr 0.8.2-preview.2026-08-19-b5c4a0176e91` |
| `herdr status` | Not verified: exit 1, Windows `PermissionDenied` (`拒绝访问`) in the current restricted command session |
| `herdr workspace list` | Not verified: exit 1, Windows `PermissionDenied` |
| `herdr api schema --json` | Passed: JSON Schema draft 2020-12, protocol `20`, schema version `1` |
| `herdr plugin list` | Not verified: exit 1, Windows `PermissionDenied` |
| `herdr plugin action list --plugin cloudmanic.herdr-plus` | Not verified: exit 1, Windows `PermissionDenied` |
| `herdr plugin config-dir cloudmanic.herdr-plus` | Not verified: exit 1, Windows `PermissionDenied` |
| `go version` | Failed: `go` is not installed or is not on `PATH` |
| `node --version` | Passed: `v24.14.1` |
| `npm --version` | Passed: `11.11.0` |
| `git --version` | Passed: `2.37.0.windows.1` |

The permission failures above do **not** prove that Herdr is stopped or that Herdr Plus is absent. They only mean the current command channel could not access the running Herdr session.

## Platform and API Findings

- Windows reports `10.0.26200.0`, x64; PowerShell is `7.4.19`.
- Microsoft Edge is installed at user-visible version `151.0.4129.86`.
- The Edge native host key `HKCU\Software\Microsoft\Edge\NativeMessagingHosts\com.herdr_web_bridge` is not registered.
- The installed Herdr CLI exposes `workspace report-metadata` with `--source`, `--token`, `--clear-token`, `--seq`, and `--ttl-ms`.
- The installed Herdr CLI exposes `notification show` with `--body` and sounds `none`, `done`, and `request`.
- The schema exposes workspace metadata tokens, optional `worktree.checkout_path`, and public pane `cwd` fields. The bridge prefers the checkout path and may use one unambiguous normalized pane cwd for an ordinary workspace; the persisted identity remains the absolute project path.

## Blockers and Consequences

1. Go is a hard build blocker. Go source and tests can be authored, but `go test`, `go vet`, the Windows executable, and the final release ZIP containing that executable cannot be produced until the user installs a supported Go toolchain.
2. Live Herdr calls, workspace matching, metadata updates, and Herdr notifications must be re-tested from a normal user session with Herdr running.
3. Herdr Plus status is unknown. Quick Action generation will be implemented, but its appearance inside Herdr Plus requires manual verification.
4. No Edge extension is installed or connected yet, so native messaging and browser acceptance scenarios remain manual gates.
