package generator

import (
	"fmt"
	"strconv"
)

// lightenHex blends a 6-digit hex color with white by amount in [0,1].
// amount=0 returns the color unchanged; amount=1 returns pure white. Used to
// derive subtle tints from accent/secondary colors for table row shading and
// callout fills, so the palette always stays in the theme's family.
func lightenHex(hex string, amount float64) string {
	r, g, b, ok := parseHex6(hex)
	if !ok {
		return hex
	}
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}
	rr := int(float64(r) + (255-float64(r))*amount)
	gg := int(float64(g) + (255-float64(g))*amount)
	bb := int(float64(b) + (255-float64(b))*amount)
	return fmt.Sprintf("%02X%02X%02X", rr, gg, bb)
}

// hexLuminance approximates perceived luminance (0 dark → 1 light) of a
// 6-digit hex color using Rec.601 coefficients.
func hexLuminance(hex string) float64 {
	r, g, b, ok := parseHex6(hex)
	if !ok {
		return 0.5
	}
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255.0
}

// readableTextOn returns a hex text color that reads cleanly on the given
// background — white for dark fills, near-black for light fills.
func readableTextOn(bg string) string {
	if hexLuminance(bg) < 0.55 {
		return "FFFFFF"
	}
	return "0F172A"
}

func parseHex6(hex string) (int64, int64, int64, bool) {
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	r, err := strconv.ParseInt(hex[0:2], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	g, err := strconv.ParseInt(hex[2:4], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	b, err := strconv.ParseInt(hex[4:6], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return r, g, b, true
}
