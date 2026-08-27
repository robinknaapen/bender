# bender

A multi-service messaging browser — WhatsApp, Slack, Discord and friends,
each in its own isolated OS webview inside one window. Like Rambox or
Ferdium, but with blackjack and hookers. And without Electron.

- **Go, no cgo.** The whole binary cross-compiles from Linux/WSL2 with
  `GOOS=windows`. WebView2 is driven through hand-written pure-Go COM
  bindings (`internal/platform/win/webview2`).
- **The OS webview.** Windows first (WebView2); the platform layer is a
  small neutral interface (`internal/platform`), so WebKitGTK and
  WKWebView backends can follow.
- **Sessions are isolated** per service via WebView2 profiles under one
  user data folder.
- **The shell UI is Go too.** The sidebar is rendered server-side-style
  with [loom](https://github.com/pietjan/loom) — a pure function from
  state to HTML, pushed into a chrome webview over the WebView2 message
  bridge. The only JavaScript in the repo is ~25 lines of glue.
- **Config lives in SQLite** (modernc.org/sqlite + sqlc): services,
  window geometry, active service.

## Milestone 1

One window, sidebar switching, per-service unread badges (title
sniffing), system tray with close-to-tray, and web-notification →
native-notification passthrough.

## Building

Needs Go, the Tailwind standalone CLI (≥ 4.1) on PATH, and a checkout of
loom next to this repo (`replace` directive) until loom is tagged.

```sh
make help      # list targets
make ui        # regenerate templ/sqlc code and the embedded stylesheet
make test      # unit tests (all app logic is platform-neutral and pure)
make audit     # tidy-diff, verify, vet (linux+windows), govulncheck, cross-build
make build     # bin/bender.exe (windowsgui)
make run       # build and launch — works from WSL2 via interop
```

`bin/bender-debug.exe` (`make build/debug`) keeps a console attached and
enables DevTools with `-debug`.

Runtime files land in `%LOCALAPPDATA%\bender` (browser profiles, log)
and `%AppData%\bender` (bender.db).
