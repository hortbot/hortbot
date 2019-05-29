# hortbot

[![Go Report Card](https://goreportcard.com/badge/github.com/hortbot/hortbot)](https://goreportcard.com/report/github.com/hortbot/hortbot) [![Build Status](https://travis-ci.com/hortbot/hortbot.svg?branch=master)](https://travis-ci.com/hortbot/hortbot) [![Coverage Status](https://coveralls.io/repos/github/hortbot/hortbot/badge.svg?branch=master)](https://coveralls.io/github/hortbot/hortbot?branch=master)

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
