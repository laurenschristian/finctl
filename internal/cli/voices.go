package cli

import (
	"fmt"
	"strings"

	"github.com/laurenschristian/finctl/internal/provider"
	"github.com/spf13/cobra"
)

func voicesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "voices",
		Short: "Curated market voices from X, read as data (fxtwitter, keyless)",
		RunE: func(cmd *cobra.Command, args []string) error {
			vs := provider.Voices(cmd.Context(), hx, args)
			return show(vs, func() string {
				b, w := newTab()
				fmt.Fprintln(w, "HANDLE\tFOLLOWERS\tPOSTS\tBIO")
				for _, v := range vs {
					if v.Err != "" {
						fmt.Fprintf(w, "@%s\t(error)\t\t%s\n", v.Handle, v.Err)
						continue
					}
					bio := strings.ReplaceAll(strings.ReplaceAll(v.Bio, "\n", " "), "\r", "")
					if len(bio) > 48 {
						bio = bio[:48]
					}
					fmt.Fprintf(w, "@%s\t%s\t%s\t%s\n", v.Handle, abbr(float64(v.Followers)), abbr(float64(v.Posts)), bio)
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
	c.AddCommand(voicesTickersCmd())
	return c
}

func voicesTickersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tickers",
		Short: "Tickers trending across the whitelist (needs a timeline source)",
		RunE: func(_ *cobra.Command, _ []string) error {
			// The timeline path (CDP/mirror) is intentionally not wired yet.
			return fmt.Errorf("timeline source unavailable: voices tickers needs a logged-in timeline reader (planned); single-post `voice read` and `voices` profiles work now")
		},
	}
}

func voiceCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "voice",
		Short: "Read X posts",
	}
	c.AddCommand(&cobra.Command{
		Use:   "read <url|handle/id>",
		Short: "Hydrate a single tweet or thread to text (fxtwitter)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := provider.VoiceRead(cmd.Context(), hx, args[0])
			if err != nil {
				return err
			}
			return show(p, func() string {
				b, w := newTab()
				fmt.Fprintf(w, "@%s  %s\n", p.Author, p.Time)
				fmt.Fprintf(w, "%s\n", p.Text)
				if p.QuotedText != "" {
					fmt.Fprintf(w, "> quoting: %s\n", p.QuotedText)
				}
				fmt.Fprintf(w, "%s likes  %s reposts  %s replies  %s views\n",
					abbr(float64(p.Likes)), abbr(float64(p.Reposts)), abbr(float64(p.Replies)), abbr(float64(p.Views)))
				if len(p.Tickers) > 0 {
					fmt.Fprintf(w, "tickers: %v\n", p.Tickers)
				}
				_ = w.Flush()
				return b.String()
			})
		},
	})
	return c
}
