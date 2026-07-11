// Package main is the entry point for the go-app task-manager UI.
//
// The same source tree builds two artifacts:
//
//  1. The WASM client (the actual UI):
//
//     GOOS=js GOARCH=wasm go build -o web/app.wasm .
//
//  2. The dev server that serves the client and reverse-proxies API calls:
//
//     go run . -addr=:8000 -api=http://localhost:8080
//
// When compiled for js/wasm, app.RunWhenOnBrowser() takes over the DOM and the
// server code below is never reached; when compiled for the host it is a no-op
// and the server starts. The server reverse-proxies /api/* to the backend so
// the browser only talks to one origin (no CORS to configure).
package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

func main() {
	app.Route("/", &TaskApp{})
	app.Route("/login", &TaskApp{})

	// In the browser this blocks and drives the UI; on the server it returns.
	app.RunWhenOnBrowser()

	addr := flag.String("addr", ":8000", "address the UI server listens on")
	api := flag.String("api", "http://localhost:8080", "base URL of the task-manager API")
	wasmPath := flag.String("wasm", "web/app.wasm", "path to the compiled wasm binary")
	flag.Parse()

	if _, err := os.Stat(*wasmPath); err != nil {
		log.Printf("warning: %s not found — build it with `GOOS=js GOARCH=wasm go build -o %s .`",
			*wasmPath, *wasmPath)
	}

	apiURL, err := url.Parse(*api)
	if err != nil {
		log.Fatalf("invalid -api url: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(apiURL)

	// The go-app handler serves the SPA shell + the wasm binary. The default
	// resource provider (app.LocalDir("")) serves files from the current
	// directory under /web/, so the wasm is expected at ./web/app.wasm.
	appHandler := &app.Handler{
		Name:        "Task Manager",
		Title:       "Task Manager",
		Description: "A small task manager built with go-app + Echo",
		Styles:      []string{inlineCSS},
	}

	mux := http.NewServeMux()
	mux.Handle("/", appHandler)
	// Any /api/* request is forwarded to the Echo backend so the client can
	// use plain relative URLs without CORS preflight.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		// Preserve the original /api path; NewSingleHostReverseProxy keeps the
		// full path including the /api prefix, which matches the Echo routes.
		r.Host = apiURL.Host
		// Strip hop-by-hop headers and let the proxy set fresh ones.
		r.Header.Del("Accept-Encoding")
		proxy.ServeHTTP(w, r)
	})

	log.Printf("UI server listening on %s (API proxied to %s)", *addr, apiURL)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

const inlineCSS = `
body{margin:0;font-family:system-ui,Segoe UI,Roboto,sans-serif;background:#f5f6fa;color:#1f2330}
.wrap{max-width:760px;margin:0 auto;padding:24px}
h1{font-size:1.4rem;margin:0 0 8px}
.card{background:#fff;border-radius:10px;box-shadow:0 1px 4px rgba(0,0,0,.08);padding:18px;margin:16px 0}
.row{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
input,select,button{font:inherit;padding:8px 10px;border:1px solid #d4d8e0;border-radius:6px;background:#fff}
input{flex:1;min-width:120px}
button{cursor:pointer;background:#3b6ef6;color:#fff;border-color:#3b6ef6}
button.secondary{background:#fff;color:#3b6ef6}
button.danger{background:#e05656;border-color:#e05656}
button:disabled{opacity:.5;cursor:default}
table{width:100%;border-collapse:collapse}
th,td{text-align:left;padding:8px;border-bottom:1px solid #eef0f4}
.muted{color:#6b7280;font-size:.9rem}
.err{color:#c0392b;background:#fdecea;padding:8px 12px;border-radius:6px;margin:8px 0}
.tab{display:flex;gap:8px;margin-bottom:12px}
.badge{font-size:.75rem;padding:2px 8px;border-radius:999px;background:#eef0f4;color:#4b5563}
.badge.done{background:#dcfce7;color:#166534}
.badge.in_progress{background:#dbeafe;color:#1e40af}
`
