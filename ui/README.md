# Task Manager UI (go-app / WebAssembly)

A single-page task manager frontend built with [go-app](https://go-app.dev/).
It compiles to a WebAssembly binary that drives the DOM directly from Go, and a
small companion HTTP server serves the page plus a reverse-proxy to the Echo
API so the browser only ever talks to one origin (no CORS to configure).

## Build

From this directory:

```sh
# 1. Build the WASM client.
GOOS=js GOARCH=wasm go build -o web/app.wasm .

# 2. Run the UI server (proxies /api/* to the Echo backend by default).
go run . -addr=:8000 -api=http://localhost:8080
```

Then open <http://localhost:8000>. Make sure the API server (the Echo app in
`../`) is running so `/api/*` has something to forward to.

## Layout

| File        | Role                                                        |
| ----------- | ----------------------------------------------------------- |
| `main.go`   | Route registration + dev server + `/api/*` reverse proxy    |
| `app.go`    | `TaskApp` component: state, lifecycle, handlers, networking |
| `render.go` | `Render()` and view helpers (auth form, task table, etc.)  |
| `browser.go`| Build-constraint marker for the `js/wasm` client build      |

## How it works

- The same `main.go` produces both artifacts. `app.RunWhenOnBrowser()` blocks
  on the client (driving the DOM) and is a no-op on the server, where the HTTP
  server starts instead.
- Auth state is a JWT stored in `localStorage`; on mount the app restores it and
  fetches the user's tasks.
- Input fields are bound to component fields via `Compo.ValueTo`, and state
  mutations from background goroutines go through `ctx.Dispatch` so re-renders
  happen safely on the UI goroutine.
- All API calls use relative `/api/...` paths. The dev server forwards those to
  the Echo backend via `httputil.NewSingleHostReverseProxy`.