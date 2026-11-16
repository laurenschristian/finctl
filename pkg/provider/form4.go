package provider

import (
	"context"
	"encoding/xml"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/pkg/httpx"
	"github.com/laurenschristian/finctl/pkg/model"
)

// Filed Form 4 documents never change, so cache them for a long time.
const form4TTL = 30 * 24 * time.Hour

type form4Doc struct {
	Owners []struct {
		Name     string `xml:"reportingOwnerId>rptOwnerName"`
		Director string `xml:"reportingOwnerRelationship>isDirector"`
		Officer  string `xml:"reportingOwnerRelationship>isOfficer"`
		TenPct   string `xml:"reportingOwnerRelationship>isTenPercentOwner"`
		Title    string `xml:"reportingOwnerRelationship>officerTitle"`
	} `xml:"reportingOwner"`
	Txns []struct {
		Date   string `xml:"transactionDate>value"`
		Code   string `xml:"transactionCoding>transactionCode"`
		Shares string `xml:"transactionAmounts>transactionShares>value"`
		Price  string `xml:"transactionAmounts>transactionPricePerShare>value"`
	} `xml:"nonDerivativeTable>nonDerivativeTransaction"`
}

func isSet(s string) bool { s = strings.TrimSpace(s); return s == "1" || strings.EqualFold(s, "true") }

// Form4 reads one Form 4 filing and sums its non-derivative rows per transaction code.
// The submissions feed points at the XSL-rendered page; the raw XML sits one directory up.
func Form4(ctx context.Context, h *httpx.Client, f model.Filing) ([]model.InsiderTrade, error) {
	if !strings.HasSuffix(strings.ToLower(f.Doc), ".xml") || !strings.HasSuffix(f.URL, f.Doc) {
		return nil, fmt.Errorf("form 4 %s: primary document is not XML", f.Accession)
	}
	raw := strings.TrimSuffix(f.URL, f.Doc) + path.Base(f.Doc)
	b, err := h.Get(ctx, "edgar", raw, form4TTL)
	if err != nil {
		return nil, edgarErr(err)
	}
	var doc form4Doc
	if err := xml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("form 4 %s: %w", f.Accession, err)
	}
	var owner, role string
	if len(doc.Owners) > 0 {
		o := doc.Owners[0]
		owner = o.Name
		var roles []string
		if t := strings.TrimSpace(o.Title); t != "" {
			roles = append(roles, t)
		} else if isSet(o.Officer) {
			roles = append(roles, "Officer")
		}
		if isSet(o.Director) {
			roles = append(roles, "Director")
		}
		if isSet(o.TenPct) {
			roles = append(roles, "10% owner")
		}
		role = strings.Join(roles, ", ")
	}
	byCode := map[string]*model.InsiderTrade{}
	var order []string
	for _, t := range doc.Txns {
		code := strings.TrimSpace(t.Code)
		if code == "" {
			continue
		}
		it, ok := byCode[code]
		if !ok {
			it = &model.InsiderTrade{Filed: f.FilingDate, Owner: owner, Role: role, Code: code, URL: f.URL}
			byCode[code] = it
			order = append(order, code)
		}
		sh := atof(strings.TrimSpace(t.Shares))
		it.Shares += sh
		it.Value += sh * atof(strings.TrimSpace(t.Price))
		if d := strings.TrimSpace(t.Date); d > it.Date {
			it.Date = d
		}
	}
	out := make([]model.InsiderTrade, 0, len(order))
	for _, c := range order {
		it := byCode[c]
		if it.Shares > 0 {
			it.Price = it.Value / it.Shares
		}
		out = append(out, *it)
	}
	return out, nil
}

// InsiderTrades parses the most recent Form 4 filings (one request each, so keep n small).
func InsiderTrades(ctx context.Context, h *httpx.Client, ticker string, n int) ([]model.InsiderTrade, error) {
	fs, err := Insider(ctx, h, ticker, n)
	if err != nil {
		return nil, err
	}
	return form4Trades(ctx, h, fs)
}

// InsiderTradesByCIK is InsiderTrades for a known CIK; it skips the ticker map.
func InsiderTradesByCIK(ctx context.Context, h *httpx.Client, cik string, n int) ([]model.InsiderTrade, error) {
	fs, err := InsiderByCIK(ctx, h, cik, n)
	if err != nil {
		return nil, err
	}
	return form4Trades(ctx, h, fs)
}

func form4Trades(ctx context.Context, h *httpx.Client, fs []model.Filing) ([]model.InsiderTrade, error) {
	var out []model.InsiderTrade
	var lastErr error
	for _, f := range fs {
		ts, err := Form4(ctx, h, f)
		if err != nil {
			lastErr = err
			continue
		}
		out = append(out, ts...)
	}
	if len(out) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return out, nil
}
