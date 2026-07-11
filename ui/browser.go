//go:build js && wasm

// This file is intentionally empty. It exists only to tag the package as
// browser-capable when building with GOOS=js GOARCH=wasm. The server build
// (default) ignores it via the build constraint above.
package main
