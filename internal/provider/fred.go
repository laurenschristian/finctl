package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

const fredTTL = 6 * time.Hour

var fredBase = "https://api.stlouisfed.org/fred"

// FredSeries returns the last n observations of a FRED series (needs a key).
func FredSeries(ctx context.Context, h *httpx.Client, key, id string, n int) (*model.Series, error) {
	if key == "" {
		return nil, fmt.Errorf("FRED needs a free key: set FINCTL_FRED_KEY (register at fred.stlouisfed.org/docs/api/api_key.html)")
	}
	if n <= 0 {
		n = 12
	}
	q := url.Values{}
	q.Set("series_id", id)
	q.Set("api_key", key)
	q.Set("file_type", "json")
	q.Set("sort_order", "desc")
	q.Set("limit", fmt.Sprintf("%d", n))
	b, err := h.Get(ctx, "fred", fredBase+"/series/observations?"+q.Encode(), fredTTL)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Observations []struct {
			Date  string `json:"date"`
			Value string `json:"value"`
		} `json:"observations"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	s := &model.Series{ID: id}
	for _, o := range raw.Observations {
		if o.Value == "." { // FRED marks missing values with a dot
			continue
		}
		s.Points = append(s.Points, model.Obs{Date: o.Date, Value: atof(o.Value)})
	}
	sort.Slice(s.Points, func(i, j int) bool { return s.Points[i].Date < s.Points[j].Date })
	if len(s.Points) == 0 {
		return nil, fmt.Errorf("fred: no observations for %s", id)
	}
	return s, nil
}
