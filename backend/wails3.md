# Wails v3 Desktop & Dual-Mode Transport

Architecture and implementation reference for packaging a Go + Web frontend application as a native **Wails v3** desktop client (Linux GTK4/WebKitGTK, macOS WebKit, Windows WebView2) while keeping 100% code sharing with the browser/server mode.

Read this file when building a Wails v3 desktop app or adding a desktop shell to a project (`cmd/<app>-desktop/`, `internal/ipc/`, `src/ipc/`, `scripts/lib/build-wails.mjs`).

## Core architecture: dual-mode, single codebase

Do not fork the frontend or business services for desktop vs. web:
1. **Shared Go core (`internal/app`)**: Holds all business services, GORM store, Eino agents, and an `ipc.Dispatcher`.
2. **Two entry points**:
   - `cmd/<app>-server`: HTTP/Hertz server exposing `POST /rpc` (or `/api/*`), SSE/WebSocket streams, and embedded static assets.
   - `cmd/<app>-desktop`: Wails v3 `application.App` wrapping the exact same `core.Application` in a single Wails `Service`.
3. **Frontend transport switch (`src/ipc/env.ts`)**:
   - The desktop asset handler injects `<script>window.__APP_TRANSPORT__="desktop";</script>` into `index.html`.
   - If `window.__APP_TRANSPORT__ === "desktop"` or `window.wails !== undefined`, the frontend routes RPCs and streams through `@wailsio/runtime`; otherwise it uses `fetch` / SSE / WebSocket.

---

## 1. Single-method Wails Service + Command Dispatcher

Instead of exposing dozens of Go methods to Wails binding codegen (which bloats generated TypeScript and couples every feature change to `wails generate bindings`), expose **one** RPC envelope method `Call` on the Wails `Service`:

### Go IPC contract (`internal/ipc/`)

```go
package ipc

import (
	"context"
	"encoding/json"
)

type ChannelRef struct {
	ID string `json:"id"`
}

type Request struct {
	Command  string          `json:"cmd"`
	Args     json.RawMessage `json:"args"`
	Channel  ChannelRef      `json:"channel,omitempty"`
	ClientID string          `json:"clientId,omitempty"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Detail  map[string]any `json:"detail,omitempty"`
}

type Response struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error *Error          `json:"error,omitempty"`
}
```

### Wails v3 `Service` wrapper (`cmd/<app>-desktop/service.go`)

```go
package main

import (
	"context"
	"errors"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct {
	app     *core.Application
	streams *desktopStreamFactory
}

func (s *Service) ServiceName() string { return "App" }

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	return s.app.Start(ctx)
}

// Wails runs ServiceShutdown on the main UI thread. Close streams first so
// blocked window dispatches release immediately instead of deadlocking shutdown.
const serviceShutdownTimeout = 5 * time.Second

func (s *Service) ServiceShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), serviceShutdownTimeout)
	defer cancel()
	streamsErr := s.streams.Close()
	appErr := s.app.Shutdown(ctx)
	return errors.Join(appErr, streamsErr)
}

func (s *Service) Call(ctx context.Context, req ipc.Request) ipc.Response {
	clientID := req.ClientID
	if clientID == "" {
		clientID = "desktop"
	}
	return s.app.Dispatcher.Dispatch(ctx, req, s.app.Environment(clientID))
}
```

---

## 2. Three streaming & event patterns in Wails v3

Wails v3 provides global events, window-targeted custom events, and binary `GoStream` connections. Use each for its right job:

### Pattern A: Global broadcast events (`wailsApp.Event.Emit`)

For lightweight state notifications (`session://status`, `sync://status`, `files://dropped`):
- **Go**:
  ```go
  wailsApp.Event.Emit(string(event.Topic), event.Payload)
  ```
- **Frontend (`@wailsio/runtime`)**:
  ```ts
  import { Events } from '@wailsio/runtime'
  const unlisten = Events.On(topic, (msg) => handler(msg.data as T))
  ```

### Pattern B: Per-request point-to-point stream (`WailsChannel` + `DispatchWailsEvent`)

For AI agent run streams (`ai://event`), terminal PTY output, or log tailing initiated by an RPC call:
1. **Frontend creates a `WailsChannel` and waits for `ready` before invoking `Call`**:
   ```ts
   const CHANNEL_TOPIC_PREFIX = 'channel://'
   const DESKTOP_BINARY_TYPE = 'app.desktop-binary'

   export class WailsChannel {
     readonly id: string
     onmessage: (message: unknown) => void = () => {}
     readonly ready: Promise<void>
     private disposed = false
     private off: (() => void) | null = null

     constructor(id: string) {
       this.id = id
       this.ready = this.listen()
     }
     toJSON(): string {
       return this.id
     }
     dispose(): void {
       if (this.disposed) return
       this.disposed = true
       this.off?.()
       this.off = null
     }
     private async listen(): Promise<void> {
       const { Events } = await import('@wailsio/runtime')
       if (this.disposed) return
       this.off = Events.On(`${CHANNEL_TOPIC_PREFIX}${this.id}`, (msg) => {
         if (!this.disposed) this.onmessage(decodeChannelData(msg.data))
       })
     }
   }

   export async function callDesktop<T>(cmd: string, args?: Record<string, unknown>): Promise<T> {
     // Gate only on the channel this call hands over so the listener is registered
     // before the backend starts pushing events on it.
     if (args?.channel instanceof WailsChannel) await args.channel.ready
     const { Call } = await bindings()
     const res = await Call(request(cmd, args))
     if (!res || res.ok !== true) throw res?.error ?? { code: 'internal', message: '调用失败' }
     return res.data as T
   }
   ```
2. **Go dispatches to the window with shutdown release (`cmd/<app>-desktop/streams.go`)**:
   - Send JSON frames directly as `application.CustomEvent{Name: "channel://" + id, Data: payload}`.
   - Send binary frames as `{Type: "app.desktop-binary", Data: base64.StdEncoding.EncodeToString(bytes)}`; on the frontend, decode with `Uint8Array.fromBase64` (falling back to a pre-allocated `atob` loop, which is ~40x faster than `Uint8Array.from(atob(...), map)`).
   - Wrap `window.DispatchWailsEvent(event)` in a goroutine selected against `<-f.closing` so `ServiceShutdown` never hangs if the webview stops draining events while closing.

### Pattern C: High-throughput bidirectional binary stream (`app.HandleStream` / `GoStream`)

For high-bandwidth frame protocols (e.g. remote desktop RDP/VNC frames) that need native WebView backpressure:
```go
app.HandleStream("remotedesktop", func(conn *application.StreamConn) {
	// conn.Receive() reads []byte; conn.Send(data) blocks while the WebView drains,
	// applying natural backpressure instead of dropping frames.
	_ = service.Serve(conn.Context(), streamFrameConn{conn: conn})
})
```

---

## 3. Desktop security: `runtimeRequestGuard` middleware

Wails v3 serves its IPC bridge over `/wails/runtime`, `/wails/stream/*`, and `/wails/eventpayload/*` on the app origin (`wails://localhost` on macOS/Linux, `http(s)://wails.localhost` on Windows/Android). Without a guard middleware, any iframe or untrusted page loaded inside the webview could invoke IPC commands via query strings or cross-origin requests.

Always wrap `AssetOptions.Middleware` with `runtimeRequestGuard` (`cmd/<app>-desktop/runtime_guard.go`):
1. **Forbid framing on every response**: Set `X-Frame-Options: DENY` and `Content-Security-Policy: frame-ancestors 'none'`, and reject any request where `Sec-Fetch-Dest` is `iframe` or `frame`.
2. **Guard `/wails/*` endpoints**:
   - Reject `OPTIONS` preflight requests.
   - Verify `req.Host` matches the platform's canonical Wails host (`wails.localhost` on `windows`/`android`; `localhost` or `wails` on `darwin`/`linux`).
   - Require `x-wails-client-id` on `/wails/runtime` and `x-wails-stream-session` on `/wails/stream/*` (headers a cross-origin document cannot attach without a preflight).
   - Verify `Sec-Fetch-Site` is `""`, `"same-origin"`, or `"none"`.

---

## 4. Asset serving & build tags (`production` vs `!production`)

Keep development builds fast (no embedded `dist` required to run `go test ./...`) and production binaries self-contained:

- **`assets_dev.go` (`//go:build !production`)**: Serves static files from disk (`dist/`), injecting `<script>window.__APP_TRANSPORT__="desktop";</script>` into `index.html`.
- **`assets_production.go` (`//go:build production`)**:
  - Embeds `//go:embed all:dist`.
  - Pre-computes `indexHTML` with the injected transport marker and its SHA-256 `ETag`.
  - Serves `assets/*` with `Cache-Control: public, max-age=31536000, immutable` and `ETag`, and falls back to `index.html` (`Cache-Control: no-cache`) for SPA routes.
  - Rejects path traversal (`..`, backslashes, NUL bytes, volume names) in `desktopAssetPath`.

---

## 5. Desktop lifecycle & native OS polish

### Application options (`cmd/<app>-desktop/main.go`)

```go
appOptions := application.Options{
	Name:        "AppName",
	Description: "Desktop client",
	Services:    []application.Service{application.NewService(service)},
	Assets:      application.AssetOptions{Handler: assets, Middleware: runtimeRequestGuard},
	Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	Linux:       application.LinuxOptions{ApplicationID: "com.example.desktop"}, // matches .desktop file & Wayland app_id
	SingleInstance: &application.SingleInstanceOptions{
		UniqueID: "com.example.desktop",
		OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
			if window != nil {
				showWindow(window)
			}
			if url := findDeepLink(data.Args); url != "" {
				service.links.deliver(url)
			}
		},
	},
}
```

### Window geometry persistence (`window_state.go`)

- Load `window-state.json` (`{x, y, width, height, maximised}`) from the user's `DataDir` before creating the window.
- **Off-screen monitor guard**: Only restore `(X, Y)` if the saved rectangle overlaps at least an `80px` strip of a currently connected screen (`app.Screen.GetAll()`); otherwise keep the size (clamped to `MinWidth`/`MinHeight`) and let the OS center the window.
- Listen to `WindowDidMove`, `WindowDidResize`, `WindowMaximise`, `WindowUnMaximise`, and `WindowRestore` with a `400ms` debounce, plus `WindowClosing`, and write atomically via `window-state.json.tmp` -> `os.Rename`. Skip saving while minimised or fullscreen.

### Native file drop bridge (`native_drop.go`)

Wails v3 delivers `wailsevents.Common.WindowFilesDropped` to Go listeners with real local filesystem paths (`event.Context().DroppedFiles()`) and drop coordinates (`event.Context().DropTargetDetails()`).
- In Go, `os.Stat` each dropped path to classify `{path, dir: info.IsDir()}` (keep unstatable paths as `dir: false` so the upload pipeline surfaces a clear error instead of silently dropping the action) and re-emit `app.Event.Emit("files://dropped", payload)` to the frontend.

---

## 6. Wails CLI bindings & CI drift check (`scripts/lib/build-wails.mjs`)

Never rely on a globally installed `wails3` binary whose version can drift across machines:
1. Declare `github.com/wailsapp/wails/v3` in `go.mod`.
2. Invoke the CLI via `go run github.com/wailsapp/wails/v3/cmd/wails3@latest` (or the version recorded in `go.mod`):
   ```bash
   go run github.com/wailsapp/wails/v3/cmd/wails3@latest generate bindings -ts -i -d <outDir> ./cmd/<app>-desktop
   ```
3. **CI drift check (`bindings:check`)**: Generate bindings into a temporary directory (`os.tmpdir()`) and recursively byte-compare with the committed `src/ipc/bindings/` directory. Fail CI on any diff. Never hand-edit `src/ipc/bindings/**`.
