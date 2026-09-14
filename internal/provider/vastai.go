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

const vastTTL = 10 * time.Minute

var vastBase = "https://console.vast.ai/api/v0"

// GPUDefaults are the compute-cost bellwethers reported by gpu-rent.
var GPUDefaults = []string{"H100 SXM", "B200"}

// GPURent returns the lowest ask and median $/GPU-hour for a GPU type on vast.ai
// on-demand offers (keyless). num_gpus normalizes multi-GPU bundles to per-GPU.
func GPURent(ctx context.Context, h *httpx.Client, gpu string) (*model.GPURent, error) {
	q := fmt.Sprintf(`{"gpu_name":{"eq":%q},"rentable":{"eq":true},"order":[["dph_total","asc"]],"limit":128,"type":"on-demand"}`, gpu)
	u := vastBase + "/bundles/?q=" + url.QueryEscape(q)
	b, err := h.Get(ctx, "vastai", u, vastTTL)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Offers []struct {
			DphTotal float64 `json:"dph_total"`
			NumGPUs  float64 `json:"num_gpus"`
		} `json:"offers"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	var per []float64
	for _, o := range raw.Offers {
		n := o.NumGPUs
		if n <= 0 {
			n = 1
		}
		if o.DphTotal > 0 {
			per = append(per, o.DphTotal/n)
		}
	}
	if len(per) == 0 {
		return nil, fmt.Errorf("vastai: no rentable %s offers right now", gpu)
	}
	sort.Float64s(per)
	return &model.GPURent{
		GPU:    gpu,
		LowAsk: per[0],
		Median: per[len(per)/2],
		Offers: len(per),
		AsOf:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}
