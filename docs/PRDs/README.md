# finctl PRDs

Product requirements for finctl, grounded in the provider research in `01`. Read `00` first, then `01`, then the command PRDs in build order.

| # | PRD | Scope |
| --- | --- | --- |
| 00 | [Overview and architecture](00-overview-architecture.md) | problem, goals, layout, cache, build order, toolkit |
| 01 | [Provider catalog](01-providers-data-sources.md) | every data source: verified endpoint, auth, rate limit, TTL |
| 02 | [Research commands](02-research-commands.md) | quote, chart, fund, research, lens, insider, short, options, congress, dilution, margins |
| 03 | [Macro commands](03-macro-commands.md) | macro brief, rates, curve, series, fedodds, cot, energy, calendar, fiscal |
| 04 | [Supply-chain edge](04-edge-supply-chain.md) | capex, tw-revenue, silicon, gpu-rent |
| 05 | [Portfolio and personal](05-portfolio-personal.md) | port, port review, watchlist, networth, vault, IBKR, Monarch |
| 06 | [Voices](06-voices.md) | voices, voices tickers, voice read (fxtwitter + CDP) |
| 07 | [Ops, daily, MCP](07-ops-daily-mcp.md) | daily brief, keys, cache, doctor, mcp, /invest rewrite |

Source research: `../data-sources.md` (prior verification) plus Exa re-confirmation 2026-09-14 recorded in PRD 01. Build order and success criteria live in PRD 00.
