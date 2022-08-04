package gokit

import "math"

// Rounds to nearest like 12.3456 -> 12.35
func Round64(val float64, precision int) float64 {
	return math.Round(val*(math.Pow10(precision))) / math.Pow10(precision)
}

// Rounds like 12.3416 -> 12.35
func Round64Up(val float64, precision int) float64 {
	return math.Ceil(val*(math.Pow10(precision))) / math.Pow10(precision)
}

// Rounds like 12.3496 -> 12.34
func Round64Down(val float64, precision int) float64 {
	return math.Floor(val*(math.Pow10(precision))) / math.Pow10(precision)
}
