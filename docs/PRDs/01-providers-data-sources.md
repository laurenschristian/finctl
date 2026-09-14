# finctl PRD 01: Provider catalog and data sources

Status: draft, 2026-09-14. Endpoints verified 2026-09-13 (data-sources.md) and re-confirmed 2026-09-14 via Exa where noted. Legend: OK = keyless, KEY = free key, UA = needs a real User-Agent, CF = Cloudflare wall, AUTH = token/crumb.

Each provider is one package `internal/provider/<name>` exposing typed functions that return `internal/model` types. Every call routes through `httpx.Get(ctx, "<name>", url, ttl)`. TTLs below are the cache lifetimes.

## Market and quotes

| Provider | Endpoint (verified) | Auth | TTL | Notes |
| --- | --- | --- | --- | --- |
| cboe | `https://cdn.cboe.com/api/global/delayed_quotes/quotes/<SYM>.json` (index `_VIX`, equity `NVDA`) | OK | 60s | delayed quote + VIX. Options chain at `.../options/<SYM>.json` (SPY ~5.7 MB) has IV, OI, greeks. Free public JSON that backs cboe.com; the paid LiveVol API is separate and not used. |
| yahoo | `https://query1.finance.yahoo.com/v8/finance/chart/<SYM>?range=6mo&interval=1d` | OK (unofficial) | 10m | OHLCV any range/interval, for `chart` sparkline and daily bars. Options need a crumb, so use cboe for chains. |
| finnhub | `https://finnhub.io/api/v1/*` (quote, profile2, metric, earnings calendar, news, recommendation) | KEY (have) | 60s quote, 24h profile | 60 req/min free. Overlay/secondary to XBRL, not the fundamentals source of truth. |
| stocktwits | `https://api.stocktwits.com/api/2/streams/symbol/<SYM>.json` | OK | 15m | symbol stream + bull/bear tags for sentiment. |
| coingecko | `https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd` | OK | 60s | crypto spot for `crypto`, `networth`. |
| frankfurter | `https://api.frankfurter.app/latest?from=USD&to=EUR` | OK (301 to https) | 6h | ECB FX for `fx`, `networth`. |
| mempool | `https://mempool.space/api/v1/fees/recommended` | OK | 5m | BTC fees, optional. |

## Filings, ownership, positioning

| Provider | Endpoint (verified) | Auth | TTL | Notes |
| --- | --- | --- | --- | --- |
| edgar (xbrl) | `https://data.sec.gov/api/xbrl/companyfacts/CIK##########.json`; `.../companyconcept/CIK##########/us-gaap/<Tag>.json`; frames `.../frames/us-gaap/<Tag>/USD/CY2025Q4.json` | UA (mandatory) | 24h | CIK is zero-padded to 10 digits. Hard cap 10 req/s TOTAL across all machines (SEC rate-control notice, 2021, still current). UA must identify app + contact email, e.g. `finctl/0.1 (lc@example.com)`. Source of truth for revenue, GM, capex, shares out, GAAP vs non-GAAP. |
| edgar (form4) | `https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=4&output=atom` | UA | 15m | latest insider transactions, Atom. Per-issuer via company filings feed. |
| edgar (fts) | `https://efts.sec.gov/LATEST/search-index?q=...&forms=10-K` | UA (403 naked) | 6h | full-text search of filings ("indium phosphide"). |
| ticker->CIK | `https://www.sec.gov/files/company_tickers.json` | UA | 7d | map symbol to CIK once, cache long. |
| finra | `POST https://api.finra.org/data/group/otcMarket/name/consolidatedShortInterest` body `{"compareFilters":[{"compareType":"EQUAL","fieldName":"settlementDate","fieldValue":"YYYY-MM-DD"}],"limit":N}` | OK (keyless POST) | 12h | consolidated + equity short interest, JSON or CSV. Fields: currentShortShareNumber, previousShortShareNumber, changePercent, daysToCoverNumber, settlementDate. Bi-monthly cadence. Confirmed 2026-09-14. |
| cftc (cot) | `https://publicreporting.cftc.gov/resource/<id>.json?$where=...&$limit=...` | OK (Socrata) | 24h | resource ids confirmed 2026-09-14: legacy futures-only `6dca-aqww`, legacy combined `jun7-fc8e`, disaggregated `72hh-3qpy`/`kh3c-gbw2`, TFF `gpe5-46if`/`yw9f-hn96`, supplemental `4zgm-a668`. SoQL `$select/$where/$order/$limit` (max 50000). Keyless works; a free Socrata app token raises limits. |
| openinsider | `https://openinsider.com/latest-cluster-buys` (HTML) | OK | 6h | cluster buys screen; parse the table. |
| disclosedcapitol | `https://disclosedcapitol.com/api/tickers/<SYM>/trades`, `/api/politicians` | OK | 12h | congress trades per ticker/politician, JSON, with 30d alpha. |

## Macro and rates

| Provider | Endpoint (verified) | Auth | TTL | Notes |
| --- | --- | --- | --- | --- |
| fred | `https://api.stlouisfed.org/fred/series/observations?series_id=DGS10&api_key=<K>&file_type=json` | KEY (free) | 6h | 800k series (rates, CPI, PPI, GDP, NFP, M2, curve). Free key, register at fred.stlouisfed.org/docs/api/api_key.html. |
| bls | `https://api.bls.gov/publicAPI/v2/timeseries/data/` (POST series_id list) | KEY (free, 500/day) | 12h | CPI/PPI/jobs. v1 keyless = 25/day; v2 keyed = 500/day. |
| eia | `https://api.eia.gov/v2/petroleum/pri/spt/data/?api_key=<K>&...` | KEY (free) | 12h | WTI/Brent/Henry Hub, inventories. |
| fed h15 | `https://www.federalreserve.gov/datadownload/Output.aspx?rel=H15&...&filetype=csv` | OK | 12h | daily Treasury constant maturities CSV, keyless fallback for `rates`/`curve` when FRED key absent. |
| treasury curve | `https://home.treasury.gov/resource-center/data-chart-center/interest-rates/pages/xml?data=daily_treasury_yield_curve&field_tdr_date_value_month=YYYYMM` | UA + `-L` | 12h | par yield curve XML. Retry with UA; naked curl returned 000. |
| fiscaldata | `https://api.fiscaldata.treasury.gov/services/api/fiscal_service/v2/accounting/od/avg_interest_rates` | OK | 24h | avg interest on debt, auctions, debt to the penny. |
| kalshi | `https://api.elections.kalshi.com/trade-api/v2/markets?series_ticker=KXFED` | OK (read) | 15m | Fed decision + CPI markets. Base also `external-api.kalshi.com/trade-api/v2`. Confirmed 2026-09-14. |
| polymarket | `https://gamma-api.polymarket.com/events?tag_slug=fed` | OK | 15m | event odds; filter by tag. |
| michigan | `https://sca.isr.umich.edu/files/tbmics.csv` | OK | 24h | consumer sentiment ICS monthly. |
| gdpnow | Atlanta Fed page + xlsx (301, follow) | OK | 12h | nowcast; parse the xlsx link. |

CME FedWatch is blocked to curl (000); derive Fed-cut odds from Kalshi + Polymarket, and optionally ZQ futures via Yahoo `ZQ=F`. AAII sentiment is a 403 bot wall; skip. Baker Hughes rig count is CF; use EIA weekly instead.

## Supply-chain edge (Serenity inputs)

| Provider | Endpoint (verified) | Auth | TTL | Notes |
| --- | --- | --- | --- | --- |
| capex (via edgar) | companyconcept tag `PaymentsToAcquirePropertyPlantAndEquipment` for MSFT/GOOGL/AMZN/META/ORCL | UA | 24h | hyperscaler quarterly capex + YoY, filing-accurate, free. |
| twse | `https://openapi.twse.com.tw/v1/opendata/t187ap05_L` | OK | 24h | every TW-listed firm's monthly revenue incl. TSMC (2330), MediaTek (2454), Hon Hai (2317), Wiwynn (6669), Quanta (2382), ASE (3711). One JSON (~600 KB). Replaces the CF-gated investor site. TSMC Aug-2026 = 514.8B TWD, +53.3% YoY (verified). |
| siliconanalysts | `https://siliconanalysts.com/api/v1/market-data`; `/api/v1/market-pulse` | OK | 12h | wafer price by node, HBM/DRAM pricing, packaging; 3 latest points per series free, with citations. Market-pulse = curated headlines with category + impact. |
| vastai | `https://console.vast.ai/api/v0/bundles/?q={"gpu_name":{"eq":"H100 SXM"}}` | OK (keyless) | 1h | live GPU rental asks (H100 SXM, B200) as a compute spot-price proxy; trend in SQLite. |

TBD / needs work (out of v1 scope, documented for later): Korea 20-day semiconductor exports (JS portal or Korea Customs PDF, or data.go.kr free key), Japan chip-equipment exports (e-Stat free key), DRAM/NAND spot beyond SiliconAnalysts (TrendForce scrape), memory-prices (KITA index).

## Personal

| Provider | Access | TTL | Notes |
| --- | --- | --- | --- |
| ibkr | Client Portal gateway on `https://localhost:5001/v1/api/*` (ibkrctl runs the daemon) | none (live) | positions, P&L, orders. finctl reuses the running gateway; it does NOT re-implement login. See ibkrctl. |
| monarch | Monarch GraphQL with a token (reuse the existing MCP token) | 1h | balances, cashflow, net-worth rollup. |
| vault | local Obsidian files: Dashboard target table, watchlist note, tracker CSV | none | read-only markdown/CSV parse. |

## Round-2 references (study, not adopt)

Alorse/trading-cli (Go CLI+MCP, closest in shape; read its provider layer), daniel3303/Equibles (self-hosted mini-Bloomberg, AGPL, 64 MCP tools; read its XBRL tag normalization), OpenBB (Python incumbent; copy provider naming only). fxtwitter (`https://api.fxtwitter.com/<handle>[/status/<id>]`) for single-tweet/profile JSON, used by PRD 06.

## Retry-from-different-network (were 000 on 2026-09-13, likely local)

FINRA Reg SHO daily short-sale volume `https://cdn.finra.org/equity/regsho/daily/CNMSshvolYYYYMMDD.txt`, DBnomics `api.db.nomics.world/v22/...`, Treasury XML. Not AdGuard-blocked (checked with adgctl). Re-verify during build; fall back to the primaries above.
