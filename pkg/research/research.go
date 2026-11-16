// Package research assembles a per-ticker dossier from the free providers and scores
// it against the Serenity lens. It builds for js/wasm so folio can reuse it.
package research

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/pkg/httpx"
	"github.com/laurenschristian/finctl/pkg/model"
	"github.com/laurenschristian/finctl/pkg/provider"
)

// Section is one independently fetched and cached slice of the dossier.
type Section string

const (
	Filings    Section = "filings"
	Insider    Section = "insider"
	Short      Section = "short"
	Options    Section = "options"
	Dilution   Section = "dilution"
	Financials Section = "financials"
)

// Sections lists every section in display order.
var Sections = []Section{Financials, Filings, Insider, Short, Options, Dilution}

// LensSections are the sections Lens reads.
var LensSections = []Section{Dilution, Short, Options, Insider, Financials}

const (
	FilingsLimit = 12
	// InsiderFilings caps Form 4 documents per refresh: each one is an extra request.
	InsiderFilings = 8
)

// Requests is the most outbound requests a cold fetch of each section makes.
var Requests = map[Section]int{Filings: 2, Insider: 2 + InsiderFilings, Short: 1, Options: 1, Dilution: 2, Financials: 2}

var filingForms = map[string]bool{"8-K": true, "8-K/A": true, "10-Q": true, "10-Q/A": true, "10-K": true, "10-K/A": true}

// Dossier holds whatever sections were fetched. Price and Earnings come from the caller.
type Dossier struct {
	Symbol     string                `json:"symbol"`
	Price      float64               `json:"price,omitempty"`
	Earnings   string                `json:"earnings,omitempty"`
	Filings    []model.Filing        `json:"filings,omitempty"`
	Insider    []model.InsiderTrade  `json:"insider,omitempty"`
	Short      *model.ShortInterest  `json:"short,omitempty"`
	Options    *model.OptionsSummary `json:"options,omitempty"`
	Dilution   *model.DilutionReport `json:"dilution,omitempty"`
	Financials *model.Financials     `json:"financials,omitempty"`
	Errors     map[string]string     `json:"errors,omitempty"`
}

// Field returns a pointer to the section's field, for decoding a cached JSON body into it.
func (d *Dossier) Field(s Section) any {
	switch s {
	case Filings:
		return &d.Filings
	case Insider:
		return &d.Insider
	case Short:
		return &d.Short
	case Options:
		return &d.Options
	case Dilution:
		return &d.Dilution
	case Financials:
		return &d.Financials
	}
	return nil
}

// Fetch loads one section from the providers into the dossier.
func Fetch(ctx context.Context, h *httpx.Client, d *Dossier, s Section) error {
	sym := d.Symbol
	var err error
	switch s {
	case Filings:
		var all []model.Filing
		if all, err = provider.Filings(ctx, h, sym, "", 300); err == nil {
			d.Filings = d.Filings[:0]
			for _, f := range all {
				if filingForms[strings.ToUpper(f.Form)] && len(d.Filings) < FilingsLimit {
					d.Filings = append(d.Filings, f)
				}
			}
		}
	case Insider:
		d.Insider, err = provider.InsiderTrades(ctx, h, sym, InsiderFilings)
	case Short:
		d.Short, err = provider.ShortInterest(ctx, h, sym)
	case Options:
		d.Options, err = provider.OptionsChain(ctx, h, sym)
	case Dilution:
		d.Dilution, err = provider.Dilution(ctx, h, sym)
	case Financials:
		d.Financials, err = provider.Financials(ctx, h, sym)
	default:
		err = fmt.Errorf("unknown section %q", s)
	}
	return err
}

// Fill fetches each section, recording failures in d.Errors instead of stopping.
func Fill(ctx context.Context, h *httpx.Client, d *Dossier, secs ...Section) {
	for _, s := range secs {
		if err := Fetch(ctx, h, d, s); err != nil {
			if d.Errors == nil {
				d.Errors = map[string]string{}
			}
			d.Errors[string(s)] = err.Error()
			continue
		}
		delete(d.Errors, string(s))
	}
}

// Status answers a lens question: yes fits the lens, no works against it.
type Status string

const (
	Yes     Status = "yes"
	No      Status = "no"
	Mixed   Status = "mixed"
	Unknown Status = "unknown"
)

// Item is one lens question. Auto items are computed from data; the rest need a human read.
type Item struct {
	Key      string `json:"key"`
	Question string `json:"question"`
	Auto     bool   `json:"auto"`
	Status   Status `json:"status"`
	Evidence string `json:"evidence,omitempty"`
}

// Lens scores the dossier against the Serenity checklist. Items it cannot compute stay
// Unknown rather than guessed. None of it is a buy signal on its own.
func Lens(d *Dossier, now time.Time) []Item {
	return []Item{
		manual("bottleneck", "Sole or near-sole source chokepoint with pricing power?"),
		manual("upstream", "Upstream of the obvious winner and a small share of downstream cost?"),
		manual("chain", "Supply chain mapped from input layer to finished system?"),
		manual("demand", "Demand driven by AI or robotics capex, not a legacy cycle?"),
		manual("contracts", "Multi-year contracts with creditworthy customers?"),
		margins(d),
		financing(d),
		manual("stage", "Pre volume ramp and not yet crowded?"),
		catalyst(d, now),
		headroom(d),
		manual("validation", "Analyst coverage still behind the supply-chain evidence?"),
		risk(d),
		insiders(d),
	}
}

func manual(key, q string) Item { return Item{Key: key, Question: q, Status: Unknown} }

func margins(d *Dossier) Item {
	it := Item{Key: "margins", Question: "GAAP margins support the quality claim?", Auto: true, Status: Unknown}
	if d.Financials == nil || d.Financials.TTM == nil || d.Financials.TTM.Revenue == 0 {
		return it
	}
	t := d.Financials.TTM
	om := t.OpMargin()
	it.Evidence = fmt.Sprintf("TTM GAAP operating margin %.1f%%, net margin %.1f%%", om*100, t.NetMargin()*100)
	switch {
	case om >= 0.15:
		it.Status = Yes
	case om > 0:
		it.Status = Mixed
	default:
		it.Status = No
	}
	return it
}

func financing(d *Dossier) Item {
	it := Item{Key: "financing", Question: "Clean financing, no dilution overhang?", Auto: true, Status: Unknown}
	if d.Dilution == nil || len(d.Dilution.Points) < 5 {
		return it
	}
	c := d.Dilution.ChangePctYr
	it.Evidence = fmt.Sprintf("Shares %+.1f%% year over year (%s)", c, d.Dilution.Flag)
	switch {
	case c <= 1:
		it.Status = Yes
	case c <= 5:
		it.Status = Mixed
	default:
		it.Status = No
	}
	return it
}

func catalyst(d *Dossier, now time.Time) Item {
	it := Item{Key: "catalyst", Question: "Dated catalyst within 90 days?", Auto: true, Status: Unknown}
	day, err := time.Parse("2006-01-02", d.Earnings)
	if err != nil {
		return it
	}
	days := int(day.Sub(now.Truncate(24*time.Hour)).Hours() / 24)
	if days < 0 {
		return it
	}
	it.Evidence = fmt.Sprintf("Earnings %s, in %d days", d.Earnings, days)
	it.Status = No
	if days <= 90 {
		it.Status = Yes
	}
	return it
}

// marketCap uses the latest reported share count, so it lags buybacks and issuance by a quarter.
func marketCap(d *Dossier) float64 {
	price := d.Price
	if price == 0 && d.Options != nil {
		price = d.Options.Underlying
	}
	var shares float64
	if d.Dilution != nil && len(d.Dilution.Points) > 0 {
		shares = d.Dilution.Points[len(d.Dilution.Points)-1].Value
	} else if d.Financials != nil && len(d.Financials.Quarters) > 0 {
		shares = d.Financials.Quarters[len(d.Financials.Quarters)-1].Shares
	}
	return price * shares
}

func headroom(d *Dossier) Item {
	it := Item{Key: "headroom", Question: "Market cap small enough (under about $3B) for a re-rating?", Auto: true, Status: Unknown}
	mc := marketCap(d)
	if mc <= 0 {
		return it
	}
	it.Evidence = fmt.Sprintf("Market cap about $%.1fB", mc/1e9)
	switch {
	case mc < 3e9:
		it.Status = Yes
	case mc < 10e9:
		it.Status = Mixed
	default:
		it.Status = No
	}
	return it
}

func risk(d *Dossier) Item {
	it := Item{Key: "risk", Question: "Low binary risk (implied vol, short crowding, dilution)?", Auto: true, Status: Unknown}
	if d.Options == nil && d.Short == nil {
		return it
	}
	var parts []string
	score := 0
	if o := d.Options; o != nil && o.FrontIV > 0 {
		parts = append(parts, fmt.Sprintf("30d IV %.0f%%", o.FrontIV))
		score += level(o.FrontIV, 50, 80)
	}
	if s := d.Short; s != nil && s.DaysToCover > 0 {
		parts = append(parts, fmt.Sprintf("%.1f days to cover", s.DaysToCover))
		score += level(s.DaysToCover, 4, 7)
	}
	if dl := d.Dilution; dl != nil && dl.Flag == "diluting" {
		parts = append(parts, "diluting")
		score += level(dl.ChangePctYr, 1, 5)
	}
	if len(parts) == 0 {
		return it
	}
	it.Evidence = strings.Join(parts, ", ")
	switch {
	case score >= 2:
		it.Status = No
	case score == 1:
		it.Status = Mixed
	default:
		it.Status = Yes
	}
	return it
}

// level is 0 below warn, 1 from warn, 2 from bad.
func level(v, warn, bad float64) int {
	switch {
	case v >= bad:
		return 2
	case v >= warn:
		return 1
	}
	return 0
}

func insiders(d *Dossier) Item {
	it := Item{Key: "insiders", Question: "Insiders buying in the open market, not only selling?", Auto: true, Status: Unknown}
	if d.Insider == nil {
		return it
	}
	var buys, sells int
	for _, t := range d.Insider {
		switch t.Code {
		case "P":
			buys++
		case "S":
			sells++
		}
	}
	it.Evidence = fmt.Sprintf("%d open-market buys, %d sales in the last %d Form 4 filings", buys, sells, InsiderFilings)
	switch {
	case buys > 0 && buys >= sells:
		it.Status = Yes
	case buys > 0:
		it.Status = Mixed
	case sells > 0:
		it.Status = No
	default:
		it.Status = Mixed
	}
	return it
}
