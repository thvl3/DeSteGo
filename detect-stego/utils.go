package main

import (
	"bytes"
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

// IsTerminal attempts to determine if we're running in an interactive terminal
// This is useful for deciding whether to show live progress updates
func IsTerminal() bool {
	// This is a simplified check - a more robust implementation would use
	// a library like github.com/mattn/go-isatty to check properly
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// IsSuspiciousLSBData checks data that was extracted to see if it might be valid steganographic content
func IsSuspiciousLSBData(data []byte) bool {
	// Too small to be interesting
	if len(data) < 5 {
		return false
	}

	// Check if it looks like ASCII text
	if IsASCIIPrintable(data) {
		return true
	}

	// Check entropy - encrypted/compressed data typically has high entropy
	entropy := ComputeEntropy(data)

	// High entropy suggests encrypted/compressed data
	if entropy > 7.5 {
		return true
	}

	// Check for some common file signatures at the start
	fileSignatures := map[string][]byte{
		"PNG": {0x89, 0x50, 0x4E, 0x47},
		"JPG": {0xFF, 0xD8, 0xFF},
		"GIF": {0x47, 0x49, 0x46, 0x38},
		"BMP": {0x42, 0x4D},
		"PDF": {0x25, 0x50, 0x44, 0x46},
		"ZIP": {0x50, 0x4B, 0x03, 0x04},
		"RAR": {0x52, 0x61, 0x72, 0x21},
		"7Z":  {0x37, 0x7A, 0xBC, 0xAF},
		"EXE": {0x4D, 0x5A},
		"ELF": {0x7F, 0x45, 0x4C, 0x46},
	}

	for _, signature := range fileSignatures {
		if len(data) >= len(signature) && bytes.Equal(data[:len(signature)], signature) {
			return true
		}
	}

	return false
}
