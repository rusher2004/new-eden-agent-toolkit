package market

import (
	"fmt"
	"strings"
)

// MultibuyLine is one line of an in-game Multibuy list.
type MultibuyLine struct {
	Name     string
	Quantity int64
}

// Multibuy formats lines for the in-game Multibuy "Buy All" import: one
// "Name<TAB>Quantity" line per item, no thousands separators. Duplicate names
// are merged; order follows first appearance.
func Multibuy(lines []MultibuyLine) string {
	idx := map[string]int{}
	var merged []MultibuyLine
	for _, l := range lines {
		if l.Quantity <= 0 || l.Name == "" {
			continue
		}
		if i, ok := idx[l.Name]; ok {
			merged[i].Quantity += l.Quantity
			continue
		}
		idx[l.Name] = len(merged)
		merged = append(merged, l)
	}
	var b strings.Builder
	for i, l := range merged {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s\t%d", l.Name, l.Quantity)
	}
	return b.String()
}
