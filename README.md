# hortbot

[![](https://github.com/hortbot/hortbot/workflows/CI/badge.svg?branch=master)](https://github.com/hortbot/hortbot/actions?query=workflow%3ACI+branch%3Amaster)
[![codecov](https://codecov.io/gh/hortbot/hortbot/branch/master/graph/badge.svg)](https://codecov.io/gh/hortbot/hortbot)
[![Go Report Card](https://goreportcard.com/badge/github.com/hortbot/hortbot)](https://goreportcard.com/report/github.com/hortbot/hortbot)

An IRC bot for Twitch.

## Features

-   Custom commands
-   Repeated / scheduled commands
-   Moderation and filters
-   Quotes
-   Variables
-   LastFM, Steam integration, and more

### Cool new stuff

-   Zero-downtime updates
-   Multi-domain website
-   OAuth token management for both users and bot instances
-   A real command parser (instead of ordered string replacements)
-   Improved URL filtering

## Credits

-   endsgamer, for the original CoeBot codebase.
-   oxguy3, for the original CoeBot website.

## Requirements

To build:

-   Go 1.13+

For development:

-   Docker (for tests and model generation)
-   [golangci-lint](https://github.com/golangci/golangci-lint) (for linting; also run in CI)
