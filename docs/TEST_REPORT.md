# Test Report

Last updated: 2026-08-21 (Asia/Hong_Kong)

## Automated Results

| Check | Result | Evidence |
| --- | --- | --- |
| JavaScript syntax check | **Passed** | `scripts/test.ps1`: 24 JavaScript files passed `node --check` |
| Extension unit/fixture tests | **Passed** | `scripts/test.ps1`: 25 tests passed, 0 failed on Node `v24.14.1` |
| PowerShell installer checks | **Passed** | 10 installer/build/test/package scripts parsed in PowerShell `7.4.19`; exact-path process control returned no match for an unrelated/nonexistent target |
| Stable extension ID | **Passed** | Manifest public key derives `pphgcjjepkodhghpncncnmikafkdjdjd` |
| Edge extension artifact | **Passed** | `dist/edge-extension/` built; Edge 151's own packer accepted its Manifest V3 manifest and 9 referenced files |
| Installed extension directory | **Passed (package validation)** | `%LOCALAPPDATA%\Programs\HerdrWebBridge\edge-extension` hash-matches the 19-file artifact and also passes Edge's pack validation |
| Go unit tests | **Passed** | `go test ./...` completed successfully |
| Go race tests | **Blocked** | `go test -race ./...` reports that `-race` requires CGO; not reported as passed |
| Go vet | **Passed** | `go vet ./...` completed successfully |
| Go executable build | **Passed** | `dist/herdr-web-bridge.exe` generated successfully |
| Release packaging | **Passed** | `release/Herdr_Web_Bridge_Windows_MVP.zip` generated; the final SHA-256 is reported with the handoff |

The passing extension suite covers ChatGPT initial history, fresh-chat health, running-to-completion stability, duplicate mutation suppression, watched/viewed transitions, selector-loss `unknown`, DOM fixtures, custom message schema/origin/state/length rejection, Native Host disconnect/reconnect, Edge service-worker timer receiver binding, empty/pathless workspace diagnostics, pane-cwd source and ambiguity display, pending final-state fallback policy, tab reuse, manifest permissions, stable ID derivation, and notification bounds.

The Go suite covers Native Messaging framing, malformed/oversized input, Windows paths, workspace matching/ambiguity, authoritative worktree paths, unique pane-cwd fallback, multi-directory rejection, degraded pane-list failure behavior, atomic binding backup/write, binding lookup, URL policy, Herdr argument construction and bounded retry, standard per-user Herdr executable fallback, Quick Action sanitization/escaping, notification dedupe, globally monotonic binding sequences, IPC allowlists/mock transport, and log redaction.

## Live Integration and Manual Acceptance

| Scenario | Result | Blocker |
| --- | --- | --- |
| A — Bind ChatGPT and create Quick Action | Partial | Edge and Native Messaging now list six live Herdr workspaces; the installed pre-0.1.3 bridge marked all unavailable because their optional worktree data were null. Binding with the new pane-cwd fallback is pending user verification. |
| B — Focus existing tab / open missing tab | Not run | Requires one successful binding with the 0.1.3 build |
| C — ChatGPT completion metadata and notification | Not run | Requires live Edge, Herdr, and a real ChatGPT generation |
| D — Custom tool running/done/error | Not run | Requires installed bridge and live Herdr workspace |
| E — Failure recovery | Not run | Requires a working end-to-end installation |

## Environment-Specific Findings

- Herdr CLI version and API schema were read successfully; `status`, workspace, and plugin operations returned Windows `PermissionDenied` in the restricted development session.
- Herdr Plus presence is therefore unverified, not assumed absent.
- Edge now loads the unpacked extension successfully, and the popup can retrieve the same six workspace records seen from the installed CLI. This confirms the extension-to-Native-Host request path for workspace listing, but not yet binding or status synchronization.
- All six returned records had `pathAvailable: false` because Herdr 0.8.2 supplied `worktree: null` for these ordinary workspaces. Version 0.1.3 now queries the public pane list only for those entries, accepts one normalized cwd, and exposes `pathSource`/`pathReason`; live results remain pending reinstall and reload.
- The current restricted command channel still cannot access the user's live Herdr session or write `%LOCALAPPDATA%`/HKCU. The new build must be installed and verified from the user's normal PowerShell session.
- A repeat installation with the Native Host running exposed a Windows executable lock. The installer now stages and hash-checks the new executable, temporarily suspends Native Host registration, stops only processes with the exact installed path, preserves the previous executable, and restores prior registration in `finally`. Live locked-file replacement remains pending verification in the user's normal PowerShell session because the development policy blocked launching a synthetic executable-lock fixture.
- Extension `0.1.1` fixes Edge's `Illegal invocation` during Native Host reconnect by preserving the browser timer receiver; a receiver-sensitive regression test covers both scheduling and cancellation.
- Extension `0.1.2` kept the workspace selector inspectable even when Herdr returned no bindable path. Extension `0.1.3` additionally displays whether a usable path came from a worktree or unique pane cwd and explains ambiguous/unavailable pane paths.
- Go is installed at the standard Windows location, but the current PowerShell process has a stale `PATH`; the scripts now resolve that standard path without changing user configuration.

MVP completion is **not declared**. Install 0.1.3, reload the exact validated extension directory, confirm that plain workspaces now expose project paths, then execute scenarios A–E. The release ZIP remains a build candidate until those live acceptance gates pass.
