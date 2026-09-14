# finctl PRD 00: Overview and architecture

Status: draft, 2026-09-14. Owner: LC. Supersedes the loose plan in `../PLAN.md` for scope and build order.

## Problem

Market, macro, and research work today is a patchwork: the `/invest` skill (300 lines of prose telling the agent which MCP to call), the `serenity` skill (a 15-point checklist run from memory), the Finnhub MCP, the IBKR MCP, and ad hoc WebSearch for prices. There is no single, cache-backed, agent-usable interface. Data lives behind a dozen APIs with different auth, rate limits, and shapes.

## Goal

One Go binary, `finctl`, that wraps every free finance/macro/edge data source behind one interface with a local SQLite cache, prints human tables by default and `--json` for the agent, and exposes the same operations as an MCP server (`finctl mcp`). The Obsidian vault stays the source of truth for targets and theses; finctl reads it and writes nothing by default.

Non-goals: backtesting, live streaming, order placement (ibkrctl owns that), anything paid.

## Success criteria

1. `finctl daily` produces a one-screen morning brief (portfolio, watchlist buy-zones, earnings this week, macro prints today, Fed odds, insider clusters) with `--json` for the agent.
2. The `/invest` skill shrinks to "run these finctl commands, then write the note."
3. The `serenity` lens computes every checklist item that is data.
4. Every provider call goes through the cache with a per-provider TTL; a cold run and a warm run differ only in latency.
5. Coverage floor 60 (providers are integration code; command/model/cache layers carry the tests).

## Architecture

```
main.go
internal/
  config/     flags > env > ~/Library/Application Support/finctl/config.yaml; keys via *_cmd (Keychain)
  cache/      SQLite (modernc.org/sqlite, no cgo): get(provider,key,ttl) -> bytes; per-provider rate guard
  httpx/      shared GET/POST: UA header, timeout, retry, ctx; every call routed through cache
  provider/   one package per source (see PRD 01); each returns normalized model types
  model/      Quote, Fundamentals, InsiderTx, ShortInterest, OptionsSummary, MacroPrint, Position, ...
  vault/      read-only markdown/CSV parsers for the Dashboard target table, watchlist note, tracker
  lens/       Serenity checklist scoring from model data
  cli/        thin cobra commands; emit() = table default, --json raw; redaction of personal ids
  mcp/        thin MCP tools over the same command functions
```

Rules that hold across the tool:
- Providers never print. They return model types or raw bytes; the cli/mcp layer renders.
- Every outbound call is `httpx.Get(ctx, provider, url, ttl)`; httpx owns the cache lookup, the UA, and the per-provider rate limit. No provider calls `http` directly.
- Keys resolve through `password_cmd` style shell commands (Keychain), never plain text in config. Env overrides: `FINCTL_FRED_KEY`, `FINCTL_EIA_KEY`, `FINCTL_BLS_KEY`, `FINCTL_FINNHUB_KEY`.
- Output: tables by default; `--json` emits the normalized model as JSON. Personal account numbers and holder names are redacted in output the way ibkrctl does it (alias map + blanked name keys), reusing that pattern.
- MCP tools are read-only. Anything that could place an order lives in ibkrctl, not here.

## Cache and rate limits

SQLite table `cache(provider TEXT, key TEXT, body BLOB, fetched_at INTEGER, PRIMARY KEY(provider,key))`. `get(provider, key, ttl)` returns the cached body when `now - fetched_at < ttl`, else fetches, stores, returns. TTL per provider (see PRD 01; e.g. quotes 60s, XBRL facts 24h, COT weekly, FRED series 6h). A per-provider token-bucket guards the documented rate limit (SEC 10 req/s is the tightest hard cap). `finctl cache clear [provider]` wipes it.

## Build order

1. Foundation: `cache` (SQLite), `httpx`, `config` keys, `model` types. Ship nothing yet.
2. Keyless core providers + commands (PRD 01, PRD 02 core): edgar, cboe, yahoo, coingecko, frankfurter -> `quote`, `chart`, `fund`, `capex`, `crypto`, `fx`. Ship v0.1.0.
3. Macro (PRD 03): fred, bls, eia, treasury/fed h15, kalshi, polymarket, cftc -> `rates`, `curve`, `series`, `macro brief`, `fedodds`, `calendar`, `cot`, `energy`, `fiscal`. Ship v0.2.0.
4. Filings/positioning (PRD 02 rest): finra, edgar form4/full-text, openinsider, disclosedcapitol -> `insider`, `short`, `congress`, `filings`, `edgar search`, `dilution`, `margins`. Ship v0.3.0.
5. Edge (PRD 04): twse, siliconanalysts, vastai -> `tw-revenue`, `silicon`, `gpu-rent`. Ship v0.4.0.
6. Portfolio/personal (PRD 05): vault parsers, ibkr gateway (:5001 via ibkrctl), monarch -> `port`, `watchlist`, `networth`, `port review`. Ship v0.5.0.
7. Research composites + lens (PRD 02/PRD 07): `research`, `lens`, `daily`, full `mcp`. Ship v1.0.0. Rewrite `/invest` to call finctl.
8. Voices (PRD 06): fxtwitter + Chrome CDP reader. Ship v1.1.0.

## Toolkit

Same as the other laurenschristian Go CLIs (see `[[personal-go-cli-toolkit]]`): Go 1.27, cobra, official MCP go-sdk, goreleaser to Homebrew cask, MIT, public. `.golangci.yml`, coverage floor script, `.githooks`, CI copied from ibkrctl. Add `modernc.org/sqlite` (no cgo) as the one new dependency.
