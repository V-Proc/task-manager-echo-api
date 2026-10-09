# syntax=docker/dockerfile:1.7
#
# Builds the go-app UI: the WebAssembly client plus the dev/server binary that
# serves it and reverse-proxies /api/* to the api service.
#
# NOTE: this Dockerfile uses the repository root as its build context (so it can
# read go.mod/go.sum), not the ui/ directory. In docker-compose that means:
#   build:
#     context: .
#     dockerfile: ui/Dockerfile

# Build stage: compile both the WASM client and the UI server.
FROM docker.io/library/golang:1.26.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 1. The browser-side client.
RUN GOOS=js GOARCH=wasm go build -trimpath -o /out/app.wasm ./ui
# 2. The server that serves the page + proxies /api to the api service.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/uiserver ./ui

# Minimal runtime. The go-app handler serves static resources from ./web/, so
# the wasm binary must live at /app/web/app.wasm (WORKDIR is /app).
FROM docker.io/library/alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/uiserver /app/uiserver
COPY --from=build /out/app.wasm  /app/web/app.wasm
# Static stylesheet, served by the go-app handler at /web/app.css.
COPY ui/web/app.css /app/web/app.css
EXPOSE 8000
ENTRYPOINT ["/app/uiserver"]