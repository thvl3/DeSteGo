package main

import (
	"image"
	"math"
	"strings"
)

// FalsePositiveCheck performs additional checks to reduce false positives
type FalsePositiveCheck struct {
	// Track previous results for comparison
	PreviousFindings map[string]int

	// Methods to store knowledge about common false positive patterns
	KnownFalsePositives map[string]bool
}

// NewFalsePositiveCheck creates a new instance for false positive reduction
func NewFalsePositiveCheck() *FalsePositiveCheck {
	return &FalsePositiveCheck{
		PreviousFindings:    make(map[string]int),
		KnownFalsePositives: make(map[string]bool),
	}
}

// EvaluateDetection examines findings and returns true if likely a false positive
func (fp *FalsePositiveCheck) EvaluateDetection(results *ScanResults, img image.Image) bool {
	falsePositiveScore := 0.0
	totalChecks := 0.0

	// Check 1: Hidden text detection
	if results.HiddenTextFound {
		totalChecks++

		// Count how many detected texts look like metadata
		metadataLikeCount := 0
		for _, text := range results.DetectedTexts {
			if IsMetadataString(text) {
				metadataLikeCount++
			}
		}

		// If most texts are metadata-like, likely false positive
		if float64(metadataLikeCount)/float64(len(results.DetectedTexts)) > 0.7 {
			falsePositiveScore += 1.0
		}

		// If text is very short or common in normal images
		if len(results.DetectedTexts) > 0 {
			shortTextCount := 0
			for _, text := range results.DetectedTexts {
				if len(text) < 10 {
					shortTextCount++
				}
			}

			if float64(shortTextCount)/float64(len(results.DetectedTexts)) > 0.8 {
				falsePositiveScore += 0.5
			}
		}
	}

	// Check 2: LSB anomalies
	if results.LSBAnomaliesFound {
		totalChecks++

		// Image format checks (some formats naturally have high entropy)
		if strings.HasSuffix(strings.ToLower(results.Filename), ".png") && results.LSBEntropy > 0.99 {
			// PNG compression can cause high entropy in LSB plane
			falsePositiveScore += 0.5
		}

		// Image complexity check - highly detailed images naturally have high entropy
		if results.ImageComplexity > 0.85 {
			falsePositiveScore += 0.5
		}
	}

	// Calculate overall false positive likelihood
	falsePositiveLikelihood := 0.0
	if totalChecks > 0 {
		falsePositiveLikelihood = falsePositiveScore / totalChecks
	}

	// Update results with false positive assessment
	results.FalsePositiveLikelihood = falsePositiveLikelihood

	return falsePositiveLikelihood > 0.7 // Return true if likely a false positive
}

// CalculateImageComplexity returns a measure of image complexity (0-1)
func CalculateImageComplexity(img image.Image) float64 {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	if width == 0 || height == 0 {
		return 0.0
	}

	// Sample the image to measure local variance (a proxy for complexity)
	sampleSize := 16
	totalVariance := 0.0
	sampleCount := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleSize {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleSize {
			variance := calculateLocalVariance(img, x, y, sampleSize)
			totalVariance += variance
			sampleCount++
		}
	}

	if sampleCount == 0 {
		return 0.0
	}

	averageVariance := totalVariance / float64(sampleCount)

	// Normalize to 0-1 range (empirically, variance > 2000 is high complexity)
	return math.Min(1.0, averageVariance/2000.0)
}

// calculateLocalVariance computes intensity variance in a local region
func calculateLocalVariance(img image.Image, startX, startY, size int) float64 {
	bounds := img.Bounds()
	endX := min(startX+size, bounds.Max.X)
	endY := min(startY+size, bounds.Max.Y)

	// Calculate mean value
	sum := 0.0
	count := 0

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			intensity := (float64(r) + float64(g) + float64(b)) / 3.0
			sum += intensity
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	mean := sum / float64(count)

	// Calculate variance
	variance := 0.0
	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			intensity := (float64(r) + float64(g) + float64(b)) / 3.0
			diff := intensity - mean
			variance += diff * diff
		}
	}

	return variance / float64(count)
}
