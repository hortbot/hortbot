# hortbot

[![Go Report Card](https://goreportcard.com/badge/github.com/hortbot/hortbot)](https://goreportcard.com/report/github.com/hortbot/hortbot) [![Build Status](https://travis-ci.com/hortbot/hortbot.svg?branch=master)](https://travis-ci.com/hortbot/hortbot) [![Coverage Status](https://coveralls.io/repos/github/hortbot/hortbot/badge.svg?branch=master)](https://coveralls.io/github/hortbot/hortbot?branch=master)

An IRC bot for Twitch.


## Features

- Custom commands
- Repeated / scheduled commands
- Moderation and filters
- Quotes
- Variables
- LastFM, Steam integration, and more


### Cool new stuff

- Zero-downtime updates
- Multi-domain website
- OAuth token management for both users and bot instances
- A real command parser (instead of ordered string replacements)
- Improved URL filtering


## Credits

- endsgamer, for the original CoeBot codebase.
- oxguy3, for the original CoeBot website.


## Requirements

To build:

- Go 1.13+

For development:

- [gobin](https://github.com/myitcv/gobin) (for `go generate` and model generation)
- Docker (for tests and model generation)
- `sh` (for model generation)
- [golangci-lint](https://github.com/golangci/golangci-lint) (for linting)

All tools used for `go generate` and model generation run through `gobin`, and
are versioned in `go.mod` like other dependencies.
