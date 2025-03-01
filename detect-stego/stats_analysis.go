package main

import (
	"image"
	"math"
)

// ChiSquareLSB calculates a basic chi-square statistic based on how many
// red-channel values are even vs. odd. If it's too close to uniform (chi-square low),
// that may indicate hidden LSB data.
func ChiSquareLSB(img image.Image) float64 {
	bounds := img.Bounds()
	var evenCount, oddCount int

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			R := uint8(r >> 8)
			if R%2 == 0 {
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

	expected := float64(total) / 2.0
	// Chi-square for 2 categories: ((O1 - E)^2 / E) + ((O2 - E)^2 / E)
	chi := math.Pow(float64(evenCount)-expected, 2)/expected +
		math.Pow(float64(oddCount)-expected, 2)/expected

	return chi
}

// IsSuspiciousChiSquare is a naive function to judge if the chi-square
// result is suspiciously low, implying uniform distribution of even/odd.
func IsSuspiciousChiSquare(chi float64) bool {
	// If the chi-square is below some small threshold, we consider it suspicious.
	// This threshold is arbitrary and for demonstration only.
	return chi < 0.5
}
