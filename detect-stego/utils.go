package main

import (
	"image"
	"image/png"
	"io"
	"math"
	"os"
)

// LoadPNG loads a PNG from disk into an image.Image.
func LoadPNG(filename string) (image.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

// DecodePNGFromReader decodes a PNG from an io.Reader into an image.Image.
func DecodePNGFromReader(r io.Reader) (image.Image, error) {
	return png.Decode(r)
}

// IsASCIIPrintable checks if a byte slice is predominantly printable ASCII.
// If >80% of bytes are in [32..126, \n, \r, \t], we consider it ASCII text.
func IsASCIIPrintable(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	printableCount := 0
	for _, b := range data {
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
			printableCount++
		}
	}
	ratio := float64(printableCount) / float64(len(data))
	return ratio > 0.8
}

// ComputeEntropy calculates the Shannon entropy of the data.
// If the data is highly random (encrypted/compressed), it will have high entropy.
func ComputeEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	// Count frequency of each byte value
	var freq [256]float64
	for _, b := range data {
		freq[b]++
	}
	size := float64(len(data))
	var entropy float64
	for i := 0; i < 256; i++ {
		if freq[i] > 0 {
			p := freq[i] / size
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}
