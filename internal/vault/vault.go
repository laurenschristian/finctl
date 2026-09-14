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

// tables splits text into separate markdown tables. Each table is the cell rows
// of one contiguous block of "|" lines (the --- separator row removed). A note
// with several tables yields several groups, so one table's header is never
// mistaken for another table's data row.
func tables(text string) [][][]string {
	var out [][][]string
	var cur [][]string
	flush := func() {
		if len(cur) > 0 {
			out = append(out, cur)
			cur = nil
		}
	}
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, "|") {
			flush()
			continue
		}
		if strings.Contains(ln, "---") {
			continue
		}
		// Obsidian escapes the alias pipe in [[Name\|TICKER]] as "\|"; shield it
		// so it is not read as a cell delimiter, then restore it inside cells.
		ln = strings.ReplaceAll(ln, "\\|", "\x00")
		cells := strings.Split(strings.Trim(ln, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(strings.ReplaceAll(cells[i], "\x00", "|"))
		}
		cur = append(cur, cells)
	}
	flush()
	return out
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
	var rows [][]string
	var tc, wc int
	for _, t := range tables(text) {
		if len(t) < 2 {
			continue
		}
		tc = colIndex(t[0], "ticker", "symbol")
		wc = colIndex(t[0], "target", "weight", "alloc")
		if tc >= 0 && wc >= 0 {
			rows = t
			break
		}
	}
	if rows == nil {
		return nil
	}
	var out []Target
	for _, r := range rows[1:] {
		if tc >= len(r) || wc >= len(r) {
			continue
		}
		tk := strings.ToUpper(stripLink(r[tc]))
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
	var out []WatchItem
	seen := map[string]bool{}
	for _, t := range tables(text) {
		if len(t) < 2 {
			continue
		}
		header := t[0]
		tc := colIndex(header, "ticker", "symbol")
		bc := colIndex(header, "buy", "zone", "entry")
		th := colIndex(header, "thesis", "note", "reason")
		// A watchlist table needs a ticker column and a buy/zone column; tables
		// without a zone (verdict logs, trade plans) are not buy-zone sources.
		if tc < 0 || bc < 0 {
			continue
		}
		for _, r := range t[1:] {
			if tc >= len(r) {
				continue
			}
			tk := strings.ToUpper(stripLink(r[tc]))
			if !tickerRe.MatchString(tk) || seen[tk] {
				continue
			}
			w := WatchItem{Ticker: tk}
			if bc < len(r) {
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
			seen[tk] = true
			out = append(out, w)
		}
	}
	return out
}

// stripLink reduces an Obsidian cell to a bare ticker: it drops bold/italic
// markers and unwraps [[Target — Name|TICKER]] or [[TICKER]] to the alias.
func stripLink(cell string) string {
	c := strings.TrimSpace(cell)
	c = strings.ReplaceAll(c, "*", "")
	c = strings.Trim(c, "[]")
	if i := strings.LastIndex(c, "|"); i >= 0 {
		c = c[i+1:]
	}
	return strings.TrimSpace(c)
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
