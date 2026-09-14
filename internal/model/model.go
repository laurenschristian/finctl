// Package model holds finctl's normalized types. Providers return these; the
// cli and mcp layers render them.
package model

// Quote is a delayed market snapshot for one symbol.
type Quote struct {
	Symbol     string  `json:"symbol"`
	Last       float64 `json:"last"`
	Change     float64 `json:"change,omitempty"`
	ChangePct  float64 `json:"changePct,omitempty"`
	DayLow     float64 `json:"dayLow,omitempty"`
	DayHigh    float64 `json:"dayHigh,omitempty"`
	Volume     float64 `json:"volume,omitempty"`
	Open       float64 `json:"open,omitempty"`
	PrevClose  float64 `json:"prevClose,omitempty"`
	Week52Low  float64 `json:"week52Low,omitempty"`
	Week52High float64 `json:"week52High,omitempty"`
	Currency   string  `json:"currency,omitempty"`
	AsOf       string  `json:"asOf,omitempty"`
}

// Bar is one OHLCV candle.
type Bar struct {
	Time   int64   `json:"t"`
	Open   float64 `json:"o"`
	High   float64 `json:"h"`
	Low    float64 `json:"l"`
	Close  float64 `json:"c"`
	Volume float64 `json:"v,omitempty"`
}

// Chart is a series of bars for a symbol.
type Chart struct {
	Symbol string `json:"symbol"`
	Range  string `json:"range"`
	Bars   []Bar  `json:"bars"`
}

// Period is one fiscal period of fundamentals.
type Period struct {
	Fiscal      string  `json:"fiscal"`
	End         string  `json:"end,omitempty"`
	Revenue     float64 `json:"revenue,omitempty"`
	GrossMargin float64 `json:"grossMargin,omitempty"`
	Opex        float64 `json:"opex,omitempty"`
	FCF         float64 `json:"fcf,omitempty"`
	SharesOut   float64 `json:"sharesOut,omitempty"`
	Capex       float64 `json:"capex,omitempty"`
}

// Fundamentals is a company's reported trend from XBRL.
type Fundamentals struct {
	Symbol  string   `json:"symbol"`
	CIK     string   `json:"cik,omitempty"`
	Periods []Period `json:"periods"`
}

// CapexPoint is one company-quarter of capital expenditure.
type CapexPoint struct {
	Company string  `json:"company"`
	Fiscal  string  `json:"fiscal"`
	Capex   float64 `json:"capex"`
	YoY     float64 `json:"yoy,omitempty"`
}

// CryptoPrice is a spot price for a coin vs a fiat currency.
type CryptoPrice struct {
	Coin     string  `json:"coin"`
	VS       string  `json:"vs"`
	Price    float64 `json:"price"`
	Change24 float64 `json:"change24h,omitempty"`
}

// FXRate is a spot exchange rate.
type FXRate struct {
	From string  `json:"from"`
	To   string  `json:"to"`
	Rate float64 `json:"rate"`
	Date string  `json:"date,omitempty"`
}

// RatePoint is one tenor on the Treasury/yield curve, or a policy rate.
type RatePoint struct {
	Tenor string  `json:"tenor"`
	Yield float64 `json:"yield"`
	AsOf  string  `json:"asOf,omitempty"`
}

// Obs is one observation of a time series.
type Obs struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// Series is a named data series (e.g. a FRED series).
type Series struct {
	ID     string `json:"id"`
	Title  string `json:"title,omitempty"`
	Units  string `json:"units,omitempty"`
	Points []Obs  `json:"points"`
}

// FedBucket is the market-implied probability the target lands in a rate band.
type FedBucket struct {
	Band string  `json:"band"`
	Prob float64 `json:"prob"`
}

// FedOdds is the implied outcome distribution for one FOMC meeting.
type FedOdds struct {
	Meeting string      `json:"meeting"`
	Source  string      `json:"source"`
	AsOf    string      `json:"asOf,omitempty"`
	Buckets []FedBucket `json:"buckets"`
}

// CotReport is CFTC Commitments of Traders positioning for one market.
type CotReport struct {
	Market   string  `json:"market"`
	Date     string  `json:"date"`
	Long     float64 `json:"long"`
	Short    float64 `json:"short"`
	Net      float64 `json:"net"`
	NetPrior float64 `json:"netPrior,omitempty"`
	AsOf     string  `json:"asOf,omitempty"`
}

// DebtSummary is the Treasury debt-to-the-penny snapshot.
type DebtSummary struct {
	Date       string  `json:"date"`
	TotalDebt  float64 `json:"totalDebt"`
	HeldPublic float64 `json:"heldPublic"`
	Intragov   float64 `json:"intragov"`
}

// MonthlyRevenue is one company-month from the TWSE monthly revenue feed (TWD).
type MonthlyRevenue struct {
	CompanyID string  `json:"companyId"`
	Name      string  `json:"name"`
	Month     string  `json:"month"`
	Revenue   float64 `json:"revenueTWD"`
	MoM       float64 `json:"mom,omitempty"`
	YoY       float64 `json:"yoy,omitempty"`
}

// GPURent is the lowest ask and median $/GPU-hour for a GPU type on vast.ai.
type GPURent struct {
	GPU    string  `json:"gpu"`
	LowAsk float64 `json:"lowAsk"`
	Median float64 `json:"median"`
	Offers int     `json:"offers"`
	AsOf   string  `json:"asOf"`
}

// ShortInterest is one FINRA consolidated short-interest settlement.
type ShortInterest struct {
	Symbol         string  `json:"symbol"`
	SettlementDate string  `json:"settlementDate"`
	Current        float64 `json:"current"`
	Previous       float64 `json:"previous,omitempty"`
	ChangePct      float64 `json:"changePct,omitempty"`
	DaysToCover    float64 `json:"daysToCover,omitempty"`
	AvgDailyVol    float64 `json:"avgDailyVol,omitempty"`
}

// Strike is one strike's open interest, for option walls.
type Strike struct {
	Strike float64 `json:"strike"`
	OI     float64 `json:"oi"`
}

// OptionsSummary distills a delayed Cboe option chain.
type OptionsSummary struct {
	Symbol       string   `json:"symbol"`
	Underlying   float64  `json:"underlying,omitempty"`
	FrontIV      float64  `json:"frontIV,omitempty"`
	PutCallRatio float64  `json:"putCallRatio,omitempty"`
	MaxPain      float64  `json:"maxPain,omitempty"`
	OIWalls      []Strike `json:"oiWalls,omitempty"`
	Expiries     int      `json:"expiries,omitempty"`
}

// Filing is one SEC submission.
type Filing struct {
	Form       string `json:"form"`
	FilingDate string `json:"filingDate"`
	ReportDate string `json:"reportDate,omitempty"`
	Accession  string `json:"accession"`
	Doc        string `json:"doc,omitempty"`
	Desc       string `json:"desc,omitempty"`
	URL        string `json:"url,omitempty"`
}

// DilutionReport is the shares-outstanding trend with a simple growth flag.
type DilutionReport struct {
	Symbol      string  `json:"symbol"`
	Points      []Obs   `json:"points"`
	ChangePctYr float64 `json:"changePctYr,omitempty"`
	Flag        string  `json:"flag,omitempty"`
}

// Position is one holding, optionally with vault target weight and drift.
type Position struct {
	Symbol       string  `json:"symbol"`
	Qty          float64 `json:"qty"`
	Price        float64 `json:"price,omitempty"`
	Value        float64 `json:"value"`
	CostBasis    float64 `json:"costBasis,omitempty"`
	UnrealPnL    float64 `json:"unrealPnL,omitempty"`
	Weight       float64 `json:"weight,omitempty"`
	TargetWeight float64 `json:"targetWeight,omitempty"`
	Drift        float64 `json:"drift,omitempty"`
	Account      string  `json:"account,omitempty"`
}

// WatchRow is a watchlist row with live price and distance to buy-zone.
type WatchRow struct {
	Symbol           string  `json:"symbol"`
	Last             float64 `json:"last,omitempty"`
	BuyLow           float64 `json:"buyLow,omitempty"`
	BuyHigh          float64 `json:"buyHigh,omitempty"`
	DistanceToBuyPct float64 `json:"distanceToBuyPct,omitempty"`
	Thesis           string  `json:"thesis,omitempty"`
	InZone           bool    `json:"inZone,omitempty"`
}
