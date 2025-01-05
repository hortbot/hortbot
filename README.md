# hortbot

[![CI](https://github.com/hortbot/hortbot/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/hortbot/hortbot/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/hortbot/hortbot/branch/master/graph/badge.svg)](https://codecov.io/gh/hortbot/hortbot)

A chat bot for Twitch.

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

-   Go 1.23+

For development:

-   Docker (for tests and model generation)
-   [golangci-lint](https://github.com/golangci/golangci-lint) (for linting; also run in CI)
