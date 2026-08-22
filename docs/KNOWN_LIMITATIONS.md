# Known Limitations

- The MVP targets Windows x64 and one active Microsoft Edge profile. Chrome compatibility is not an acceptance target; multiple profiles and simultaneous browsers are unsupported.
- A workspace is selectable when Herdr exposes either `worktree.checkout_path` or one unambiguous `pane cwd`. A plain workspace with no cwd, an invalid path, or panes in multiple directories remains unavailable; private Herdr state files and label-based guesses are never used.
- A pane-derived project identity can become temporarily unavailable if the workspace's panes later move to different directories. Returning them to one project directory restores resolution without persisting a workspace ID.
- ChatGPT support targets `https://chatgpt.com`. DOM changes may reduce adapter health to `unknown`; the bridge intentionally avoids guessing.
- The generic adapter has no automatic task detection. It supports binding, opening, and manual state controls only.
- Custom tools must use the documented same-origin `window.postMessage` schema and receive per-origin Edge permission.
- Chromium host match patterns are host-scoped rather than port-scoped. A localhost permission can cover multiple ports, but the content script stays inert unless the exact URL resolves to a trusted binding.
- Herdr Plus is optional. If absent or inaccessible, bindings and status sync remain available but Quick Actions are not generated.
- The named-pipe broker normally lives as long as Edge keeps the Native Messaging connection. If the extension is disconnected, `open --binding` falls back to the Windows default browser rather than guaranteeing Edge.
- Metadata is display-only and may need resynchronization after Herdr restarts. The bridge retries pending final state on native reconnect, not indefinitely in the background.
- The installer cannot silently install an unpacked Edge extension. The user must enable Developer mode and load the printed directory.
- The audited development machine lacks Go, so the executable, Go verification, live Edge/Herdr acceptance, and release ZIP are currently blocked. See `docs/TEST_REPORT.md`.
