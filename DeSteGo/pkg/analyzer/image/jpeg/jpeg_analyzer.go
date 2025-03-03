package jpeg

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"

	"DeSteGo/pkg/analyzer"
	"DeSteGo/pkg/models"
)

// JPEGAnalyzer implements analysis for JPEG images
type JPEGAnalyzer struct {
	analyzer.BaseAnalyzer
}

// NewJPEGAnalyzer creates a new JPEG analyzer
func NewJPEGAnalyzer() *JPEGAnalyzer {
	return &JPEGAnalyzer{
		BaseAnalyzer: analyzer.NewBaseAnalyzer(
			"JPEG Analyzer",
			"Analyzes JPEG images for steganography",
			[]string{"jpeg", "jpg"},
		),
	}
}

// Analyze performs analysis on a JPEG file
func (a *JPEGAnalyzer) Analyze(filePath string, options analyzer.AnalysisOptions) (*models.AnalysisResult, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Decode the JPEG image
	img, err := jpeg.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG: %w", err)
	}

	// Create result object
	result := &models.AnalysisResult{
		FileType:        "jpeg",
		Filename:        filePath,
		Findings:        []models.Finding{},
		Recommendations: []string{},
	}

	// Check for appended data (reopen the file to check for appended data)
	file.Seek(0, 0)
	hasAppendedData, appendedSize, err := checkForAppendedData(file)
	if err != nil {
		return nil, fmt.Errorf("failed to check for appended data: %w", err)
	}

	if hasAppendedData {
		result.AddFinding("Found appended data after EOF", 0.8,
			fmt.Sprintf("Found %d bytes of appended data", appendedSize))
		result.DetectionScore = 0.7
		result.Confidence = 0.8
		result.Recommendations = append(result.Recommendations,
			"Extract and analyze the appended data after JPEG EOF marker")
	}

	// Run image-based analysis (common for all image types)
	imgResult, err := a.AnalyzeImage(img, options)
	if err != nil {
		return nil, fmt.Errorf("image analysis failed: %w", err)
	}

	// Merge results
	if imgResult != nil {
		for _, finding := range imgResult.Findings {
			result.AddFinding(finding.Description, finding.Confidence, finding.Details)
		}

		// Take the higher detection score
		if imgResult.DetectionScore > result.DetectionScore {
			result.DetectionScore = imgResult.DetectionScore
		}

		// Add image recommendations
		result.Recommendations = append(result.Recommendations, imgResult.Recommendations...)

		// Set possible algorithm if not already set
		if result.PossibleAlgorithm == "" {
			result.PossibleAlgorithm = imgResult.PossibleAlgorithm
		}
	}

	return result, nil
}

// AnalyzeImage analyzes a decoded JPEG image
func (a *JPEGAnalyzer) AnalyzeImage(img image.Image, options analyzer.AnalysisOptions) (*models.AnalysisResult, error) {
	// Create a basic result structure
	result := &models.AnalysisResult{
		FileType:        "jpeg",
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

	// Perform simple pixel analysis (in a real implementation, this would be more sophisticated)
	result.DetectionScore = 0.1 // Default low score
	result.Confidence = 0.5     // Medium confidence

	// Add general recommendations for JPEG
	result.Recommendations = append(result.Recommendations,
		"Use specialized JPEG steganalysis tools for deeper analysis")

	return result, nil
}

// checkForAppendedData looks for data after the JPEG EOF marker
func checkForAppendedData(file *os.File) (bool, int64, error) {
	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return false, 0, err
	}
	fileSize := fileInfo.Size()

	// Buffer for reading
	buffer := make([]byte, 2)

	// JPEG files end with the EOI marker: 0xFF 0xD9
	// Start from the end and search backwards for the EOI marker
	for pos := fileSize - 2; pos >= 0; pos -= 1 {
		_, err = file.Seek(pos, 0)
		if err != nil {
			return false, 0, err
		}

		_, err = file.Read(buffer)
		if err != nil {
			return false, 0, err
		}

		// Check if we found the EOI marker
		if buffer[0] == 0xFF && buffer[1] == 0xD9 {
			// If the marker is not at the end, we have appended data
			if pos+2 < fileSize {
				appendedSize := fileSize - (pos + 2)
				return true, appendedSize, nil
			}
			return false, 0, nil
		}
	}

	// If we reach here, we didn't find an EOI marker
	return false, 0, fmt.Errorf("invalid JPEG: no EOI marker found")
}
