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

const finraTTL = 12 * time.Hour

var finraBase = "https://api.finra.org"

// ShortInterest returns the latest FINRA consolidated short-interest settlement
// for a symbol (keyless POST). FINRA rejects sort-by-date without a partition
// key, so we fetch recent rows and pick the latest settlement in code.
func ShortInterest(ctx context.Context, h *httpx.Client, symbol string) (*model.ShortInterest, error) {
	symbol = strings.ToUpper(symbol)
	body := fmt.Sprintf(`{"limit":1000,"compareFilters":[{"compareType":"EQUAL","fieldName":"symbolCode","fieldValue":%q}]}`, symbol)
	b, err := h.PostJSON(ctx, "finra", finraBase+"/data/group/otcMarket/name/consolidatedShortInterest", []byte(body), finraTTL)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Symbol      string  `json:"symbolCode"`
		Settlement  string  `json:"settlementDate"`
		Current     float64 `json:"currentShortPositionQuantity"`
		Previous    float64 `json:"previousShortPositionQuantity"`
		AvgVol      float64 `json:"averageDailyVolumeQuantity"`
		DaysToCover float64 `json:"daysToCoverQuantity"`
		ChangePct   float64 `json:"changePercent"`
	}
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("finra: no short interest for %s", symbol)
	}
	latest := rows[0]
	for _, r := range rows {
		if r.Settlement > latest.Settlement {
			latest = r
		}
	}
	return &model.ShortInterest{
		Symbol:         symbol,
		SettlementDate: latest.Settlement,
		Current:        latest.Current,
		Previous:       latest.Previous,
		ChangePct:      latest.ChangePct,
		DaysToCover:    latest.DaysToCover,
		AvgDailyVol:    latest.AvgVol,
	}, nil
}
