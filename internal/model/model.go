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
