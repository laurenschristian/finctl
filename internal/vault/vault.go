// Package vault reads target weights and watchlist buy-zones from the user's
// Obsidian markdown vault. It is read-only: finctl never writes the vault.
package vault

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Target is a portfolio target weight for a ticker (0..1).
type Target struct {
	Ticker string
	Weight float64
}

// WatchItem is a watchlist entry with an optional buy-zone and thesis line.
type WatchItem struct {
	Ticker  string
	BuyLow  float64
	BuyHigh float64
	Thesis  string
}

var pctRe = regexp.MustCompile(`(-?\d+(?:\.\d+)?)\s*%`)
var moneyRe = regexp.MustCompile(`\$?\s*(\d+(?:\.\d+)?)`)
var tickerRe = regexp.MustCompile(`^[A-Z][A-Z.\-]{0,6}$`)

// tableRows returns the split cells of each markdown table row in text, minus
// the header separator row (---).
func tableRows(text string) [][]string {
	var rows [][]string
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, "|") {
			continue
		}
		if strings.Contains(ln, "---") {
			continue
		}
		cells := strings.Split(strings.Trim(ln, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		rows = append(rows, cells)
	}
	return rows
}

// colIndex finds the first column whose header contains any of the keywords.
func colIndex(header []string, keywords ...string) int {
	for i, h := range header {
		lh := strings.ToLower(h)
		for _, k := range keywords {
			if strings.Contains(lh, k) {
				return i
			}
		}
	}
	return -1
}

// ParseTargets extracts ticker -> target weight from the first markdown table
// that has a ticker column and a target/weight column. Weights are normalized
// to fractions (25% -> 0.25).
func ParseTargets(text string) []Target {
	rows := tableRows(text)
	if len(rows) < 2 {
		return nil
	}
	header := rows[0]
	tc := colIndex(header, "ticker", "symbol")
	wc := colIndex(header, "target", "weight", "alloc")
	if tc < 0 || wc < 0 {
		return nil
	}
	var out []Target
	for _, r := range rows[1:] {
		if tc >= len(r) || wc >= len(r) {
			continue
		}
		tk := strings.ToUpper(r[tc])
		if !tickerRe.MatchString(tk) {
			continue
		}
		m := pctRe.FindStringSubmatch(r[wc])
		if m == nil {
			continue
		}
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		out = append(out, Target{Ticker: tk, Weight: v / 100})
	}
	return out
}

// ParseWatchlist extracts watch items (ticker, buy-zone, thesis) from the first
// markdown table with a ticker column and a buy/zone column.
func ParseWatchlist(text string) []WatchItem {
	rows := tableRows(text)
	if len(rows) < 2 {
		return nil
	}
	header := rows[0]
	tc := colIndex(header, "ticker", "symbol")
	bc := colIndex(header, "buy", "zone", "entry")
	th := colIndex(header, "thesis", "note", "reason")
	if tc < 0 {
		return nil
	}
	var out []WatchItem
	for _, r := range rows[1:] {
		if tc >= len(r) {
			continue
		}
		tk := strings.ToUpper(r[tc])
		if !tickerRe.MatchString(tk) {
			continue
		}
		w := WatchItem{Ticker: tk}
		if bc >= 0 && bc < len(r) {
			nums := moneyRe.FindAllStringSubmatch(r[bc], -1)
			if len(nums) >= 1 {
				w.BuyLow, _ = strconv.ParseFloat(nums[0][1], 64)
			}
			if len(nums) >= 2 {
				w.BuyHigh, _ = strconv.ParseFloat(nums[1][1], 64)
			}
		}
		if th >= 0 && th < len(r) {
			w.Thesis = r[th]
		}
		out = append(out, w)
	}
	return out
}

// readNote returns the content of the first file under dir whose name contains
// name (case-insensitive), or empty if none.
func readNote(dir, name string) string {
	var found string
	name = strings.ToLower(name)
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || found != "" {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".md") &&
			strings.Contains(strings.ToLower(filepath.Base(path)), name) {
			if b, err := os.ReadFile(path); err == nil {
				found = string(b)
			}
		}
		return nil
	})
	return found
}

// Targets reads and parses the Dashboard target table from the vault.
func Targets(dir string) []Target {
	if dir == "" {
		return nil
	}
	return ParseTargets(readNote(dir, "dashboard"))
}

// Watchlist reads and parses the watchlist note from the vault.
func Watchlist(dir string) []WatchItem {
	if dir == "" {
		return nil
	}
	return ParseWatchlist(readNote(dir, "watchlist"))
}
