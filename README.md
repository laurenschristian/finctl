# finctl

[![CI](https://github.com/laurenschristian/finctl/actions/workflows/ci.yml/badge.svg)](https://github.com/laurenschristian/finctl/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

CLI and MCP server for markets and macro data One static binary: a CLI and an [MCP](https://modelcontextprotocol.io) server for free market, rates and macro data plus portfolio review.

> Status: planning + scaffold. See [PLAN.md](PLAN.md) for the full command set, data providers and build order, and [docs/data-sources.md](docs/data-sources.md) for ~60 verified free data sources. `finctl doctor` and `finctl mcp` work today.

## Install

```console
brew install laurenschristian/tap/finctl
go install github.com/laurenschristian/finctl@latest
```

## Configure

Precedence is flags, then environment (`FIN_URL`, `FIN_USER`, `FIN_PASS`, `FIN_CONFIG`), then the config file
(`~/Library/Application Support/finctl/config.yaml` on macOS, `~/.config/finctl/config.yaml` on Linux).
`password_cmd` runs any command that prints the secret, so it can live in a keychain, `op read`, `pass` or sops.

## MCP

```console
claude mcp add fin -- finctl mcp
```

## Development

```console
make hooks    # gofmt, dash check, gitleaks, build, lint on commit; tests + coverage floor on push
make test
make lint
make docs     # regenerate man/ and docs/cli/
```

See [CONTRIBUTING.md](CONTRIBUTING.md). MIT.
