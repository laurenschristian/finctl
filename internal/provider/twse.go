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

const twseTTL = 24 * time.Hour

var twseBase = "https://openapi.twse.com.tw/v1"

// TWODMBasket is the default AI-server supply-chain set: TSMC, MediaTek, Wiwynn,
// Quanta, Hon Hai (Foxconn), ASE.
var TWODMBasket = []string{"2330", "2454", "6669", "2382", "2317", "3711"}

// TWRevenue returns the latest monthly revenue (TWD) with MoM/YoY for the given
// company ids from the TWSE OpenAPI monthly revenue feed (keyless). Empty ids
// returns the AI-server ODM basket.
func TWRevenue(ctx context.Context, h *httpx.Client, ids []string) ([]model.MonthlyRevenue, error) {
	if len(ids) == 0 {
		ids = TWODMBasket
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[strings.TrimSpace(id)] = true
	}
	b, err := h.Get(ctx, "twse", twseBase+"/opendata/t187ap05_L", twseTTL)
	if err != nil {
		return nil, err
	}
	var rows []map[string]string
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, err
	}
	byID := map[string]model.MonthlyRevenue{}
	for _, r := range rows {
		id := r["公司代號"]
		if !want[id] {
			continue
		}
		byID[id] = model.MonthlyRevenue{
			CompanyID: id,
			Name:      r["公司名稱"],
			Month:     rocMonth(r["資料年月"]),
			Revenue:   atof(r["營業收入-當月營收"]) * 1000, // feed reports NT$ thousands
			MoM:       atof(r["營業收入-上月比較增減(%)"]),
			YoY:       atof(r["營業收入-去年同月增減(%)"]),
		}
	}
	// Preserve the requested order.
	var out []model.MonthlyRevenue
	for _, id := range ids {
		if mr, ok := byID[strings.TrimSpace(id)]; ok {
			out = append(out, mr)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("twse: none of %v found in the monthly revenue feed", ids)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Revenue > out[j].Revenue })
	return out, nil
}

// rocMonth converts a TWSE ROC year-month (e.g. "11508") to Gregorian "2026-08".
func rocMonth(s string) string {
	if len(s) < 5 {
		return s
	}
	var roc int
	if _, err := fmt.Sscanf(s[:len(s)-2], "%d", &roc); err != nil {
		return s
	}
	return fmt.Sprintf("%d-%s", roc+1911, s[len(s)-2:])
}
