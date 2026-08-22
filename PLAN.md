# Herdr Web Bridge Windows MVP Plan

Last updated: 2026-08-21

## Decisions and Scope

- Target Windows 10/11 x64 and one active Microsoft Edge profile.
- Use a Manifest V3 extension plus one Go executable with CLI, Native Messaging host, and per-user named-pipe broker modes.
- Persist bindings by normalized absolute project path, never by a permanent Herdr workspace ID.
- Use strict public workspace paths: prefer `worktree.checkout_path`; for ordinary workspaces accept only one unambiguous normalized `pane cwd`. Never infer paths from labels or private state. Missing/ambiguous paths remain unavailable, and duplicate workspace matches return `ambiguous_workspace`.
- Give the unpacked extension a stable manifest `key`; installation validates the derived ID and accepts `-ExtensionId` when needed.
- Do not install or upgrade Go, Herdr, Herdr Plus, or Edge. Do not push to GitHub.

## Planned Structure

```text
cmd/herdr-web-bridge/       Go entry point
internal/                   bindings, Herdr, IPC, native protocol, security
extension/                  MV3 source, popup/options UI, adapters, tests
installer/                  repeatable per-user install/uninstall scripts
examples/                   custom web-tool integration sample
scripts/                    build, test, and package entry points
docs/                       architecture, security, setup, audit, reports
dist/ and release/          generated artifacts only
```

## Implementation Phases

### Phase 0 — Audit and contracts

- [x] Run every required environment command without changing the machine.
- [x] Record versions, permission failures, API capabilities, and blockers in `docs/environment-audit.md`.
- [x] Freeze message envelopes, state names, size limits, URL policy, and persistent data model before transports are built.

Acceptance: every failed check is reported as failed or unverified; no dependency is silently installed.

### Phase 1 — Go bridge core

- [x] Implement versioned binding storage with validation, backup, and atomic replace.
- [x] Implement path/URL normalization, log redaction, UUID lookup, notification deduplication, and monotonic per-binding sequence numbers.
- [x] Parse `herdr workspace list` JSON; prefer normalized checkout paths and safely fall back to a unique public pane cwd; match current workspaces by path, focused state, then label.
- [x] Construct metadata and notification invocations as argument arrays without a shell.
- [x] Generate/remove bridge-owned project Quick Action TOML files safely.
- [x] Implement framed Native Messaging with a 1 MiB host-frame limit and schema/type allowlists.
- [x] Implement a 64 KiB, current-user-only Windows named pipe with `ping`, `status`, `list_bindings`, and `open_binding`.
- [x] Implement CLI commands: `doctor`, `list-bindings`, `list-workspaces`, `open`, `notify-test`, and `install-status`, plus internal `broker` and `native-host` modes.
- [x] Resolve the Herdr executable from `PATH` with a safe fallback to the standard per-user install path so an already-running Edge process does not inherit a stale CLI path.

Acceptance: Go unit tests cover all boundaries listed in `idea.md`; the process never sends logs to Native Messaging stdout.

### Phase 2 — Edge extension

- [x] Add minimal permissions, explicit ChatGPT host access, and optional host permissions for custom tools.
- [x] Route all page events through content script → service worker → native host.
- [x] Implement the popup workflow for workspace selection, bind/unbind, state tests, and workspace focus.
- [x] Implement reliable native-host reconnect and notification fallback.
- [x] Add isolated `chatgpt`, `custom-tool`, and `generic` adapters.
- [x] Enforce the ChatGPT baseline/running/stable-completion/viewed state machine without reading message bodies.
- [x] Validate custom `postMessage` origin, source, version, type, state, and length limits.

Acceptance: Node's built-in lightweight test runner covers adapters, state transitions, validation, reconnect, disconnect, and notification fallback without downloading packages.

### Phase 3 — Install, documentation, and packaging

- [x] Add idempotent per-user install/uninstall/rollback scripts and HKCU Native Messaging registration.
- [x] Make repeat installation safe while Edge is running by staging the executable, suspending host registration, stopping only exact-path bridge processes, and restoring registration on failure.
- [x] Keep binding data and project files by default on uninstall; remove generated Quick Actions only with explicit consent.
- [x] Add architecture, Windows quick-start, security, known-limitations, and custom integration documentation.
- [x] Build the Go executable, copy the extension, validate it with Edge's packer, and create `release/Herdr_Web_Bridge_Windows_MVP.zip`.

Acceptance: installer records every mutation, validates exact extension origin, and needs no administrator rights.

### Phase 4 — Verification

- [x] Run `go test ./...` and `go vet ./...` successfully.
- [ ] Run `go test -race ./...` successfully (`-race` is currently blocked because this Windows Go installation has CGO disabled).
- [x] Run extension and mock integration tests through `scripts/test.ps1` (including pane-cwd source/reason display, inspectable pathless/empty workspace selection, and Edge timer receiver binding).
- [ ] Manually execute scenarios A–E from `idea.md` on Edge with Herdr and Herdr Plus.
- [ ] Record exact results, failures, and environment limitations in `docs/TEST_REPORT.md`.

MVP is complete only when all automated and manual gates pass and the ZIP contains a tested executable. Missing Go or inaccessible live Herdr must be reported as a blocker, not treated as success.

## Key Protocol and Security Rules

- Native messages use UTF-8 JSON with native-endian uint32 length prefixes; stdout contains frames only.
- Native requests are limited to `hello`, `list_workspaces`, `list_bindings`, `bind_page`, `unbind_page`, `report_status`, `focus_workspace`, and `ping`.
- URLs allow HTTPS and explicitly bound localhost HTTP only. `file:`, `javascript:`, `data:`, `ftp:`, `shell:`, custom schemes, arbitrary executables, paths, and commands are rejected.
- Bindings store metadata only: no page body, prompt, response, Cookie, credential, or token.
- Quick Actions call the trusted executable with a UUID; URLs never enter PowerShell command text.
- Completion notifications are keyed by event ID and emitted at most once.

## Risks and Mitigations

- ChatGPT DOM drift: require multiple signals and degrade to `unknown`.
- MV3 worker suspension: reconnect on startup/disconnect and keep authoritative state in the local bridge.
- Herdr unavailable: use bounded exponential retry, persist final state, and avoid duplicate notifications after recovery.
- Workspace IDs changing: resolve from project path before every Herdr update.
- Plain workspace pane directories changing: use a pane cwd only when all valid panes agree; expose the source and reject multi-directory ambiguity.
- Extension ID mismatch: fail installation with a diagnostic unless an explicit verified ID is supplied.
- The standard Go installation is present but the current PowerShell process has stale `PATH`; build/test scripts also resolve the standard installation path without changing user configuration.
- Edge unpacked-load errors can hide directory-selection or policy details: installation now runs Edge's own pack validation and prints the exact leaf directory containing `manifest.json`.
