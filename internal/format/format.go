package format

import (
	"fmt"
	"math"
)

// OnOff returns a human-readable on/off status.
func OnOff(on bool) string {
	if on {
		return "ON"
	}
	return "OFF"
}

// Brightness formats a brightness percentage.
func Brightness(b float64) string {
	return fmt.Sprintf("%.0f%%", b)
}

// XYToRGBHex converts CIE xy + brightness to an approximate hex RGB string.
func XYToRGBHex(x, y, bri float64) string {
	// Convert CIE xy to XYZ
	z := 1.0 - x - y
	if y == 0 {
		return "#000000"
	}
	Y := bri / 100.0
	X := (Y / y) * x
	Z := (Y / y) * z

	// XYZ to sRGB (Wide RGB D65 conversion)
	r := X*1.656492 - Y*0.354851 - Z*0.255038
	g := -X*0.707196 + Y*1.655397 + Z*0.036152
	b := X*0.051713 - Y*0.121364 + Z*1.011530

	// Gamma correction
	r = gammaCorrect(r)
	g = gammaCorrect(g)
	b = gammaCorrect(b)

	// Clamp
	r = clamp(r, 0, 1)
	g = clamp(g, 0, 1)
	b = clamp(b, 0, 1)

	return fmt.Sprintf("#%02X%02X%02X", int(r*255), int(g*255), int(b*255))
}

func gammaCorrect(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return (1.0+0.055)*math.Pow(v, 1.0/2.4) - 0.055
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// MirekToKelvin converts mirek color temperature to Kelvin.
func MirekToKelvin(mirek int) int {
	if mirek == 0 {
		return 0
	}
	return 1000000 / mirek
}

// HexToXY converts a hex RGB string to CIE xy coordinates.
func HexToXY(hex string) (float64, float64, error) {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 0, 0, fmt.Errorf("invalid hex color: must be 6 characters")
	}

	var r, g, b int
	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hex color: %w", err)
	}

	// Normalize to 0-1
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	// Reverse gamma correction
	rf = reverseGamma(rf)
	gf = reverseGamma(gf)
	bf = reverseGamma(bf)

	// sRGB to XYZ
	X := rf*0.664511 + gf*0.154324 + bf*0.162028
	Y := rf*0.283881 + gf*0.668433 + bf*0.047685
	Z := rf*0.000088 + gf*0.072310 + bf*0.986039

	sum := X + Y + Z
	if sum == 0 {
		return 0, 0, nil
	}

	return X / sum, Y / sum, nil
}

func reverseGamma(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/(1.0+0.055), 2.4)
}
