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

// Fundamentals returns the revenue, capex, and shares-outstanding trend for a
// ticker from SEC XBRL (frame-clean quarterly facts).
func Fundamentals(ctx context.Context, h *httpx.Client, ticker string) (*model.Fundamentals, error) {
	cik, err := CIKFor(ctx, h, ticker)
	if err != nil {
		return nil, err
	}
	byFrame := map[string]*model.Period{}
	get := func(frame string) *model.Period {
		p, ok := byFrame[frame]
		if !ok {
			p = &model.Period{Fiscal: frame}
			byFrame[frame] = p
		}
		return p
	}
	if rev, err := conceptAny(ctx, h, cik, "us-gaap", revenueTags); err == nil {
		for _, p := range rev {
			if isQuarter(p.Frame) {
				row := get(p.Frame)
				row.Revenue = p.Val
				row.End = p.End
			}
		}
	}
	if cpx, err := capexPoints(ctx, h, cik); err == nil {
		for _, p := range cpx {
			pd := get(p.Frame)
			pd.Capex = p.Val
			if pd.End == "" {
				pd.End = p.End
			}
		}
	}
	if sh, err := conceptAny(ctx, h, cik, "us-gaap", sharesTags); err == nil {
		for _, p := range sh {
			if row, ok := byFrame[strings.TrimSuffix(p.Frame, "I")]; ok {
				row.SharesOut = p.Val
			}
		}
	}
	frames := make([]string, 0, len(byFrame))
	for f := range byFrame {
		frames = append(frames, f)
	}
	sort.Strings(frames)
	cutoff := time.Now().AddDate(-3, 0, 0).Format("2006-01-02")
	out := &model.Fundamentals{Symbol: strings.ToUpper(ticker), CIK: cik}
	for _, f := range frames {
		p := byFrame[f]
		if p.End != "" && p.End < cutoff {
			continue // drop stale frame-aligned data rather than mislabel it as recent
		}
		out.Periods = append(out.Periods, *p)
	}
	return out, nil
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
