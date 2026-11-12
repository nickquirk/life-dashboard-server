package domain

// Swatches are the named colours a thing can be given. Must match the
// client's Swatch type (life-dashboard-client/src/lib/swatches.ts).
var Swatches = map[string]struct{}{
	"gray":   {},
	"blue":   {},
	"green":  {},
	"yellow": {},
	"orange": {},
	"pink":   {},
	"indigo": {},
}

// swatchPickOrder is the order PickSwatch tries swatches in; gray is last
// because it reads as "no colour".
var swatchPickOrder = []string{"blue", "green", "yellow", "orange", "pink", "indigo", "gray"}

// PickSwatch returns the least-used swatch in used, so a new thing avoids
// colours already taken where possible. Ties go by swatchPickOrder.
func PickSwatch(used []string) string {
	counts := map[string]int{}
	for _, c := range used {
		counts[c]++
	}
	best := swatchPickOrder[0]
	for _, s := range swatchPickOrder[1:] {
		if counts[s] < counts[best] {
			best = s
		}
	}
	return best
}

func IsSwatch(s string) bool {
	_, ok := Swatches[s]
	return ok
}
