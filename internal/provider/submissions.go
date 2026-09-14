package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

const submissionsTTL = 6 * time.Hour

// Filings returns recent SEC submissions for a ticker, optionally filtered to a
// form type (e.g. 8-K, 10-K, 4). limit caps the rows.
func Filings(ctx context.Context, h *httpx.Client, ticker, form string, limit int) ([]model.Filing, error) {
	cik, err := CIKFor(ctx, h, ticker)
	if err != nil {
		return nil, err
	}
	b, err := h.Get(ctx, "edgar", fmt.Sprintf("%s/submissions/%s.json", edgarData, cik), submissionsTTL)
	if err != nil {
		return nil, edgarErr(err)
	}
	var raw struct {
		Filings struct {
			Recent struct {
				Accession  []string `json:"accessionNumber"`
				Form       []string `json:"form"`
				FilingDate []string `json:"filingDate"`
				ReportDate []string `json:"reportDate"`
				PrimaryDoc []string `json:"primaryDocument"`
				PrimaryDsc []string `json:"primaryDocDescription"`
			} `json:"recent"`
		} `json:"filings"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	r := raw.Filings.Recent
	form = strings.ToUpper(strings.TrimSpace(form))
	cikNum := strings.TrimLeft(strings.TrimPrefix(cik, "CIK"), "0")
	if limit <= 0 {
		limit = 20
	}
	var out []model.Filing
	for i := range r.Form {
		if form != "" && strings.ToUpper(r.Form[i]) != form {
			continue
		}
		f := model.Filing{
			Form:       r.Form[i],
			FilingDate: strAt(r.FilingDate, i),
			ReportDate: strAt(r.ReportDate, i),
			Accession:  r.Accession[i],
			Doc:        strAt(r.PrimaryDoc, i),
			Desc:       strAt(r.PrimaryDsc, i),
		}
		acc := strings.ReplaceAll(f.Accession, "-", "")
		f.URL = fmt.Sprintf("%s/Archives/edgar/data/%s/%s/%s", edgarWWW, cikNum, acc, f.Doc)
		out = append(out, f)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// Insider returns recent Form 4 (and 4/A) insider filings for a ticker.
func Insider(ctx context.Context, h *httpx.Client, ticker string, limit int) ([]model.Filing, error) {
	all, err := Filings(ctx, h, ticker, "", 400)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	var out []model.Filing
	for _, f := range all {
		if f.Form == "4" || f.Form == "4/A" {
			out = append(out, f)
			if len(out) >= limit {
				break
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no recent Form 4 filings for %s", strings.ToUpper(ticker))
	}
	return out, nil
}

// at safely indexes a possibly-short parallel array.
func strAt(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}
