package main

import (
	"image"
	"math"
)

// LSBDistribution represents the statistical distribution of LSB values
type LSBDistribution struct {
	Entropy      float64
	Samples      int
	ChannelStats map[string]ChannelStatistics
	Uniformity   float64 // 0.0 = not uniform, 1.0 = perfectly uniform
	PatternScore float64 // Higher values indicate suspicious patterns
	AnomalyScore float64 // Overall anomaly score
}

// ChannelStatistics holds statistical information about a specific channel's LSBs
type ChannelStatistics struct {
	Entropy        float64 // Shannon entropy of LSB distribution
	Samples        int     // Number of samples analyzed
	Transitions    float64 // Rate of 0->1 and 1->0 transitions
	FirstOrderBias float64 // First-order bias score (correlation with adjacent pixels)
}

// AnalyzeLSBStatistics performs advanced statistical analysis on the LSBs of an image
func AnalyzeLSBStatistics(img image.Image) (*LSBDistribution, error) {
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	totalPixels := width * height

	// Initialize distribution for tracking
	dist := &LSBDistribution{
		Samples:      totalPixels,
		ChannelStats: make(map[string]ChannelStatistics),
	}

	// Initialize counters for each channel
	rLSBs := make([]byte, totalPixels)
	gLSBs := make([]byte, totalPixels)
	bLSBs := make([]byte, totalPixels)

	// Extract all LSB values first to allow for complete analysis
	pixelIndex := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			rLSBs[pixelIndex] = byte(r & 1)
			gLSBs[pixelIndex] = byte(g & 1)
			bLSBs[pixelIndex] = byte(b & 1)
			pixelIndex++
		}
	}

	// Analyze each channel
	dist.ChannelStats["R"] = analyzeChannel(rLSBs, width)
	dist.ChannelStats["G"] = analyzeChannel(gLSBs, width)
	dist.ChannelStats["B"] = analyzeChannel(bLSBs, width)

	// Calculate the overall uniformity
	dist.Uniformity = (uniformityScore(rLSBs) +
		uniformityScore(gLSBs) +
		uniformityScore(bLSBs)) / 3.0

	// Calculate pattern scores to detect non-random arrangements
	dist.PatternScore = (calculatePatternScore(rLSBs, width) +
		calculatePatternScore(gLSBs, width) +
		calculatePatternScore(bLSBs, width)) / 3.0

	// Calculate overall entropy as the average of channel entropies
	dist.Entropy = (dist.ChannelStats["R"].Entropy +
		dist.ChannelStats["G"].Entropy +
		dist.ChannelStats["B"].Entropy) / 3.0

	// Calculate an overall anomaly score that combines multiple metrics
	// In statistical analysis, we want to detect deviations from expected natural LSB distributions
	rBias := math.Abs(0.5-dist.ChannelStats["R"].FirstOrderBias) * 2.0 // Scale to 0-1
	gBias := math.Abs(0.5-dist.ChannelStats["G"].FirstOrderBias) * 2.0
	bBias := math.Abs(0.5-dist.ChannelStats["B"].FirstOrderBias) * 2.0
	biasScore := (rBias + gBias + bBias) / 3.0

	// Final anomaly score is a weighted combination of our metrics
	// Higher is more suspicious
	dist.AnomalyScore = 0.3*dist.Uniformity + 0.4*dist.PatternScore + 0.3*biasScore

	return dist, nil
}

// analyzeChannel performs statistical analysis on a single channel's LSB values
func analyzeChannel(lsbValues []byte, width int) ChannelStatistics {
	stats := ChannelStatistics{
		Samples: len(lsbValues),
	}

	// Count occurrences
	ones := 0
	transitions := 0
	pairs00 := 0
	pairs01 := 0
	pairs10 := 0
	pairs11 := 0

	// Count ones and transitions
	for i := 0; i < len(lsbValues); i++ {
		if lsbValues[i] == 1 {
			ones++
		}
		if i > 0 && lsbValues[i] != lsbValues[i-1] {
			transitions++
		}

		// Count pairs for assessing first-order dependencies
		if i > 0 {
			pair := lsbValues[i-1]<<1 | lsbValues[i]
			switch pair {
			case 0: // 00
				pairs00++
			case 1: // 01
				pairs01++
			case 2: // 10
				pairs10++
			case 3: // 11
				pairs11++
			}
		}
	}

	// Calculate transition rate (0->1 and 1->0)
	// Natural images tend to have a specific transition distribution
	// Steganographic content often alters this
	stats.Transitions = float64(transitions) / float64(len(lsbValues)-1)

	// Calculate average first-order bias
	// In natural images, there's often correlation between adjacent pixels
	// Steganography tends to disturb this correlation
	totalPairs := pairs00 + pairs01 + pairs10 + pairs11
	if totalPairs > 0 {
		consistentPairs := pairs00 + pairs11 // Same value maintained
		stats.FirstOrderBias = float64(consistentPairs) / float64(totalPairs)
	} else {
		stats.FirstOrderBias = 0.5 // Neutral
	}

	// Calculate entropy (uses probability of 0s vs 1s)
	p1 := float64(ones) / float64(len(lsbValues))
	p0 := 1.0 - p1

	// Calculate Shannon entropy
	if p0 > 0 && p1 > 0 {
		stats.Entropy = -p0*math.Log2(p0) - p1*math.Log2(p1)
	}

	// This yields a value between 0 and 1, where:
	//  0.0 = All 0s or all 1s (lowest entropy, very suspicious)
	//  1.0 = Equal numbers of 0s and 1s (highest entropy, suspicious if too uniform)

	return stats
}

// uniformityScore measures how evenly distributed the LSB values are
// Returns a score from 0.0 to 1.0, where 1.0 means perfectly uniform
func uniformityScore(lsbValues []byte) float64 {
	ones := 0
	for _, v := range lsbValues {
		if v == 1 {
			ones++
		}
	}

	// Calculate how close the distribution is to 50/50
	expected := float64(len(lsbValues)) * 0.5
	observed := float64(ones)
	deviation := math.Abs(observed-expected) / expected

	// Convert to a score where 1.0 is perfectly uniform (50/50 split)
	// and 0.0 is completely skewed (all 0s or all 1s)
	return 1.0 - math.Min(deviation*2.0, 1.0)
}

// Fix the out of bounds error in the calculatePatternScore function
func calculatePatternScore(lsbValues []byte, width int) float64 {
	// Guard against invalid inputs more aggressively
	if len(lsbValues) <= 3 || width <= 3 {
		return 0
	}

	// Make sure we have enough data for actual analysis
	if len(lsbValues) < width*4 {
		return 0 // Need at least 4 rows for meaningful vertical patterns
	}

	// Look for horizontal patterns
	horizontalPatternScore := 0.0
	for y := 0; y < len(lsbValues)/width; y++ {
		repeats := 0
		for x := 3; x < width; x++ { // Start from 3 instead of 2 to ensure valid indices
			idx := y*width + x
			if idx >= len(lsbValues) || idx-3 < 0 {
				continue // Skip if indices would be out of range
			}

			// Look for repeating patterns of length 2 with bounds checking
			if lsbValues[idx] == lsbValues[idx-2] &&
				lsbValues[idx-1] == lsbValues[idx-3] {
				repeats++
			}
		}

		// Avoid division by zero
		if width > 3 {
			horizontalPatternScore += float64(repeats) / float64(width-3)
		}
	}

	// Avoid division by zero
	if len(lsbValues)/width > 0 {
		horizontalPatternScore /= float64(len(lsbValues) / width)
	}

	// Look for vertical patterns
	verticalPatternScore := 0.0
	height := len(lsbValues) / width
	if height <= 3 {
		return horizontalPatternScore // Not enough rows for vertical analysis
	}

	for x := 0; x < width; x++ {
		repeats := 0
		for y := 3; y < height; y++ { // Start from 3 instead of 2
			idx := y*width + x
			idxMinus2 := (y-2)*width + x
			idxMinus1 := (y-1)*width + x
			idxMinus3 := (y-3)*width + x

			// Add bounds checking
			if idx >= len(lsbValues) || idxMinus3 < 0 {
				continue
			}

			// Look for repeating patterns of length 2
			if lsbValues[idx] == lsbValues[idxMinus2] &&
				lsbValues[idxMinus1] == lsbValues[idxMinus3] {
				repeats++
			}
		}

		// Avoid division by zero
		if height > 3 {
			verticalPatternScore += float64(repeats) / float64(height-3)
		}
	}

	// Avoid division by zero
	if width > 0 {
		verticalPatternScore /= float64(width)
	}

	// Combine scores, giving higher weight to the pattern dimension with more repetitions
	return math.Max(horizontalPatternScore, verticalPatternScore)
}

// DetectSteganoAnomaly combines statistical measures to determine if an image likely contains steganography
// Returns an anomaly score and detailed analysis
func DetectSteganoAnomaly(img image.Image) (float64, *LSBDistribution, error) {
	// Perform the statistical analysis
	dist, err := AnalyzeLSBStatistics(img)
	if err != nil {
		return 0, nil, err
	}

	// Return the anomaly score and detailed distribution information
	return dist.AnomalyScore, dist, nil
}
