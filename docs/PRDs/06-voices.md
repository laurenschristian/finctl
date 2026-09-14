# finctl PRD 06: Voices (X/Twitter reading)

Status: draft, 2026-09-14. Ships last (v1.1.0). Depends on PRD 00. Providers: fxtwitter, plus a Chrome CDP reader on the Mac mini for timelines. See the plan's `docs/voices.md` for the handle whitelist.

## Purpose

A curated set of market voices, read into the terminal as data: who is saying what about which tickers, and each author's thesis, without opening X. The archive of theses stays a reference doc; finctl pulls the live posts.

## Commands

- `voices` *: the whitelisted handles with follower count, recent post count, and a one-line bio (fxtwitter profile JSON per handle).
- `voices tickers` *: tickers mentioned across the whitelist in the last N posts, ranked by mention count, so a rising name surfaces.
- `voices thesis <handle>`: the stored thesis doc for a handle (reference), plus their latest few posts.
- `voice read <url>`: hydrate a single tweet/thread to text via fxtwitter (`api.fxtwitter.com/<handle>/status/<id>`), including quoted/replied context.

## Data sources

- fxtwitter: `https://api.fxtwitter.com/<handle>` (profile: followers, tweet count, bio) and `.../<handle>/status/<id>` (single post + author + stats) as JSON. Keyless. Primary path for single posts and profiles.
- Timelines: fxtwitter does not give a full timeline. For `voices tickers` (needs recent posts per handle), use a logged-in Chrome via CDP on the Mac mini (the "use the logged-in browser" path that himself65/finance-skills confirms), or a nitter/xcancel mirror when one is reachable. This is the fragile part; isolate it behind the provider interface so a mirror swap is one file.

## Data model

`Voice{Handle, Followers, Posts, Bio}`. `Post{Handle, ID, Text, Time, Likes, Reposts, Tickers []string, QuotedText}`. Ticker extraction = `$CASHTAG` regex plus a known-symbol match.

## Output

`voices` and `voices tickers` are tables; `voice read` prints the hydrated text block. `--json` everywhere.

## MCP tools

`fin_voices`, `fin_voices_tickers`, `fin_voice_read`. Read-only.

## Risk and constraints

- X actively breaks scrapers; the timeline path will rot. Keep it behind one provider file, cache aggressively (15m), and fail soft ("timeline source unavailable, single-post read still works").
- No credential storage in the repo; the CDP reader attaches to an already-logged-in Chrome profile on the Mac mini, it does not log in.
- Respect rate: cache profiles 6h, posts 15m.

## Acceptance criteria

1. `finctl voice read <url>` returns the tweet text + author + stats from fxtwitter.
2. `finctl voices` lists the whitelist with follower/bio from fxtwitter.
3. `voices tickers` degrades to a clear "timeline source unavailable" when the mirror/CDP path is down, without failing the whole command.
