# finctl PRD 05: Portfolio and personal

Status: draft, 2026-09-14. Depends on PRD 00, PRD 01, and ibkrctl (gateway on :5001). Providers: ibkr (via the running gateway), monarch, vault, plus quote/earnings/insider/short for `port review`.

## Purpose

My actual book in one command, reconciled to the vault targets, so allocation drift and buy-zones are computed, not eyeballed. This is where finctl meets the financial-advisor workflow.

## Commands

- `port` *: positions, P&L, allocation vs target, drift flags, cash. Positions and P&L come from the IBKR Client Portal gateway on `https://localhost:5001/v1/api/*` (ibkrctl runs the daemon; finctl reuses it, it does NOT log in). Targets are parsed from the vault Dashboard table. Drift = actual weight minus target weight, flagged past a threshold.
- `port review` *: `port` plus earnings-this-week for holdings, insider/short changes since the last review, and a paste-ready review block for the vault note. Read-only; `--write` (later) appends to the note.
- `watchlist` *: the vault watchlist note with live price, distance to buy-zone, next earnings date, and IV rank. Buy-zones parsed from the note.
- `networth`: Monarch balances + IBKR + crypto (coingecko) rollup, one number with a breakdown.

## Vault parsers (internal/vault)

Read-only. Parse: the Dashboard target table (ticker -> target weight), the watchlist note (ticker -> buy-zone range + thesis line), and the tracker CSV (transaction log). Markdown-table and CSV parsers with fixtures; tolerant of column reordering. finctl never writes the vault by default; `--write` is a later, explicit flag.

## Redaction

Account numbers and holder names from the IBKR gateway are redacted in output exactly as ibkrctl does (alias map + blanked name keys). The agent sees `account-1`, never the real Uxxxx. Reuse the ibkrctl redaction approach so both tools behave identically.

## Data model

`Position{Symbol, Qty, Price, Value, CostBasis, UnrealPnL, Weight, TargetWeight, Drift}`. `NetWorth{Total, Buckets []{Name, Value, Source}}`. `WatchRow{Symbol, Last, BuyLow, BuyHigh, DistanceToBuyPct, NextEarnings, IVRank}`.

## Output

`port` is a positions table with a Target and Drift column and a cash line; drift past threshold is flagged in the text. `watchlist` sorts by distance-to-buy ascending (closest to a buy first). `--json` everywhere.

## MCP tools

`fin_port`, `fin_port_review`, `fin_watchlist`, `fin_networth`. Read-only.

## Dependencies and failure modes

- Gateway down or logged out: `port` returns a clear "IBKR gateway not authenticated: run `ibkrctl login`" message, not a stack trace. finctl checks `iserver/auth/status` first.
- Monarch token missing: `networth` degrades to IBKR + crypto and notes the missing bucket.
- Vault path: configurable; a missing Dashboard table means `port` still shows positions but no drift, with a hint.

## Acceptance criteria

1. `finctl port` shows positions with target weights from the vault and computed drift, reusing the live gateway.
2. `finctl watchlist` lists vault watch names sorted by distance to buy-zone with next earnings.
3. Personal ids are redacted identically to ibkrctl.
4. Vault parsers have fixture tests; gateway/monarch calls are integration-only (skipped without a live endpoint).
