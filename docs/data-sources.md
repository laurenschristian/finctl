# Data sources, verified 2026-09-13 with plain curl (no key unless noted)

Legend: OK = works keyless; KEY = free key; UA = needs a real User-Agent; CF = Cloudflare challenge (needs browser UA or alt source); AUTH = token/crumb.

## Market
| Source | What | Status | Endpoint / note |
| --- | --- | --- | --- |
| Cboe delayed quotes | index + equity quotes, VIX | OK | `cdn.cboe.com/api/global/delayed_quotes/quotes/_VIX.json` |
| Cboe delayed options | full option chains (SPY = 5.7 MB) with IV, OI, greeks | OK | `cdn.cboe.com/api/global/delayed_quotes/options/<SYM>.json` |
| Yahoo chart | OHLCV any range/interval | OK (unofficial) | `query1.finance.yahoo.com/v8/finance/chart/NVDA?range=5d&interval=1d` |
| Yahoo options | chains | AUTH (crumb) | use Cboe instead |
| Finnhub | quotes, profile, metrics, earnings calendar, news, recs | KEY (have) | 60/min free |
| StockTwits | symbol stream + bull/bear tags | OK | `api.stocktwits.com/api/2/streams/symbol/NVDA.json` |
| Stooq | daily CSV | returns HTML to curl; needs `&h` params or UA | fallback only |
| CoinGecko | crypto prices | OK | `api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd` |
| Frankfurter | FX (ECB) | OK (follow 301 to https) | `api.frankfurter.app/latest?from=USD&to=EUR` |
| mempool.space | BTC fees/mempool | OK | `mempool.space/api/v1/fees/recommended` |

## Filings / ownership / positioning
| Source | What | Status | Endpoint / note |
| --- | --- | --- | --- |
| SEC XBRL companyconcept | every reported fact per company (capex, revenue, shares out, GM) | OK, UA required | `data.sec.gov/api/xbrl/companyconcept/CIK0000789019/us-gaap/PaymentsToAcquirePropertyPlantAndEquipment.json` |
| SEC XBRL frames | one fact across ALL companies for a period | OK | `data.sec.gov/api/xbrl/frames/us-gaap/Revenues/USD/CY2025Q4.json` |
| SEC companyfacts | all facts for one company in one call | OK | `data.sec.gov/api/xbrl/companyfacts/CIK##########.json` |
| SEC Form 4 feed | latest insider transactions (Atom) | OK | `sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=4&output=atom` |
| SEC full-text search | search 10-K/8-K text ("indium phosphide") | 403 naked; OK with real UA | `efts.sec.gov/LatestSearch/search-index?q=...&forms=10-K` |
| SEC fails-to-deliver | bi-monthly FTD files | OK (zip index page) | `sec.gov/data/foiadocsfailsdatahtm` |
| OpenInsider | cluster buys, screened insider trades (HTML) | OK | `openinsider.com/latest-cluster-buys` |
| FINRA short interest | consolidated short interest, bi-monthly | OK (CSV) | `api.finra.org/data/group/otcMarket/name/consolidatedShortInterest?limit=1` |
| CFTC COT | futures positioning (Socrata) | OK | `publicreporting.cftc.gov/resource/6dca-aqww.json` (legacy), `jun7-fc8e` (TFF) |
| House PTR disclosures | congress trades | OK (HTML/PDF index) | `disclosures-clerk.house.gov/FinancialDisclosure` |
| Senate eFD | congress trades | 302 to search, needs session cookie | scrape later |

## Macro
| Source | What | Status | Endpoint / note |
| --- | --- | --- | --- |
| FRED | 800k series: rates, CPI, PPI, GDP, NFP, M2, yield curve | KEY (free) | `api.stlouisfed.org/fred/series/observations?series_id=DGS10` |
| Fed H.15 | daily Treasury constant maturities CSV | OK | `federalreserve.gov/datadownload/Output.aspx?rel=H15&...&filetype=csv` |
| Treasury daily curve XML | par yield curve | 000 from curl (TLS/redirect), retry with -L and UA | `home.treasury.gov/resource-center/data-chart-center/interest-rates/pages/xml?data=daily_treasury_yield_curve&field_tdr_date_value_month=YYYYMM` |
| FiscalData | auctions, debt, avg interest rates | request failed in probe, retry (API is public) | `api.fiscaldata.treasury.gov/services/api/fiscal_service/v2/accounting/od/avg_interest_rates` |
| BLS v1 | CPI/PPI/jobs series, 25 req/day keyless | OK | `api.bls.gov/publicAPI/v1/timeseries/data/CUUR0000SA0` (v2 with key = 500/day) |
| Michigan sentiment | ICS monthly CSV | OK | `sca.isr.umich.edu/files/tbmics.csv` |
| AAII sentiment | weekly bull/bear | 403 (bot wall) | skip or scrape weekly page |
| Atlanta Fed GDPNow | nowcast | 301 (follow) | page + xlsx download |
| Kalshi | Fed decision markets, CPI markets | OK | `api.elections.kalshi.com/trade-api/v2/markets?series_ticker=KXFED` |
| Polymarket gamma | event odds (filter tag) | OK | `gamma-api.polymarket.com/events?tag_slug=fed` |
| CME FedWatch | rate probabilities | 000 (blocked to curl) | derive from ZQ futures via Yahoo `ZQ=F` or use Kalshi |
| EIA | WTI/Brent/HH, inventories | KEY (free) | `api.eia.gov/v2/petroleum/pri/spt/data/?...&api_key=` |
| Baker Hughes rig count | weekly rigs | CF | alt: EIA weekly or Baker Hughes xlsx link |
| Nasdaq Data Link | LBMA gold etc | 403 | needs key; skip, use Yahoo `GC=F` |

## Supply-chain edge (Serenity inputs)
| Source | What | Status | Endpoint / note |
| --- | --- | --- | --- |
| Hyperscaler capex | MSFT/GOOGL/AMZN/META/ORCL quarterly capex | OK via XBRL companyconcept | tag `PaymentsToAcquirePropertyPlantAndEquipment` |
| TSMC monthly revenue | monthly revenue TWD | CF on investor.tsmc.com | alt: TWSE MOPS monthly revenue endpoint `mops.twse.com.tw/nas/t21/sii/t21sc03_<yy>_<mm>_0.html` or TWSE openapi `openapi.twse.com.tw/v1/opendata/t187ap05_L` (verify) |
| Korea 20-day exports | semiconductor exports, first 20 days | 302 portal, JS | alt: Korea Customs press release PDFs, or tradingeconomics (paid). TBD |
| Japan chip-equipment exports | MoF trade stats | TBD | e-Stat API (free key) |
| DRAM/NAND spot | TrendForce free tables | TBD | scrape |
| GPU rental index | H100/B200 $/hr | TBD | public price pages (sfcompute, vast.ai API is free) |
| Google Trends | search interest | 429 to curl | skip or pytrends-style cookie dance; low value |

## Personal
| Source | What | Status |
| --- | --- | --- |
| IBKR Client Portal gateway | positions, P&L, orders | needs ibkrctl daemon on :5001 (see plan) |
| Monarch Money | balances, cashflow | GraphQL with token (existing MCP works; reuse token) |
| Obsidian vault | targets, watchlist, theses, tracker CSV | local files, parse markdown tables |

## Round 2 (Exa research + curl, 2026-09-13 late)

### New, verified working keyless
| Source | What | Endpoint / note |
| --- | --- | --- |
| TWSE OpenAPI monthly revenue | every TW-listed company's monthly revenue incl. TSMC (2330), MoM/YoY, one JSON (600 KB) | `openapi.twse.com.tw/v1/opendata/t187ap05_L` ; TSMC Aug-2026 = 514.8B TWD, +53.3% YoY. Replaces the Cloudflare-gated investor site. Also covers ASE, MediaTek, Hon Hai, Wiwynn, Quanta (AI server ODMs). |
| DisclosedCapitol | congress trades per ticker / politician, JSON, with 30d alpha | `disclosedcapitol.com/api/tickers/NVDA/trades`, `/api/politicians` |
| SiliconAnalysts market-data | wafer price by node, HBM/DRAM pricing, packaging, 3 latest points free per series, with citations | `siliconanalysts.com/api/v1/market-data` |
| SiliconAnalysts market-pulse | curated semiconductor headlines with category + impact | `siliconanalysts.com/api/v1/market-pulse` |
| vast.ai bundles | live GPU rental asks (H100 SXM, B200) = compute spot price proxy | `console.vast.ai/api/v0/bundles/?q={"gpu_name":{"eq":"H100 SXM"}}` |
| FiscalData | Treasury avg interest rates, auctions, debt to the penny | `api.fiscaldata.treasury.gov/services/api/fiscal_service/v2/accounting/od/avg_interest_rates` (works; first probe was a local hiccup) |
| fxtwitter | any single tweet + profile stats (followers, tweet count, bio) as JSON | `api.fxtwitter.com/<handle>` and `api.fxtwitter.com/<handle>/status/<id>` |
| market.vlsi.kr | Korean semiconductor official-stat hub (HTML, 21 data routes, sources trade data + filings) | scrape candidates; note as reference |

### Needs a free key
| Source | What | Note |
| --- | --- | --- |
| Korea data.go.kr | Korea Customs export/import stats API (관세청) incl. HS-code level = semiconductor exports | register on data.go.kr, XML API `apis.data.go.kr/1220000/...`; the 20-day flash number itself is a press release (PDF) from Korea Customs Service; parse or use KITA |
| DBnomics | 90 providers (ECB, IMF, BIS, OECD, Eurostat) keyless | `api.db.nomics.world/v22/series/...` (TLS failed from the Mac tonight; retry from NAS) |
| Business Quant | 13F institutional ownership, fundamentals, filings | free key |

### Failed tonight, retry from a different network
FINRA daily short-sale volume files `cdn.finra.org/equity/regsho/daily/CNMSshvolYYYYMMDD.txt` (000), DBnomics (000), Treasury XML (000), nitter.net (000). Not AdGuard (checked with `adgctl check`).

### Confirmed blocked / paid, drop
House Stock Watcher S3 (403, dead), Tessara (403), Equibles hosted MCP (needs key; self-host is AGPL Docker, 64 tools, worth a look as a reference), Quartr/EODHD/SEC-API (paid), Reddit JSON (403 without OAuth app; free app key works), AAII.

## Prior art to study, not adopt
- Alorse/trading-cli: Go, CLI + MCP, 25 tools, TA/screener/sentiment on public APIs. Closest in shape to finctl; read its provider layer.
- daniel3303/Equibles: self-hosted "mini Bloomberg", 64 MCP tools, XBRL parsing, EDGAR full-text. AGPL. Read how it normalizes XBRL tags.
- himself65/finance-skills: skills pack incl. twitter-reader via Chrome Browser Bridge. Confirms the "use the logged-in Chrome" path for X.
- OpenBB: the incumbent. Python, heavy. Copy its provider naming, nothing else.
