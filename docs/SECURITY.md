# Security

## Trust Model

Web pages are untrusted. Page messages cannot choose a local command, executable, project path, Herdr workspace ID, Native Host name, or shell text. Content scripts forward only a binding ID, allowlisted state, title, URL, timestamp, and short status reason.

Native and IPC messages use versioned JSON envelopes, reject unknown fields and types, and enforce 1 MiB and 64 KiB frame limits respectively. The IPC command set is only `open_binding`, `list_bindings`, `ping`, and `status`.

## Local Execution and Paths

- Quick Actions contain only the trusted executable path and a validated UUID. The executable resolves the URL from local configuration; webpage text never enters PowerShell.
- Project paths are absolute, normalized Windows paths and must match public Herdr data before binding. `worktree.checkout_path` is authoritative; an ordinary workspace may use `pane cwd` only when all valid pane paths agree. Labels, webpage data, common-directory guesses, and private Herdr files are never used to construct a project path.
- URLs allow `https:` plus explicitly bound `http://localhost` or `http://127.0.0.1`. `file:`, `javascript:`, `data:`, `ftp:`, `shell:`, custom schemes, URL credentials, arbitrary executables, and arbitrary shell commands are rejected.
- The named pipe includes the current user's SID and has a protected DACL granting access only to that SID. No TCP listener or public port exists.

## Browser and Native Host

The native manifest is registered under HKCU and has one exact `allowed_origins` entry. The Go host independently checks the launch origin and the extension's `hello` ID. ChatGPT has one explicit host permission; every other origin is optional and requested only during a user binding action. The extension has no Cookie, history, download-history, clipboard-read, or permanent all-sites permission.

## Data and Notifications

Bindings store URL, title, project path/label, adapter, state, sequence, and dedupe metadata. They do not store page bodies, prompts, responses, Cookies, credentials, Tokens, or ChatGPT messages. Logs strip URL queries/fragments and redact common secret fields; binding IDs are hashed where a full ID is unnecessary.

Each completion/error `eventId` is notified once. Initial ChatGPT history cannot produce a completion. Repeated DOM changes do not repeat notifications, and adapter uncertainty becomes `unknown`.

Configuration writes use a same-directory temporary file, flush, atomic Windows replace, and a prior `.bak`. Uninstall preserves bindings and project Quick Actions unless the user explicitly requests removal; even then, Quick Actions require the expected directory and bridge ownership marker.

Prohibited implementation patterns include `Invoke-Expression`, JavaScript `eval`, shell concatenation from page text, unauthenticated `0.0.0.0` services, and page-selected executables or project paths.
