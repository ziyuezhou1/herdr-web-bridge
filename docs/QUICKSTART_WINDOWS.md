# Windows Quick Start

## 1. Prerequisites

Open PowerShell as the normal Windows user—not as administrator—and verify:

```powershell
herdr --version
go version
node --version
```

Do not continue to packaging if Go is missing. The installer never installs or upgrades dependencies.

## 2. Test, Build, and Install

```powershell
Set-Location D:\herdr-web-bridge
.\scripts\test.ps1
.\scripts\build.ps1
.\installer\install.ps1
```

Installation defaults to `%LOCALAPPDATA%\Programs\HerdrWebBridge` and registers `com.herdr_web_bridge` under the current user's Edge Native Messaging registry key. Every mutation is appended to `%LOCALAPPDATA%\HerdrWebBridge\install.log.jsonl`.

During an upgrade, the installer briefly unregisters the Native Host and stops only `herdr-web-bridge.exe` processes whose executable path exactly matches the selected install directory. This releases Windows' executable lock; Edge may show a short disconnect until installation completes and the extension is reloaded.

The manifest public key should derive extension ID `pphgcjjepkodhghpncncnmikafkdjdjd`. If Edge displays a different stable ID, rerun installation explicitly:

```powershell
.\installer\install.ps1 -ExtensionId <32-character-edge-id>
```

## 3. Load the Edge Extension

1. Open `edge://extensions`.
2. Enable **Developer mode**.
3. Select **Load unpacked**.
4. Choose the exact `edge-extension` leaf directory printed by the installer—not its parent. With the default install it is:

   ```text
   %LOCALAPPDATA%\Programs\HerdrWebBridge\edge-extension
   ```

   The selected folder must directly contain `manifest.json`, `popup.html`, and `src\`.
5. Confirm that the displayed ID equals the installer value.
6. Keep Herdr running, then run the installed executable by its full path (the installer does not modify `PATH`):

   ```powershell
   $bridgeExe = Join-Path $env:LOCALAPPDATA 'Programs\HerdrWebBridge\herdr-web-bridge.exe'
   & $bridgeExe doctor
   ```

If Edge displays **Failed to load extension**, validate the exact installed directory first:

```powershell
$extensionDir = Join-Path $env:LOCALAPPDATA 'Programs\HerdrWebBridge\edge-extension'
Test-Path (Join-Path $extensionDir 'manifest.json')
.\scripts\validate-edge-extension.ps1 -ExtensionDirectory $extensionDir
```

Both commands must succeed. If they do but Edge still rejects the folder, copy the full detail shown under Edge's error heading; the heading alone does not distinguish a manifest error, a browser policy, or selecting the parent directory.

### Empty workspace list

An installed extension still needs the per-user Native Messaging registration and a readable Herdr session. In a normal, non-administrator PowerShell window run:

```powershell
Set-Location D:\herdr-web-bridge
.\installer\install.ps1

reg.exe query "HKCU\Software\Microsoft\Edge\NativeMessagingHosts\com.herdr_web_bridge" /ve
herdr status
herdr workspace list
herdr pane list

$bridgeExe = Join-Path $env:LOCALAPPDATA 'Programs\HerdrWebBridge\herdr-web-bridge.exe'
& $bridgeExe list-workspaces
```

The registry query must point to `%LOCALAPPDATA%\HerdrWebBridge\native-host-manifest.json`. After registering, open `edge://extensions` and select **Reload** on Herdr Web Bridge. A bindable workspace must expose either `worktree.checkout_path` or a single unambiguous absolute `cwd` through `herdr pane list`; the bridge intentionally does not guess paths from workspace labels.

## 4. Bind and Test

Open a ChatGPT conversation or another HTTPS page, select the extension, choose a Herdr workspace with an available project path, and select **绑定**. The selector marks whether the path came from `worktree` or `pane cwd`. Use the four manual status buttons before testing automatic adapters. For a custom localhost tool, Edge asks for that origin only.

Run useful diagnostics with:

```powershell
$bridgeExe = Join-Path $env:LOCALAPPDATA 'Programs\HerdrWebBridge\herdr-web-bridge.exe'
& $bridgeExe list-workspaces
& $bridgeExe list-bindings
& $bridgeExe install-status
& $bridgeExe notify-test
```

## 5. Uninstall

```powershell
.\installer\uninstall.ps1
```

Bindings and project Quick Actions are preserved by default. Remove them only with explicit switches:

```powershell
.\installer\uninstall.ps1 -RemoveBindings -RemoveGeneratedQuickActions
```

The uninstaller never removes Herdr, Herdr Plus, or unrelated project files.

To undo only the most recent installation/upgrade and restore its captured prior executable, extension, config, manifest, and registry value, run:

```powershell
.\installer\rollback.ps1
```
