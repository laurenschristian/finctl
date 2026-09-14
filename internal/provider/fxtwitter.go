package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

const (
	fxProfileTTL = 6 * time.Hour
	fxPostTTL    = 15 * time.Minute
)

var fxBase = "https://api.fxtwitter.com"

// VoicesWhitelist is the default curated set of market voices (docs/voices.md).
var VoicesWhitelist = []string{
	"dylan522p", "Beth_Kindig", "HFI_Research", "unusual_whales", "KobeissiLetter",
	"NickTimiraos", "LizAnnSonders", "charliebilello", "MacroCharts", "spotgamma",
	"JavierBlas", "zerohedge", "Barchart", "lisaabramowicz1", "ian_cutress",
	"rwang07", "dnystedt", "Jukan05", "jukanlosreve", "kakashiii111",
}

var cashtagRe = regexp.MustCompile(`\$([A-Za-z]{1,6})\b`)

// VoiceProfile fetches one handle's public profile from fxtwitter (keyless).
func VoiceProfile(ctx context.Context, h *httpx.Client, handle string) (*model.Voice, error) {
	handle = strings.TrimPrefix(strings.TrimSpace(handle), "@")
	b, err := h.Get(ctx, "fxtwitter", fxBase+"/"+handle, fxProfileTTL)
	if err != nil {
		return nil, err
	}
	var raw struct {
		User struct {
			ScreenName  string `json:"screen_name"`
			Name        string `json:"name"`
			Followers   int64  `json:"followers"`
			Tweets      int64  `json:"tweets"`
			Description string `json:"description"`
		} `json:"user"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	if raw.User.ScreenName == "" {
		return nil, fmt.Errorf("fxtwitter: no profile for %s", handle)
	}
	return &model.Voice{
		Handle:    raw.User.ScreenName,
		Name:      raw.User.Name,
		Followers: raw.User.Followers,
		Posts:     raw.User.Tweets,
		Bio:       raw.User.Description,
	}, nil
}

// Voices fetches profiles for a set of handles, tolerating per-handle failures.
func Voices(ctx context.Context, h *httpx.Client, handles []string) []model.Voice {
	if len(handles) == 0 {
		handles = VoicesWhitelist
	}
	out := make([]model.Voice, 0, len(handles))
	for _, hn := range handles {
		v, err := VoiceProfile(ctx, h, hn)
		if err != nil {
			out = append(out, model.Voice{Handle: strings.TrimPrefix(hn, "@"), Err: err.Error()})
			continue
		}
		out = append(out, *v)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Followers > out[j].Followers })
	return out
}

var statusURLRe = regexp.MustCompile(`(?:twitter\.com|x\.com)/([^/]+)/status/(\d+)`)

// VoiceRead hydrates a single tweet from a URL or a handle/id pair via fxtwitter.
func VoiceRead(ctx context.Context, h *httpx.Client, ref string) (*model.Post, error) {
	var handle, id string
	if m := statusURLRe.FindStringSubmatch(ref); m != nil {
		handle, id = m[1], m[2]
	} else if parts := strings.Split(strings.TrimPrefix(ref, "@"), "/"); len(parts) == 2 {
		handle, id = parts[0], parts[1]
	} else {
		return nil, fmt.Errorf("voice read: pass a tweet URL or handle/id, got %q", ref)
	}
	b, err := h.Get(ctx, "fxtwitter", fmt.Sprintf("%s/%s/status/%s", fxBase, handle, id), fxPostTTL)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Tweet struct {
			ID        string `json:"id"`
			Text      string `json:"text"`
			CreatedAt string `json:"created_at"`
			Likes     int64  `json:"likes"`
			Retweets  int64  `json:"retweets"`
			Replies   int64  `json:"replies"`
			Views     int64  `json:"views"`
			URL       string `json:"url"`
			Author    struct {
				ScreenName string `json:"screen_name"`
			} `json:"author"`
			Quote struct {
				Text string `json:"text"`
			} `json:"quote"`
		} `json:"tweet"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	tw := raw.Tweet
	if tw.Text == "" && tw.ID == "" {
		return nil, fmt.Errorf("fxtwitter: no tweet at %s/%s", handle, id)
	}
	p := &model.Post{
		Handle:     tw.Author.ScreenName,
		Author:     tw.Author.ScreenName,
		ID:         tw.ID,
		Text:       tw.Text,
		Time:       tw.CreatedAt,
		Likes:      tw.Likes,
		Reposts:    tw.Retweets,
		Replies:    tw.Replies,
		Views:      tw.Views,
		URL:        tw.URL,
		QuotedText: tw.Quote.Text,
		Tickers:    extractTickers(tw.Text),
	}
	return p, nil
}

// extractTickers pulls $CASHTAG symbols from text, upper-cased and deduped.
func extractTickers(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range cashtagRe.FindAllStringSubmatch(text, -1) {
		t := strings.ToUpper(m[1])
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}
