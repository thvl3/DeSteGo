package png

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"

	"DeSteGo/pkg/analyzer"
	"DeSteGo/pkg/analyzer/image/lsb"
	"DeSteGo/pkg/models"
)

/*
Summary of this file and these functions:
- This file contains the implementation of the PNGAnalyzer struct, which is an implementation of the ImageAnalyzer interface.
- The PNGAnalyzer struct provides methods for analyzing PNG images for steganography.
- The NewPNGAnalyzer function creates a new PNGAnalyzer instance.
- The Analyze method decodes a PNG image from a file and performs analysis on it. (It calls the PNGAnalyzer.AnalyzeImage method.)
- The AnalyzeImage method performs analysis on a decoded PNG image.
- The PNGAnalyzer uses the LSB analysis from the shared package to detect steganography in PNG images.
- The analysis results include findings based on LSB distribution and entropy, as well as recommendations for further analysis.
*/

// PNGAnalyzer implements analysis for PNG images
type PNGAnalyzer struct {
	analyzer.BaseAnalyzer
}

// NewPNGAnalyzer creates a new PNG analyzer
func NewPNGAnalyzer() *PNGAnalyzer {
	return &PNGAnalyzer{
		BaseAnalyzer: analyzer.NewBaseAnalyzer(
			"PNG Analyzer",
			"Analyzes PNG images for steganography",
			[]string{"png"},
		),
	}
}

// Analyze performs analysis on a PNG file
func (a *PNGAnalyzer) Analyze(filePath string, options analyzer.AnalysisOptions) (*models.AnalysisResult, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Decode the PNG image
	img, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}

	// Pass to image analyzer
	return a.AnalyzeImage(img, options)
}

// AnalyzeImage analyzes a decoded PNG image
func (a *PNGAnalyzer) AnalyzeImage(img image.Image, options analyzer.AnalysisOptions) (*models.AnalysisResult, error) {
	if img == nil {
		return nil, errors.New("nil image provided")
	}

	// Create a basic result structure
	result := &models.AnalysisResult{
		FileType:        "png",
		Findings:        []models.Finding{},
		Recommendations: []string{},
	}

	// Get image dimensions
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// Add basic image info
	result.Details = map[string]interface{}{
		"width":  width,
		"height": height,
	}

	// Run LSB analysis using the shared package
	lsbResult, err := lsb.AnalyzeDistribution(img)
	if err != nil {
		return nil, fmt.Errorf("LSB analysis failed: %w", err)
	}

	// Update result with LSB findings
	result.DetectionScore = lsbResult.AnomalyScore
	result.Confidence = lsbResult.Confidence

	// Add findings based on LSB analysis
	if lsbResult.AnomalyScore > 0.8 {
		result.AddFinding("Highly anomalous LSB distribution", 0.9,
			fmt.Sprintf("Statistical anomaly score=%.4f (>0.8 is suspicious)", lsbResult.AnomalyScore))
		result.PossibleAlgorithm = "LSB Steganography"

		result.Recommendations = append(result.Recommendations,
			"Extract LSB data using specialized tools",
			"Check for hidden text patterns in LSB data")
	} else if lsbResult.AnomalyScore > 0.5 {
		result.AddFinding("Unusual LSB distribution", 0.7,
			fmt.Sprintf("Statistical anomaly score=%.4f (>0.5 is unusual)", lsbResult.AnomalyScore))
		result.Recommendations = append(result.Recommendations,
			"Run further analysis with specialized tools")
	}

	// Add entropy-based findings
	if lsbResult.Entropy > 0.99 {
		result.AddFinding("Perfect LSB entropy", 0.9,
			fmt.Sprintf("LSB entropy=%.4f (unnaturally perfect randomness)", lsbResult.Entropy))
	} else if lsbResult.Entropy < 0.3 {
		result.AddFinding("Abnormally low LSB entropy", 0.8,
			fmt.Sprintf("LSB entropy=%.4f (unnaturally low randomness)", lsbResult.Entropy))
	}

	return result, nil
}
