package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

// edgarErr annotates a 403 with the SEC User-Agent requirement.
func edgarErr(err error) error {
	if err != nil && strings.Contains(err.Error(), "HTTP 403") {
		return fmt.Errorf("SEC rejected the request: set a contact User-Agent (config user_agent or FINCTL_USER_AGENT), e.g. \"finctl/0.1 (you@example.com)\"")
	}
	return err
}

const (
	edgarFactsTTL  = 24 * time.Hour
	edgarTickerTTL = 7 * 24 * time.Hour
)

var (
	edgarData = "https://data.sec.gov"
	edgarWWW  = "https://www.sec.gov"
)

// CapexTag is the primary XBRL concept for capital expenditure; some filers use
// alternates, tried in order.
var capexTags = []string{
	"PaymentsToAcquirePropertyPlantAndEquipment",
	"PaymentsToAcquireProductiveAssets",
}
var revenueTags = []string{
	"RevenueFromContractWithCustomerExcludingAssessedTax",
	"Revenues",
	"SalesRevenueNet",
}
var sharesTags = []string{"CommonStockSharesOutstanding"}

// CIKFor resolves a ticker to a zero-padded 10-digit CIK using SEC's ticker map.
func CIKFor(ctx context.Context, h *httpx.Client, ticker string) (string, error) {
	b, err := h.Get(ctx, "edgar", edgarWWW+"/files/company_tickers.json", edgarTickerTTL)
	if err != nil {
		return "", edgarErr(err)
	}
	var raw map[string]struct {
		CIK    int    `json:"cik_str"`
		Ticker string `json:"ticker"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return "", err
	}
	up := strings.ToUpper(ticker)
	for _, e := range raw {
		if strings.ToUpper(e.Ticker) == up {
			return fmt.Sprintf("CIK%010d", e.CIK), nil
		}
	}
	return "", fmt.Errorf("unknown ticker %q (not in SEC ticker map)", ticker)
}

type factPoint struct {
	Start string
	End   string
	Val   float64
	Frame string
	Form  string
}

// concept pulls the USD (or shares) facts for one XBRL tag, keeping only the
// clean calendar-period facts SEC assigns a frame to.
func concept(ctx context.Context, h *httpx.Client, cik, taxonomy, tag string) ([]factPoint, error) {
	u := fmt.Sprintf("%s/api/xbrl/companyconcept/%s/%s/%s.json", edgarData, cik, taxonomy, tag)
	b, err := h.Get(ctx, "edgar", u, edgarFactsTTL)
	if err != nil {
		return nil, edgarErr(err)
	}
	var raw struct {
		Units map[string][]struct {
			Start string  `json:"start"`
			End   string  `json:"end"`
			Val   float64 `json:"val"`
			Frame string  `json:"frame"`
			Form  string  `json:"form"`
		} `json:"units"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	var pts []factPoint
	for _, arr := range raw.Units { // USD, or shares
		for _, e := range arr {
			if e.Frame == "" { // frame-tagged facts are the deduped calendar values
				continue
			}
			pts = append(pts, factPoint{Start: e.Start, End: e.End, Val: e.Val, Frame: e.Frame, Form: e.Form})
		}
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].End < pts[j].End })
	return pts, nil
}

// conceptAny tries a list of tags and returns the first that yields facts.
func conceptAny(ctx context.Context, h *httpx.Client, cik, taxonomy string, tags []string) ([]factPoint, error) {
	var lastErr error
	for _, tag := range tags {
		pts, err := concept(ctx, h, cik, taxonomy, tag)
		if err == nil && len(pts) > 0 {
			return pts, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, nil
}

// isQuarter keeps duration facts spanning roughly one quarter (frames ending in
// Qn but not the instant "I" or the full-year frames).
func isQuarter(fr string) bool {
	return strings.Contains(fr, "Q") && !strings.HasSuffix(fr, "I")
}

// Capex returns quarterly capex points (most recent last) for a CIK.
func capexPoints(ctx context.Context, h *httpx.Client, cik string) ([]factPoint, error) {
	pts, err := conceptAny(ctx, h, cik, "us-gaap", capexTags)
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().AddDate(-3, 0, 0).Format("2006-01-02")
	var q []factPoint
	for _, p := range pts {
		if isQuarter(p.Frame) && p.End >= cutoff {
			q = append(q, p)
		}
	}
	return q, nil
}

// factRow is one XBRL fact from the companyfacts blob, keeping the fiscal
// tagging (fy/fp/form/filed) frames throw away.
type factRow struct {
	Start, End, FP, Form, Filed string
	FY                          int
	Val                         float64
}

// companyFacts fetches every fact for a CIK in one call and indexes it by
// us-gaap tag. Unlike frames, companyfacts includes off-calendar fiscal-year
// filers (NVDA, AAPL), so fund works for them too.
func companyFacts(ctx context.Context, h *httpx.Client, cik string) (map[string][]factRow, error) {
	u := fmt.Sprintf("%s/api/xbrl/companyfacts/%s.json", edgarData, cik)
	b, err := h.Get(ctx, "edgar", u, edgarFactsTTL)
	if err != nil {
		return nil, edgarErr(err)
	}
	var raw struct {
		Facts map[string]map[string]struct {
			Units map[string][]struct {
				Start string  `json:"start"`
				End   string  `json:"end"`
				Val   float64 `json:"val"`
				FY    int     `json:"fy"`
				FP    string  `json:"fp"`
				Form  string  `json:"form"`
				Filed string  `json:"filed"`
			} `json:"units"`
		} `json:"facts"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := map[string][]factRow{}
	for _, tags := range raw.Facts { // us-gaap, dei
		for tag, fact := range tags {
			for _, arr := range fact.Units { // USD, shares
				for _, e := range arr {
					out[tag] = append(out[tag], factRow{
						Start: e.Start, End: e.End, FP: e.FP, Form: e.Form,
						Filed: e.Filed, FY: e.FY, Val: e.Val,
					})
				}
			}
		}
	}
	return out, nil
}

// spanDays returns the day count of a duration fact, or -1 if unparseable.
func spanDays(start, end string) int {
	const d = "2006-01-02"
	s, err1 := time.Parse(d, start)
	e, err2 := time.Parse(d, end)
	if start == "" || err1 != nil || err2 != nil {
		return -1
	}
	return int(e.Sub(s).Hours() / 24)
}

// cyLabel maps a period-end date to a calendar-quarter label (CY2024Q2). The
// fact's own end date is authoritative; companyfacts fy/fp describe the filing,
// not the period, so a 10-K's prior-year comparatives carry the filing's fy/fp.
func cyLabel(end string) string {
	t, err := time.Parse("2006-01-02", end)
	if err != nil {
		return end
	}
	return fmt.Sprintf("CY%dQ%d", t.Year(), (int(t.Month())-1)/3+1)
}

// quarterly keeps one ~13-week duration fact per period end (latest filed wins),
// keyed on the fact's own end (not fy/fp). Among the candidate tags it returns
// the one whose data reaches furthest forward, so a stale legacy tag with only
// old quarters never shadows the tag a filer switched to.
func quarterly(facts map[string][]factRow, tags []string) []factRow {
	var best []factRow
	var bestEnd string
	for _, tag := range tags {
		rows, ok := facts[tag]
		if !ok {
			continue
		}
		m := map[string]factRow{} // key = period end
		for _, r := range rows {
			if n := spanDays(r.Start, r.End); n < 80 || n > 100 {
				continue // keep single quarters, drop 6mo/9mo/annual durations
			}
			if prev, ok := m[r.End]; !ok || r.Filed > prev.Filed {
				m[r.End] = r
			}
		}
		if len(m) == 0 {
			continue
		}
		out := make([]factRow, 0, len(m))
		for _, r := range m {
			out = append(out, r)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].End < out[j].End })
		if last := out[len(out)-1].End; last > bestEnd {
			best, bestEnd = out, last
		}
	}
	return best
}

// quarterlyFlow de-cumulates cash-flow facts (capex, FCF), which SEC reports
// year-to-date. YTD facts within a fiscal year share the same start date, so
// group by start and difference consecutive ends to recover discrete quarters.
func quarterlyFlow(facts map[string][]factRow, tags []string) []factRow {
	var best []factRow
	var bestEnd string
	for _, tag := range tags {
		rows, ok := facts[tag]
		if !ok {
			continue
		}
		byStart := map[string]map[string]factRow{} // start -> end -> latest-filed
		for _, r := range rows {
			if n := spanDays(r.Start, r.End); n < 80 || n > 300 {
				continue // 3..9 month YTD (or a discrete quarter) spans only
			}
			if byStart[r.Start] == nil {
				byStart[r.Start] = map[string]factRow{}
			}
			if prev, ok := byStart[r.Start][r.End]; !ok || r.Filed > prev.Filed {
				byStart[r.Start][r.End] = r
			}
		}
		var out []factRow
		for _, m := range byStart {
			seq := make([]factRow, 0, len(m))
			for _, r := range m {
				seq = append(seq, r)
			}
			sort.Slice(seq, func(i, j int) bool { return seq[i].End < seq[j].End })
			for i, r := range seq {
				d := r
				if i > 0 {
					d.Val = r.Val - seq[i-1].Val // discrete = YTD now minus YTD prior quarter
				}
				out = append(out, d)
			}
		}
		if len(out) == 0 {
			continue
		}
		sort.Slice(out, func(i, j int) bool { return out[i].End < out[j].End })
		if last := out[len(out)-1].End; last > bestEnd {
			best, bestEnd = out, last
		}
	}
	return best
}

// Fundamentals returns the revenue, capex, and shares-outstanding trend for a
// ticker from SEC XBRL companyfacts (quarterly, deduped by fiscal period).
func Fundamentals(ctx context.Context, h *httpx.Client, ticker string) (*model.Fundamentals, error) {
	cik, err := CIKFor(ctx, h, ticker)
	if err != nil {
		return nil, err
	}
	facts, err := companyFacts(ctx, h, cik)
	if err != nil {
		return nil, err
	}
	byPeriod := map[string]*model.Period{}
	order := []string{}
	get := func(end string) *model.Period {
		label := cyLabel(end)
		p, ok := byPeriod[label]
		if !ok {
			p = &model.Period{Fiscal: label, End: end}
			byPeriod[label] = p
			order = append(order, label)
		}
		return p
	}
	for _, r := range quarterly(facts, revenueTags) {
		get(r.End).Revenue = r.Val
	}
	for _, r := range quarterlyFlow(facts, capexTags) {
		get(r.End).Capex = r.Val
	}
	// Shares outstanding are instant facts; overlay the value dated nearest each
	// period end onto rows that already exist (never create shares-only rows).
	var shares []factRow
	for _, tag := range append(sharesTags, "EntityCommonStockSharesOutstanding") {
		if rows, ok := facts[tag]; ok {
			shares = append(shares, rows...)
		}
	}
	for _, p := range byPeriod {
		var best factRow
		for _, s := range shares {
			if s.End == "" {
				continue
			}
			if best.End == "" || absDayDiff(s.End, p.End) < absDayDiff(best.End, p.End) {
				best = s
			}
		}
		if best.End != "" && absDayDiff(best.End, p.End) <= 45 {
			p.SharesOut = best.Val
		}
	}
	sort.Slice(order, func(i, j int) bool { return byPeriod[order[i]].End < byPeriod[order[j]].End })
	out := &model.Fundamentals{Symbol: strings.ToUpper(ticker), CIK: cik}
	// Keep the most recent 12 periods (roughly 8 quarters plus prior year).
	if len(order) > 12 {
		order = order[len(order)-12:]
	}
	for _, label := range order {
		out.Periods = append(out.Periods, *byPeriod[label])
	}
	return out, nil
}

// absDayDiff is the absolute day gap between two YYYY-MM-DD dates (large on parse error).
func absDayDiff(a, b string) int {
	const d = "2006-01-02"
	ta, e1 := time.Parse(d, a)
	tb, e2 := time.Parse(d, b)
	if e1 != nil || e2 != nil {
		return 1 << 30
	}
	n := int(ta.Sub(tb).Hours() / 24)
	if n < 0 {
		n = -n
	}
	return n
}

// HyperscalerCapex returns recent quarterly capex per company with YoY.
func HyperscalerCapex(ctx context.Context, h *httpx.Client, tickers []string, quarters int) ([]model.CapexPoint, error) {
	if quarters <= 0 {
		quarters = 6
	}
	var out []model.CapexPoint
	for _, t := range tickers {
		cik, err := CIKFor(ctx, h, t)
		if err != nil {
			return nil, err
		}
		pts, err := capexPoints(ctx, h, cik)
		if err != nil {
			return nil, err
		}
		byFrame := map[string]float64{}
		for _, p := range pts {
			byFrame[p.Frame] = p.Val
		}
		start := len(pts) - quarters
		if start < 0 {
			start = 0
		}
		for _, p := range pts[start:] {
			cp := model.CapexPoint{Company: strings.ToUpper(t), Fiscal: p.Frame, Capex: p.Val}
			if prior, ok := byFrame[yoyFrame(p.Frame)]; ok && prior != 0 {
				cp.YoY = (p.Val - prior) / prior * 100
			}
			out = append(out, cp)
		}
	}
	return out, nil
}

// yoyFrame maps CY2025Q2 -> CY2024Q2 for a year-over-year lookup.
func yoyFrame(fr string) string {
	if len(fr) < 6 || !strings.HasPrefix(fr, "CY") {
		return ""
	}
	var year int
	if _, err := fmt.Sscanf(fr[2:6], "%d", &year); err != nil {
		return ""
	}
	return fmt.Sprintf("CY%d%s", year-1, fr[6:])
}
