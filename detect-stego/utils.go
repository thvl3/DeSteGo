package main

import (
	"image"
	"image/png"
	"io"
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
// This is a common heuristic to determine if hidden data might be text.
func IsASCIIPrintable(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	printableCount := 0
	for _, b := range data {
		// "printable" range: [32..126], plus newline, carriage return, tab
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
			printableCount++
		}
	}
	ratio := float64(printableCount) / float64(len(data))
	return ratio > 0.8
}
