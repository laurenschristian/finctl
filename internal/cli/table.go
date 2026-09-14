package cli

import (
	"strconv"
	"strings"
	"text/tabwriter"
)

func newTab() (*strings.Builder, *tabwriter.Writer) {
	var b strings.Builder
	return &b, tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
}

// money formats a float with thousands separators and two decimals.
func money(f float64) string {
	neg := f < 0
	if neg {
		f = -f
	}
	s := strconv.FormatFloat(f, 'f', 2, 64)
	dot := strings.IndexByte(s, '.')
	intPart, frac := s[:dot], s[dot:]
	var b strings.Builder
	for i, d := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}
	out := b.String() + frac
	if neg {
		return "-" + out
	}
	return out
}

// abbr renders large numbers compactly (1.2B, 345.0M, 12.3K).
func abbr(f float64) string {
	neg := ""
	if f < 0 {
		neg, f = "-", -f
	}
	switch {
	case f >= 1e12:
		return neg + strconv.FormatFloat(f/1e12, 'f', 2, 64) + "T"
	case f >= 1e9:
		return neg + strconv.FormatFloat(f/1e9, 'f', 2, 64) + "B"
	case f >= 1e6:
		return neg + strconv.FormatFloat(f/1e6, 'f', 2, 64) + "M"
	case f >= 1e3:
		return neg + strconv.FormatFloat(f/1e3, 'f', 1, 64) + "K"
	}
	return neg + strconv.FormatFloat(f, 'f', 2, 64)
}

func pctStr(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64) + "%"
}

var sparkRunes = []rune("▁▂▃▄▅▆▇█")

func sparkline(vals []float64) string {
	if len(vals) == 0 {
		return ""
	}
	lo, hi := vals[0], vals[0]
	for _, v := range vals {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	span := hi - lo
	var b strings.Builder
	for _, v := range vals {
		idx := 0
		if span > 0 {
			idx = int((v - lo) / span * float64(len(sparkRunes)-1))
		}
		b.WriteRune(sparkRunes[idx])
	}
	return b.String()
}
