# bender

A multi-service messaging browser — WhatsApp, Slack, Discord and friends,
each in its own isolated OS webview inside one window. Like Rambox or
Ferdium, but with blackjack and hookers. And without Electron.

- **Go, no cgo.** One codebase cross-compiles to Windows and Linux from
  anywhere. WebView2 is driven through hand-written pure-Go COM bindings; GTK4 +
  WebKitGTK 6.0 through hand-written purego dlopen bindings — both live
  in [spectacle](https://github.com/pietjan/spectacle), the extracted
  webview-shell library.
- **The OS webview.** WebView2 on Windows, WebKitGTK on Linux — behind
  one small neutral interface (`spectacle.Backend`); a WKWebView backend
  can follow. The Linux build needs the runtime libraries only
  (`libgtk-4-1 libwebkitgtk-6.0-4` on Debian/Ubuntu); tray via
  StatusNotifierItem (GNOME needs the AppIndicator extension),
  notifications via org.freedesktop.Notifications.
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

Needs Go and the Tailwind standalone CLI (≥ 4.1) on PATH.

```sh
make help          # list targets
make ui            # regenerate templ/sqlc code and the embedded stylesheet
make test          # unit tests (all app logic is platform-neutral and pure)
make audit         # tidy-diff, verify, vet (linux+windows), govulncheck, cross-build
make build         # both binaries into bin/
make build/windows # bin/bender.exe (windowsgui)
make run/windows   # build and launch — works from WSL2 via interop
make build/linux   # bin/bender (GTK4/WebKitGTK)
make run/linux     # build and launch — works under WSLg too
```

`bin/bender-debug.exe` (`make build/windows/debug`) keeps a console attached and
enables DevTools with `-debug`.

Runtime files land in `%LOCALAPPDATA%\bender` (browser profiles, log)
and `%AppData%\bender` (bender.db).
