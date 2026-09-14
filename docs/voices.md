# Top voices on X: how finctl gets their intel

## Reality (verified 2026-09-13)
- Official X API: read access starts at ~$200/mo (Basic). Free tier cannot read other accounts. Not doing that.
- Nitter public instances are mostly dead; nitter.net timed out. xcancel.com is alive and serves RSS but returns "RSS reader not yet whitelisted!" for unknown user agents. They whitelist readers on request (form on the site). Test again with FreshRSS on the NAS (its UA may already be whitelisted) or ask for a whitelist for a `finctl/1.0` UA.
- fxtwitter (`api.fxtwitter.com`) works keyless for single tweets, profiles, and threads by id. No timelines.
- The `serenity-aleabitoreddit` skill already regenerates from the live feed every ~30 min via `skills update`. That pipeline exists somewhere; find out what it uses (likely a logged-in session) and reuse it instead of building a second scraper.

## Strategy (in order of preference)
1. **RSS via xcancel** for the curated handle list, whitelisted UA. Poll every 15 min into SQLite. Cheap, no login.
2. **Chrome CDP reader** (headless-twitter / finance-skills twitter-reader pattern): attach to the user's logged-in Chrome, intercept the GraphQL responses for `UserTweets` and `ListLatestTweetsTimeline`. Read-only, no keys, survives X changes better than DOM scraping. Use for X Lists (one list = all voices) and full threads/articles. Runs on the Mac mini, where Chrome stays open.
3. **fxtwitter** to hydrate any single tweet id (from RSS or from a pasted link) into clean JSON with media and quoted tweet.
4. Fallback: user pastes a link, `finctl voice read <url>`.

## What finctl does with it
- `finctl voices` : latest posts from the list, deduped, with $TICKER extraction, grouped by handle, `--since 24h`.
- `finctl voices tickers` : ticker mention counts across all voices in the window, new tickers vs vault, cross-ref with watchlist.
- `finctl voices thesis <handle>` : pull the last N posts for one handle, feed to the agent for a thesis refresh (this is what the serenity skill does by hand).
- Every post stored with id, handle, ts, text, tickers, url in `voices` table. The serenity skill reads from finctl instead of a tweet archive on disk.

## Seed list (edit freely; keep under ~30 so the list stays readable)
Supply-chain / semis: @aleabitoreddit (Serenity, 1.03M followers, 7.6k tweets), @dylan522p (SemiAnalysis), @sravanthi_r? (verify), @FoundryChat, @Beth_Kindig, @ian_cutress, @dnystedt (Dan Nystedt, TSMC/Taiwan), @kakashiii111, @rwang07 (Ray Wang, Asia semis), @jukanlosreve, @Jukan05.
Macro / rates: @LizAnnSonders, @NickTimiraos (Fed), @lisaabramowicz1, @M_McDonough, @KobeissiLetter, @zerohedge (noise, flag only), @charliebilello.
Flows / positioning: @unusual_whales, @spotgamma, @GoldmanSachs (no), @Barchart, @MacroCharts.
Energy (PetroBench): @JavierBlas, @HFI_Research, @EIAgov.
Prediction / odds: @Polymarket, @Kalshi.
Rule: every handle gets a `weight` (1-3) and a `topic`. The daily brief shows weight-3 voices only.

## Not doing
Reddit (needs OAuth app; low signal), StockTwits already covered as a data provider, Discord/Telegram groups (no).
