# Herdr Web Bridge

Herdr Web Bridge is a Windows MVP that binds an Edge tab to a Herdr project directory. A binding adds a project-level Herdr Plus Quick Action, reuses an existing tab when possible, and synchronizes web task state to Herdr workspace metadata and notifications.

## MVP Capabilities

- Microsoft Edge Manifest V3 extension with ChatGPT, custom-tool, and manual generic adapters.
- One Go executable for Edge Native Messaging, Herdr CLI calls, binding storage, named-pipe IPC, diagnostics, and URL fallback.
- Project identity based on a normalized absolute path: `worktree.checkout_path` is preferred, with a unique public `pane cwd` fallback for ordinary workspaces. Workspace IDs are resolved again before every Herdr update.
- Per-user, repeatable Windows install/uninstall with exact extension-origin registration.
- No page body, prompt, answer, Cookie, Token, or credential storage.

## Prerequisites

- Windows 10/11 x64, Microsoft Edge, and Herdr on `PATH`.
- Go 1.26 or newer to build the executable; Node.js 22+ for extension tests.
- Herdr Plus is optional. Without it, bindings and state sync work, but project Quick Actions are not generated.

## Build and Test

```powershell
.\scripts\test.ps1
.\scripts\build.ps1
.\scripts\package.ps1
```

The initial [environment audit](docs/environment-audit.md) records the machine state before implementation. Current build and verification results are maintained separately in the [test report](docs/TEST_REPORT.md).

## Install

After a successful build:

```powershell
.\installer\install.ps1
```

Then open `edge://extensions`, enable **Developer mode**, choose **Load unpacked**, and select the exact leaf directory printed by the installer. It must directly contain `manifest.json`. Run the installed executable by full path because installation does not change `PATH`:

```powershell
$bridgeExe = Join-Path $env:LOCALAPPDATA 'Programs\HerdrWebBridge\herdr-web-bridge.exe'
& $bridgeExe doctor
```

See [Windows Quick Start](docs/QUICKSTART_WINDOWS.md), [Architecture](docs/ARCHITECTURE.md), and [Security](docs/SECURITY.md) before live use.
