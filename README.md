# hortbot

An IRC bot for Twitch. Very work in progress.

You can find a todo list in [TODO.md](TODO.md).

## Requirements

To build:

- Go 1.11+

For development:

- [gobin](https://github.com/myitcv/gobin) (for `go generate` and model generation)
- Docker (for tests and model generation)
- `sh` (for model generation)
- [golangci-lint](https://github.com/golangci/golangci-lint) (for linting)

All tools used for `go generate` and model generation run through `gobin`, and
are versioned in `go.mod` like other dependencies.

This project expects to be used in module mode, which means (as of Go 1.12)
cloning this repo outside of `GOPATH`, or in `GOPATH` with `GO111MODULE=on`.
The former is easier.
