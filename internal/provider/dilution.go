package provider

import (
	"context"
	"strings"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

// Dilution builds the shares-outstanding trend from XBRL fundamentals and flags
// whether the share count is growing (dilution) or shrinking (buybacks).
func Dilution(ctx context.Context, h *httpx.Client, ticker string) (*model.DilutionReport, error) {
	f, err := Fundamentals(ctx, h, ticker)
	if err != nil {
		return nil, err
	}
	rep := &model.DilutionReport{Symbol: strings.ToUpper(ticker)}
	for _, p := range f.Periods {
		if p.SharesOut > 0 {
			rep.Points = append(rep.Points, model.Obs{Date: p.End, Value: p.SharesOut})
		}
	}
	if n := len(rep.Points); n >= 5 {
		yrAgo := rep.Points[n-5].Value // ~4 quarters back
		if yrAgo > 0 {
			rep.ChangePctYr = (rep.Points[n-1].Value - yrAgo) / yrAgo * 100
		}
	}
	switch {
	case rep.ChangePctYr > 1:
		rep.Flag = "diluting"
	case rep.ChangePctYr < -1:
		rep.Flag = "buying back"
	default:
		rep.Flag = "flat"
	}
	return rep, nil
}
