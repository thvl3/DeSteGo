package lsb

import (
	"errors"
	"image"
	"math"
)

// AnalysisResult represents the result of LSB distribution analysis
type AnalysisResult struct {
	AnomalyScore float64
	Entropy      float64
	Confidence   float64
	ChannelStats map[string]float64
}

// AnalyzeDistribution analyzes the LSB distribution in an image across all color channels
func AnalyzeDistribution(img image.Image) (*AnalysisResult, error) {
	if img == nil {
		return nil, errors.New("nil image provided")
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	totalPixels := width * height

	// Initialize counters for each channel's LSB values
	rZeros, rOnes := 0, 0
	gZeros, gOnes := 0, 0
	bZeros, bOnes := 0, 0
	aZeros, aOnes := 0, 0

	// Analyze LSB distribution across all pixels
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			// Extract LSBs from each channel (16-bit color values from RGBA())
			// Using just the 8 most significant bits (>>8) to match standard 8-bit color depth
			// Then extracting just the least significant bit (&1)
			rLSB := uint8(r>>8) & 1
			gLSB := uint8(g>>8) & 1
			bLSB := uint8(b>>8) & 1
			aLSB := uint8(a>>8) & 1

			// Count occurrences of 0s and 1s for each channel
			if rLSB == 0 {
				rZeros++
			} else {
				rOnes++
			}

			if gLSB == 0 {
				gZeros++
			} else {
				gOnes++
			}

			if bLSB == 0 {
				bZeros++
			} else {
				bOnes++
			}

			if aLSB == 0 {
				aZeros++
			} else {
				aOnes++
			}
		}
	}

	// Calculate channel-specific statistics
	rZeroPercent := float64(rZeros) / float64(totalPixels)
	rOnePercent := float64(rOnes) / float64(totalPixels)
	gZeroPercent := float64(gZeros) / float64(totalPixels)
	gOnePercent := float64(gOnes) / float64(totalPixels)
	bZeroPercent := float64(bZeros) / float64(totalPixels)
	bOnePercent := float64(bOnes) / float64(totalPixels)
	aZeroPercent := float64(aZeros) / float64(totalPixels)
	aOnePercent := float64(aOnes) / float64(totalPixels)

	// Calculate Shannon entropy for each channel
	rEntropy := calculateEntropy(rZeroPercent, rOnePercent)
	gEntropy := calculateEntropy(gZeroPercent, gOnePercent)
	bEntropy := calculateEntropy(bZeroPercent, bOnePercent)
	aEntropy := calculateEntropy(aZeroPercent, aOnePercent)

	// Calculate average entropy across RGB channels
	avgEntropy := (rEntropy + gEntropy + bEntropy) / 3.0

	// Calculate anomaly score based on entropy and distribution patterns
	anomalyScore := calculateAnomalyScore(
		rEntropy, gEntropy, bEntropy, aEntropy,
		rZeroPercent, gZeroPercent, bZeroPercent, aZeroPercent,
	)

	// Calculate confidence based on sample size and entropy variance
	entropyVariance := calculateVariance([]float64{rEntropy, gEntropy, bEntropy, aEntropy})
	confidence := calculateConfidence(totalPixels, entropyVariance)

	return &AnalysisResult{
		AnomalyScore: anomalyScore,
		Entropy:      avgEntropy,
		Confidence:   confidence,
		ChannelStats: map[string]float64{
			"R":       rEntropy,
			"G":       gEntropy,
			"B":       bEntropy,
			"A":       aEntropy,
			"R_zeros": rZeroPercent,
			"G_zeros": gZeroPercent,
			"B_zeros": bZeroPercent,
			"A_zeros": aZeroPercent,
		},
	}, nil
}

// calculateEntropy calculates Shannon entropy from probability distribution
func calculateEntropy(zeroProb, oneProb float64) float64 {
	// Avoid log(0) errors
	if zeroProb <= 0 || oneProb <= 0 {
		return 0
	}

	// Shannon entropy formula: -sum(p_i * log2(p_i))
	return -zeroProb*math.Log2(zeroProb) - oneProb*math.Log2(oneProb)
}

// calculateAnomalyScore determines how likely the LSB distribution indicates steganography
func calculateAnomalyScore(rEntropy, gEntropy, bEntropy, aEntropy,
	rZeroPercent, gZeroPercent, bZeroPercent, aZeroPercent float64) float64 {

	score := 0.0

	// Perfect entropy (close to 1.0) is suspicious for steganography
	// Natural images rarely have perfect entropy in LSBs
	avgRGBEntropy := (rEntropy + gEntropy + bEntropy) / 3.0
	if avgRGBEntropy > 0.97 {
		score += 0.4 // High entropy is suspicious
	} else if avgRGBEntropy > 0.92 {
		score += 0.2 // Moderately high entropy
	}

	// Check for suspicious patterns across channels
	// Equal distributions across channels can indicate embedded data

	// Calculate deviation from 50/50 distribution for each channel
	rDeviation := math.Abs(rZeroPercent-0.5) * 2 // Normalized to [0,1]
	gDeviation := math.Abs(gZeroPercent-0.5) * 2
	bDeviation := math.Abs(bZeroPercent-0.5) * 2

	// Low deviation (close to 50/50 split) in all channels is suspicious
	avgDeviation := (rDeviation + gDeviation + bDeviation) / 3.0
	if avgDeviation < 0.05 {
		score += 0.3 // Very close to 50/50 is highly suspicious
	} else if avgDeviation < 0.1 {
		score += 0.2 // Moderately close to 50/50
	}

	// Check for similar entropy across RGB channels
	// Natural images typically have variation between channels
	entropyVariance := calculateVariance([]float64{rEntropy, gEntropy, bEntropy})
	if entropyVariance < 0.0001 {
		score += 0.3 // Almost identical entropy across channels is suspicious
	} else if entropyVariance < 0.001 {
		score += 0.15 // Low variance is somewhat suspicious
	}

	// Alpha channel should typically differ from RGB in natural images
	// If LSB in alpha matches RGB pattern, that's suspicious
	alphaDiff := math.Abs(aEntropy - avgRGBEntropy)
	if alphaDiff < 0.05 && aEntropy > 0.9 {
		score += 0.2 // Suspicious alpha channel pattern
	}

	// Ensure score is in [0,1] range
	if score > 1.0 {
		return 1.0
	}
	return score
}

// calculateVariance calculates statistical variance of a slice of values
func calculateVariance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate variance
	varSum := 0.0
	for _, v := range values {
		diff := v - mean
		varSum += diff * diff
	}

	return varSum / float64(len(values))
}

// calculateConfidence estimates confidence level based on sample size and variance
func calculateConfidence(sampleSize int, variance float64) float64 {
	// Larger samples give higher confidence
	sampleConfidence := math.Min(float64(sampleSize)/10000.0, 1.0)

	// Extreme variance (very high or very low) can increase confidence in detection
	varianceConfidence := 0.0
	if variance < 0.0001 { // Very low variance
		varianceConfidence = 0.9
	} else if variance < 0.001 {
		varianceConfidence = 0.7
	} else if variance < 0.01 {
		varianceConfidence = 0.5
	} else {
		varianceConfidence = 0.3
	}

	// Combine factors (weighted average)
	return 0.7*sampleConfidence + 0.3*varianceConfidence
}
