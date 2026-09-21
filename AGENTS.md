# AGENTS.md

Guidance for AI coding agents working in this repository.

## Overview

`github.com/smallstep/logging` is a small, public (Apache-2.0) Go library that wraps
`go.uber.org/zap` with a request-oriented `Logger`, custom zap encoders (text, JSON,
Common Log Format), an HTTP middleware (`httplog`), gRPC server/client interceptors
(`grpclog`), and a W3C `traceparent` implementation (`tracing`). It has no
Smallstep-internal dependencies; only `zap`, `grpc`, `protobuf`, and `pkg/errors`.

The library is considered legacy. Newer Smallstep code logs with `log/slog`; treat
this repo as maintenance-only and avoid API changes that would break existing
importers (the README warns the API may change, but downstream users exist).

## Commands

```bash
make bootstrap   # install golangci-lint, govulncheck, gotestsum into $(go env GOPATH)/bin
make test        # gotestsum -- -coverpkg=./... -coverprofile=coverage.out ./...
make race        # gotestsum -- -race ./...
make lint        # golangci-lint (config curl'd from smallstep/workflows) + govulncheck; needs network
make fmt         # goimports -w on all .go files
make             # lint test build
```

- `make build` is a no-op (`build: ;`); use `go build ./...` to compile-check.
- Run a single test: `go test -run TestParse ./tracing/`
- No Docker, env vars, `GOPRIVATE`, or submodules are needed. `go build ./...` and
  `make test` both complete in a few seconds. Only `tracing/` has tests today
  (34 cases); every other package reports no test files.
- `coverage.out` is gitignored (`*.out`).

CI (`.github/workflows/ci.yml`) calls the shared `smallstep/workflows` `goCI.yml`
with `only-latest-golang: false` and `run-codeql: true`, so changes must build on
the older Go versions that workflow tests, not just the `go 1.25.0` in `go.mod`.
`actionci.yml` runs actionlint + zizmor on workflow files; `.github/zizmor.yml`
pins the policy (first-party `smallstep/*` actions may use `@main`, third-party
actions need SHA pins).

## Architecture

```
logging/
├── logger.go          # Logger (embeds *zap.Logger), Level, New(), Clone(), Writer()/StdLogger()
├── options.go         # Option funcs; defaults read LOG_FORMAT / LOG_LEVEL
├── context.go         # Tracing() HTTP middleware; traceparent + log-name context helpers
├── encoder/           # zapcore.Encoder impls: text (colored), CLF ("common"), shared object/array encoders
├── httplog/           # Middleware(logger, next, ...Option): per-request access log with
│                      #   optional raw request/response capture and redactors
├── grpclog/           # Unary/stream server interceptors (access log, optional payload logging
│                      #   as protojson), client tracing interceptors that forward traceparent metadata
├── requestid/         # Context key for an externally supplied request ID
├── tracing/           # W3C traceparent: New(), Must(), Parse(), String(), TraceID()
└── examples/httplog.go  # Runnable demo: `go run examples/httplog.go`, then curl :8080
```

### Request flow (HTTP)

`httplog.Middleware` wraps the handler in `logging.Tracing(logger.TraceHeader())`,
which parses or generates a `Traceparent` and stores it in the request context.
`LoggerHandler.ServeHTTP` swaps in a `ResponseLogger` (or `RawResponseLogger` when
`WithLogResponses` is set) to capture status, size, and body, runs any
`RedactorFunc`s, then emits one entry with fixed field names (`name`, `system`,
`request-id`, `tracing-id`, `remote-address`, `duration`, `method`, `path`, `status`,
`size`, ...). Level is chosen by status: `<400` info, `<500` warn, else error. Handlers
can attach `httplog.MessageKey` / `httplog.ErrorKey` via `ResponseLogger.WithField`.

### Request flow (gRPC)

`grpclog.UnaryServerInterceptor` / `StreamServerInterceptor` call `TracingContext` to
read or create the traceparent from incoming metadata (header name is lower-cased),
then `serverLogger.Log` emits an entry keyed by `grpc.package`/`grpc.service`/
`grpc.method` and the status code. `TracingUnaryClientInterceptor` /
`TracingStreamClientInterceptor` append the traceparent to outgoing metadata so it
propagates across hops.

### Encoders

`logging.New` selects the encoder from the `format` option: `text`/`docker` (custom
text encoder), `json`/`k8s`/`kubernetes` (zap JSON), or `common` (CLF). Info and
debug go to stdout, warn and above to stderr, via a `zapcore.NewTee`. The CLF
encoder depends on the exact httplog field names in `encoder/clf_encoder.go`
(`clfFields`); renaming a field in `httplog` breaks CLF output.

## Conventions

- Error wrapping: `github.com/pkg/errors` (`errors.Wrap`, `errors.Errorf`) in the
  root package; `fmt.Errorf` with `%w` in `tracing/`. Match the surrounding file.
- Logging API: structured `zap.Field`s; the `*f` helpers (`Infof`, ...) are
  convenience wrappers around `fmt.Sprintf`. Do not add new dependencies.
- Tests: plain `testing` package with table-driven cases and `reflect.DeepEqual`;
  no testify, no mocks. `tracing/traceparent_test.go` swaps the package-level
  `randReader` to make output deterministic.
- Formatting: `goimports`; golangci-lint config is the shared one from
  `smallstep/workflows`, not checked in here.
- No `go generate` directives and no generated code.

## Environment variables

Read once at `logging.New` time (via `defaultOptions`) and overridable by options:

- `LOG_FORMAT` — `text`, `json` (default), or `common`
- `LOG_LEVEL` — `debug`, `info` (default), `warn`, `error`, `fatal`

## Public repo

This repository is public. Keep commit messages, PR descriptions, and code comments
free of internal service names, customer names, and private hostnames.
