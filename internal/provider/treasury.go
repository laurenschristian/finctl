package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

const treasuryTTL = 12 * time.Hour

var (
	treasuryBase   = "https://home.treasury.gov"
	fiscalDataBase = "https://api.fiscaldata.treasury.gov"
)

// curveTenors maps the Treasury XML fields to display tenors, in curve order.
var curveTenors = []struct{ field, tenor string }{
	{"BC_1MONTH", "1M"}, {"BC_2MONTH", "2M"}, {"BC_3MONTH", "3M"},
	{"BC_4MONTH", "4M"}, {"BC_6MONTH", "6M"}, {"BC_1YEAR", "1Y"},
	{"BC_2YEAR", "2Y"}, {"BC_3YEAR", "3Y"}, {"BC_5YEAR", "5Y"},
	{"BC_7YEAR", "7Y"}, {"BC_10YEAR", "10Y"}, {"BC_20YEAR", "20Y"},
	{"BC_30YEAR", "30Y"},
}

// TreasuryCurve returns the latest daily par yield curve (keyless).
func TreasuryCurve(ctx context.Context, h *httpx.Client) ([]model.RatePoint, error) {
	m := time.Now().Format("200601")
	u := fmt.Sprintf("%s/resource-center/data-chart-center/interest-rates/pages/xml?data=daily_treasury_yield_curve&field_tdr_date_value_month=%s", treasuryBase, m)
	b, err := h.Get(ctx, "treasury", u, treasuryTTL)
	if err != nil {
		return nil, err
	}
	// The feed is an Atom document; each entry's properties hold the tenors as
	// namespaced elements (d:BC_10YEAR), so walk the inner tokens by local name.
	var feed struct {
		Entries []struct {
			Props struct {
				Inner []byte `xml:",innerxml"`
			} `xml:"content>properties"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(b, &feed); err != nil {
		return nil, err
	}
	if len(feed.Entries) == 0 {
		return nil, fmt.Errorf("treasury: no curve rows for %s", m)
	}
	var latest map[string]string
	var latestDate string
	for _, e := range feed.Entries {
		flat := map[string]string{}
		dec := xml.NewDecoder(bytes.NewReader(e.Props.Inner))
		var cur string
		for {
			tok, err := dec.Token()
			if err != nil {
				break
			}
			switch t := tok.(type) {
			case xml.StartElement:
				cur = t.Name.Local
			case xml.CharData:
				if cur != "" {
					flat[cur] += string(t)
				}
			case xml.EndElement:
				cur = ""
			}
		}
		if d := flat["NEW_DATE"]; d > latestDate {
			latestDate, latest = d, flat
		}
	}
	asOf := latestDate
	if len(asOf) >= 10 {
		asOf = asOf[:10]
	}
	var out []model.RatePoint
	for _, ct := range curveTenors {
		s, ok := latest[ct.field]
		if !ok || s == "" {
			continue
		}
		var v float64
		if _, err := fmt.Sscanf(s, "%g", &v); err != nil || v == 0 {
			continue
		}
		out = append(out, model.RatePoint{Tenor: ct.tenor, Yield: v, AsOf: asOf})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("treasury: no tenors parsed")
	}
	return out, nil
}

// TreasuryDebt returns the latest debt-to-the-penny snapshot (keyless).
func TreasuryDebt(ctx context.Context, h *httpx.Client) (*model.DebtSummary, error) {
	u := fiscalDataBase + "/services/api/fiscal_service/v2/accounting/od/debt_to_penny?sort=-record_date&page%5Bsize%5D=1"
	b, err := h.Get(ctx, "treasury", u, treasuryTTL)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Data []struct {
			Date       string `json:"record_date"`
			HeldPublic string `json:"debt_held_public_amt"`
			Intragov   string `json:"intragov_hold_amt"`
			Total      string `json:"tot_pub_debt_out_amt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	if len(raw.Data) == 0 {
		return nil, fmt.Errorf("fiscaldata: no debt rows")
	}
	d := raw.Data[0]
	return &model.DebtSummary{
		Date: d.Date, TotalDebt: atof(d.Total),
		HeldPublic: atof(d.HeldPublic), Intragov: atof(d.Intragov),
	}, nil
}

// atof parses a decimal string, returning 0 on any error.
func atof(s string) float64 {
	var v float64
	_, _ = fmt.Sscanf(s, "%g", &v)
	return v
}
