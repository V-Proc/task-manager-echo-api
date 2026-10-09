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
	// directory under /web/, so the wasm is expected at ./web/app.wasm and the
	// stylesheet at ./web/app.css. Handler.Styles takes *paths* (not raw CSS),
	// which go-app renders as <link rel="stylesheet" href="/web/app.css"> and
	// caches by default — more idiomatic than inlining via RawHeaders.
	appHandler := &app.Handler{
		Name:        "Task Manager",
		Title:       "Task Manager",
		Description: "A small task manager built with go-app + Echo",
		Styles:      []string{"/web/app.css"},
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
