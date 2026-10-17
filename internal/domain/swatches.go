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

func IsSwatch(s string) bool {
	_, ok := Swatches[s]
	return ok
}
