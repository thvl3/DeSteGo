package main

import (
	"image"
	"math"
	"strings"
)

// LSBDistribution holds statistics about the distribution of LSB bits in an image
type LSBDistribution struct {
	TotalPixels  int
	ZeroBits     int
	OneBits      int
	ZeroPercent  float64
	OnePercent   float64
	Entropy      float64
	ChannelStats map[string]ChannelLSBStats
}

// ChannelLSBStats holds LSB statistics for a specific channel
type ChannelLSBStats struct {
	ZeroBits    int
	OneBits     int
	ZeroPercent float64
	OnePercent  float64
	Entropy     float64
}

// DetectJSLSB tries to determine if the image contains data hidden using
// the JavaScript LSB algorithm from the provided code, which:
// - Uses 1 bit per RGB channel (3 bits per pixel)
// - Doesn't use the Alpha channel
// - Stores data from left to right, top to bottom
func DetectJSLSB(img image.Image) bool {
	dist := AnalyzeLSBDistribution(img)

	// Analyze the distribution to find signs of the JS LSB algorithm
	// The JavaScript algorithm has these characteristics:
	// 1. It normalizes images by making all pixels even before encoding
	// 2. It only uses R,G,B channels (not Alpha)

	// Check if R,G,B channels have similar distributions
	// This would indicate data spread across all channels
	rgbEntropy := (dist.ChannelStats["R"].Entropy +
		dist.ChannelStats["G"].Entropy +
		dist.ChannelStats["B"].Entropy) / 3

	// If RGB entropy is in a reasonable range for encoded data
	// and Alpha channel has little entropy (or is closer to natural distribution)
	// The JS algorithm doesn't use Alpha channel
	if rgbEntropy > 0.7 && rgbEntropy < 1.0 {
		// Calculate the difference between RGB and Alpha entropy
		// In JS LSB, Alpha should be unchanged while RGB channels are modified
		if alphaStat, ok := dist.ChannelStats["A"]; ok && alphaStat.Entropy < 0.3 {
			return true
		}
	}

	return false
}

// AnalyzeLSBDistribution analyzes the distribution of LSB values across channels
func AnalyzeLSBDistribution(img image.Image) LSBDistribution {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Temporary counters for channel statistics
	rZeros, rOnes := 0, 0
	gZeros, gOnes := 0, 0
	bZeros, bOnes := 0, 0
	aZeros, aOnes := 0, 0

	result := LSBDistribution{
		TotalPixels: width * height,
		ChannelStats: map[string]ChannelLSBStats{
			"R": {},
			"G": {},
			"B": {},
			"A": {},
		},
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			// Extract LSBs from each channel (taking into account 16-bit color values from RGBA())
			rLSB := uint8(r>>8) & 1
			gLSB := uint8(g>>8) & 1
			bLSB := uint8(b>>8) & 1
			aLSB := uint8(a>>8) & 1

			// Update overall statistics
			if rLSB == 0 {
				result.ZeroBits++
				rZeros++
			} else {
				result.OneBits++
				rOnes++
			}

			if gLSB == 0 {
				result.ZeroBits++
				gZeros++
			} else {
				result.OneBits++
				gOnes++
			}

			if bLSB == 0 {
				result.ZeroBits++
				bZeros++
			} else {
				result.OneBits++
				bOnes++
			}

			if aLSB == 0 {
				result.ZeroBits++
				aZeros++
			} else {
				result.OneBits++
				aOnes++
			}
		}
	}

	// Calculate percentages and entropy
	totalBits := result.ZeroBits + result.OneBits
	if totalBits > 0 {
		result.ZeroPercent = float64(result.ZeroBits) / float64(totalBits)
		result.OnePercent = float64(result.OneBits) / float64(totalBits)

		// Calculate entropy using Shannon entropy formula
		if result.ZeroPercent > 0 && result.OnePercent > 0 {
			result.Entropy = -result.ZeroPercent*math.Log2(result.ZeroPercent) -
				result.OnePercent*math.Log2(result.OnePercent)
		}
	}

	// Update channel statistics in the map
	rStats := result.ChannelStats["R"]
	rStats.ZeroBits = rZeros
	rStats.OneBits = rOnes
	result.ChannelStats["R"] = rStats

	gStats := result.ChannelStats["G"]
	gStats.ZeroBits = gZeros
	gStats.OneBits = gOnes
	result.ChannelStats["G"] = gStats

	bStats := result.ChannelStats["B"]
	bStats.ZeroBits = bZeros
	bStats.OneBits = bOnes
	result.ChannelStats["B"] = bStats

	aStats := result.ChannelStats["A"]
	aStats.ZeroBits = aZeros
	aStats.OneBits = aOnes
	result.ChannelStats["A"] = aStats

	// Calculate per-channel statistics
	for channel, stats := range result.ChannelStats {
		total := stats.ZeroBits + stats.OneBits
		if total > 0 {
			stats.ZeroPercent = float64(stats.ZeroBits) / float64(total)
			stats.OnePercent = float64(stats.OneBits) / float64(total)

			// Calculate per-channel entropy
			if stats.ZeroPercent > 0 && stats.OnePercent > 0 {
				stats.Entropy = -stats.ZeroPercent*math.Log2(stats.ZeroPercent) -
					stats.OnePercent*math.Log2(stats.OnePercent)
			}

			// Update the map with modified stats
			result.ChannelStats[channel] = stats
		}
	}

	return result
}

// IsLikelyASCII checks if the data looks like ASCII text
func IsLikelyASCII(data []byte) bool {
	if len(data) < 4 { // Too short to analyze
		return false
	}

	// Check if the data contains reasonable ASCII characters
	// Focus on common printable ASCII range
	printableCount := 0

	for _, b := range data {
		if (b >= 32 && b <= 126) || // Printable ASCII
			b == 9 || b == 10 || b == 13 { // Tab, LF, CR
			printableCount++
		}
	}

	// If more than 75% is printable ASCII, it's likely a text message
	return float64(printableCount)/float64(len(data)) > 0.75
}

// GetLSBMask returns a ChannelMask suitable for the JavaScript LSB implementation
func GetLSBMask() ChannelMask {
	return ChannelMask{
		RBits: 1,
		GBits: 1,
		BBits: 1,
		ABits: 0, // JavaScript algorithm doesn't use Alpha channel
	}
}

// TryExtractJSLSB attempts to extract data using the JavaScript LSB algorithm
func TryExtractJSLSB(img image.Image) (string, error) {
	mask := GetLSBMask()

	// For the JS implementation, we know it's using LSB first order
	data, err := ExtractData(img, mask, LSBFirst)
	if err != nil {
		return "", err
	}

	// Try an alternate approach if standard extraction fails
	if !IsLikelyASCII(data) {
		// The JS implementation may not include length prefix
		// so try direct bit extraction
		result := extractBitsDirectly(img, mask)
		if len(result) > 0 && IsLikelyASCII([]byte(result)) {
			return result, nil
		}
	}

	// Convert binary data to text and clean it up
	result := string(data)

	// Clean up the string - remove unprintable characters at the end
	result = strings.TrimRight(result, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x0b\x0c\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f")

	return result, nil
}

// extractBitsDirectly extracts bits directly from the image without assuming a length prefix
// This is a fallback method for the JavaScript LSB extraction which may not use a length prefix
func extractBitsDirectly(img image.Image, mask ChannelMask) string {
	bounds := img.Bounds()
	// Remove unnecessary width and height declarations

	// Collect bits
	var bits []byte

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			// Extract LSB from each channel according to mask
			if mask.RBits > 0 {
				bits = append(bits, byte(r>>8)&1)
			}
			if mask.GBits > 0 {
				bits = append(bits, byte(g>>8)&1)
			}
			if mask.BBits > 0 {
				bits = append(bits, byte(b>>8)&1)
			}

			// We know JS implementation doesn't use Alpha channel
		}
	}

	// Convert bits to bytes (ASCII characters)
	var result strings.Builder
	for i := 0; i < len(bits)/8; i++ {
		var charByte byte
		for j := 0; j < 8; j++ {
			if i*8+j < len(bits) {
				charByte = (charByte << 1) | bits[i*8+j]
			}
		}

		// Stop at first null byte as it's likely the end of the message
		if charByte == 0 {
			break
		}

		result.WriteByte(charByte)
	}

	return result.String()
}
