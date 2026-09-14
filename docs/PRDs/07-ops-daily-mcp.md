# finctl PRD 07: Ops, the daily brief, and MCP

Status: SHIPPED v1.0.0 (2026-09-14), the capstone. Delivered: daily (six sections with per-section error isolation: Book, Buy-zones in range, Earnings this week, Macro today, Fed odds, Insider clusters; --json returns {section: data|error}; keyless sections populate while gateway/key sections degrade to one-line reasons), doctor upgraded (cache size + IBKR gateway reachability, no secrets printed), fin_daily MCP tool. 24 MCP tools total. Earnings section is a placeholder (needs FINCTL_FINNHUB_KEY). Follow-up (not code): rewrite the /invest skill to a short runbook calling `finctl daily --json` + `finctl research/lens`.

Original draft: Depends on all prior PRDs; `daily` and full `mcp` are the v1.0.0 capstone. This is where the pieces become one habit.

## Purpose

The reason finctl exists as one tool: a single morning command that answers "what do I need to know before the open," and an MCP surface so the agent runs the same thing. Plus the housekeeping (keys, cache, doctor).

## Commands

- `daily` *: the morning brief in one screen. Composes: `port` (book + drift + cash), `watchlist` buy-zones that are in range, `earnings --week` for holdings and watchlist, `macro brief` prints landing today, `fedodds`, and insider clusters in holdings (`insider --clusters` filtered to my names). `--json` for the agent to summarize into the vault note. This is the command the `/invest` skill collapses into.
- `keys`: which providers are configured (FRED, EIA, BLS, Finnhub present or missing) and where each key resolves from (Keychain cmd vs env). No secrets printed.
- `cache clear [provider]`: wipe the SQLite cache, all or one provider.
- `doctor`: config path, cache path + size, key presence, IBKR gateway reachability (`iserver/auth/status`), and a one-line reachability probe per provider group.
- `mcp` *: MCP server over stdio exposing every read command as a tool.

## The daily brief contract

`daily` must never blank because one input failed. Each section is computed independently and a failure is rendered as a one-line "section unavailable: <reason>" (the ibkrctl `review` pattern). Order: Book -> Buy-zones in range -> Earnings this week -> Macro today -> Fed odds -> Insider clusters. `--json` returns a `{section: data|error}` object so the agent can render or skip per section.

## MCP topology

One stdio server, `finctl mcp`, registered in Claude Code as `fin`. Tools are the union of the read tools from PRDs 02 to 06 plus `fin_daily`, `fin_port`, `fin_macro_brief`. Naming: `fin_<command>`. No write tools; the vault `--write` path, if it ever lands, stays CLI-only and confirm-gated, mirroring ibkrctl's order-placement policy. Output wrapping and personal-id redaction match ibkrctl (`rawOut{Data}`, omitempty slices, alias map).

## Skill rewrite (the payoff)

Once `daily`, `research`, and `lens` ship, rewrite the `/invest` skill to: "run `finctl daily --json`, then `finctl research <T>` / `finctl lens <T>` for anything flagged, then write the vault note." The 300-line prose skill drops to a short runbook. The serenity skill keeps its qualitative checklist but points its data items at `finctl lens`.

## Ops details

- Config: `~/Library/Application Support/finctl/config.yaml`; `FINCTL_CONFIG` override. Keys via `*_cmd` (Keychain), env `FINCTL_*_KEY` overrides.
- Cache: `~/Library/Application Support/finctl/cache.db` (SQLite). `cache clear` and a size line in `doctor`.
- Release: goreleaser to the Homebrew cask, same as ibkrctl; tag `vX.Y.Z` triggers CI. Coverage floor 60.

## Acceptance criteria

1. `finctl daily` renders the six sections with per-section error isolation; `--json` returns the `{section: data|error}` map.
2. `finctl mcp` exposes `fin_daily` and the read tools; a stdio client can list and call them.
3. `finctl doctor` reports key presence, cache size, and gateway reachability without printing secrets.
4. After v1.0.0, the `/invest` skill is a short runbook calling finctl, tracked as a follow-up task.
