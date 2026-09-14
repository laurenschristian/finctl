package cli

import (
	"fmt"
	"time"

	"github.com/laurenschristian/finctl/internal/model"
	"github.com/laurenschristian/finctl/internal/provider"
	"github.com/spf13/cobra"
)

func twRevenueCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "tw-revenue [id...]",
		Aliases: []string{"twrev"},
		Short:   "TWSE monthly revenue with MoM/YoY (default = AI-server ODM basket)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rows, err := provider.TWRevenue(cmd.Context(), hx, args)
			if err != nil {
				return err
			}
			return show(rows, func() string {
				b, w := newTab()
				fmt.Fprintln(w, "ID\tNAME\tMONTH\tREVENUE(TWD)\tMOM\tYOY")
				for _, r := range rows {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
						r.CompanyID, r.Name, r.Month, abbr(r.Revenue), pctStr(r.MoM), pctStr(r.YoY))
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
}

func gpuRentCmd() *cobra.Command {
	var trend bool
	c := &cobra.Command{
		Use:   "gpu-rent [gpu...]",
		Short: "Lowest ask and median $/GPU-hour on vast.ai (compute-cost proxy)",
		RunE: func(cmd *cobra.Command, args []string) error {
			gpus := args
			if len(gpus) == 0 {
				gpus = provider.GPUDefaults
			}
			type row struct {
				Rent  *model.GPURent `json:"rent,omitempty"`
				Trend []float64      `json:"trend,omitempty"`
				Err   string         `json:"error,omitempty"`
			}
			var rows []row
			for _, g := range gpus {
				r, err := provider.GPURent(cmd.Context(), hx, g)
				if err != nil {
					rows = append(rows, row{Err: err.Error()})
					continue
				}
				var hist []float64
				if store != nil {
					_ = store.Append(cmd.Context(), "gpu:"+g, time.Now().Unix(), r.LowAsk)
					if trend {
						hist, _ = store.Series(cmd.Context(), "gpu:"+g)
					}
				}
				rows = append(rows, row{Rent: r, Trend: hist})
			}
			return show(rows, func() string {
				b, w := newTab()
				fmt.Fprintln(w, "GPU\tLOW $/hr\tMEDIAN\tOFFERS\tTREND")
				for _, r := range rows {
					if r.Rent == nil {
						fmt.Fprintf(w, "\t\t\t\t%s\n", r.Err)
						continue
					}
					spark := ""
					if trend && len(r.Trend) > 1 {
						spark = sparkline(r.Trend)
					}
					fmt.Fprintf(w, "%s\t$%.3f\t$%.3f\t%d\t%s\n",
						r.Rent.GPU, r.Rent.LowAsk, r.Rent.Median, r.Rent.Offers, spark)
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
	c.Flags().BoolVar(&trend, "trend", false, "show a sparkline of stored low-ask points")
	return c
}
