# Architecture

## Components and Trust Boundaries

```text
bound page content
  → content script
  → MV3 service worker
  → Edge Native Messaging framing
  → herdr-web-bridge.exe
  → Herdr CLI

Herdr Plus Quick Action
  → CLI open --binding <UUID>
  → per-user named pipe
  → connected service worker
  → focus matching tab or create one
```

The content script never calls Native Messaging. The service worker owns the native port, reconnect logic, tab operations, optional origin permissions, and Edge notification fallback. The Go process owns trusted configuration, path and URL validation, Herdr calls, notification deduplication, and Quick Action generation.

The Go host resolves `herdr.exe` from `PATH`, then falls back to `%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe`. This avoids relying on the environment inherited by an Edge process that was started before Herdr was added to `PATH`.

## Executable Modes

- **CLI:** `doctor`, `list-bindings`, `list-workspaces`, `open`, `notify-test`, and `install-status`.
- **Native host:** detected from Edge's `chrome-extension://…/` first argument, with framed JSON on stdout and diagnostics on stderr only.
- **IPC broker:** normally hosted inside the connected native-host process; the explicit `broker` mode is diagnostic.

The Windows named pipe includes the current user's SID, uses an explicit protected DACL for that SID, limits frames to 64 KiB, and accepts only `open_binding`, `list_bindings`, `ping`, and `status`.

## Binding and Workspace Resolution

`%LOCALAPPDATA%\HerdrWebBridge\bindings.json` is schema-versioned and replaced atomically after a `.bak` snapshot. A binding stores the project path, metadata-only page identity, state, event dedupe key, and monotonic sequence. It never stores a workspace ID.

Before each metadata update, the bridge calls `herdr workspace list`. A valid `worktree.checkout_path` is authoritative. When an ordinary workspace has no worktree path, the bridge also calls the public `herdr pane list` API and accepts `cwd` only when every valid pane for that workspace normalizes to the same absolute directory. It never derives a path from the label, a common ancestor, or private Herdr files.

The popup and `list-workspaces` expose `pathSource` (`worktree` or `pane_cwd`) and a bounded `pathReason` when unavailable. Matching uses the resolved normalized path, then focused workspace and label for duplicate workspaces. Any remaining conflict returns `ambiguous_workspace`.

## State and Failure Behavior

ChatGPT begins with a history baseline, requires explicit running signals, a new assistant DOM structure, and 800 ms stable completion. It becomes viewed only after 1500 ms of visible focus. Insufficient selectors produce `unknown`.

Metadata retries are capped at three immediate attempts plus three bounded delayed attempts. Failed Go-side state remains `syncPending`; a disconnected extension also keeps only the latest pending state per binding and replays it after a successful native handshake. Notification markers are persisted before side effects, so a completion event ID is delivered at most once. Edge notifications are used when Herdr/native delivery is unavailable and replay tells Go not to notify the same event again.
