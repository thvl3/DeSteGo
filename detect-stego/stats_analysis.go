package main

import (
	"image"
	"math"
)

// ChiSquareLSB calculates a basic chi-square statistic on the distribution of even/odd values for a specific color channel.
// If the distribution is suspiciously uniform, this might indicate LSB stego.
// channel should be 'R', 'G', or 'B'.
func ChiSquareLSB(img image.Image, channel byte) float64 {
	bounds := img.Bounds()
	var evenCount, oddCount int

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			var value uint8

			switch channel {
			case 'R':
				value = uint8(r >> 8)
			case 'G':
				value = uint8(g >> 8)
			case 'B':
				value = uint8(b >> 8)
			default:
				// Default to red if invalid channel specified
				value = uint8(r >> 8)
			}

			if value%2 == 0 {
				evenCount++
			} else {
				oddCount++
			}
		}
	}

	total := evenCount + oddCount
	if total == 0 {
		return 0
	}

	// Expected if random: ~ half even, half odd
	expected := float64(total) / 2.0
	// Chi-square for 2 categories
	chi := math.Pow(float64(evenCount)-expected, 2)/expected +
		math.Pow(float64(oddCount)-expected, 2)/expected

	return chi
}

// IsSuspiciousChiSquare returns true if the chi-square value is below an arbitrary threshold
// indicating an unusually even distribution (which might suggest LSB stego).
func IsSuspiciousChiSquare(chi float64) bool {
	// if it's very low, distribution might be too uniform
	return chi < 0.5
}
