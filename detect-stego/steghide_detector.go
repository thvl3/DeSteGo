package main

import (
	"bytes"
	"fmt"

	//"io"
	"os"
	"regexp"
)

// StegHideSignatures contains patterns that might indicate Steghide usage
var StegHideSignatures = [][]byte{
	// Steghide often starts embedded content with these headers
	[]byte("sthv"),                 // Steghide version indicator
	[]byte{0x73, 0x74, 0x68, 0x76}, // "sthv" in hex
	[]byte("Steghide"),             // Text from a header or container
}

// StegHideStatistics contains statistical information about a potential Steghide payload
type StegHideStatistics struct {
	PotentialHeader      bool
	ModifiedCoefficients float64 // Percentage of coefficients that look modified
	EvenOddRatio         float64 // Ratio of even to odd coefficient values
	AbnormalDistribution bool    // If coefficient distribution looks abnormal
	ConfidenceScore      int     // 0-10 confidence score
}

// DetectStegHide analyzes a JPEG file for signs of Steghide modification
// Returns a confidence level and detailed statistics
func DetectStegHide(filename string) (bool, StegHideStatistics, error) {
	stats := StegHideStatistics{
		PotentialHeader:      false,
		ModifiedCoefficients: 0,
		EvenOddRatio:         0,
		AbnormalDistribution: false,
		ConfidenceScore:      0,
	}

	// Check for Steghide signatures
	found, err := checkForStegHideSignatures(filename)
	if err != nil {
		return false, stats, err
	}
	stats.PotentialHeader = found

	// Extract JPEG metadata for analysis
	metadata, err := ExtractJPEGMetadata(filename)
	if err != nil {
		return false, stats, err
	}

	// Analyze coefficient distributions
	// Steghide modifies DCT coefficients in a specific way
	coeffStats, err := analyzeCoefficients(filename)
	if err != nil {
		return stats.PotentialHeader, stats, nil // Still return header detection result
	}

	stats.ModifiedCoefficients = coeffStats.modifiedPercentage
	stats.EvenOddRatio = coeffStats.evenOddRatio
	stats.AbnormalDistribution = coeffStats.abnormalDistribution

	// Analyze entropy of different JPEG blocks
	entropyAbnormal := checkEntropyDistribution(filename)

	// Check comment field - Steghide sometimes leaves data in the comment
	commentSuspicious := false
	for _, comment := range metadata.Comments {
		if isSuspiciousComment(comment) {
			commentSuspicious = true
			break
		}
	}

	// Calculate overall confidence score
	confidence := 0

	if stats.PotentialHeader {
		confidence += 3 // Strong indicator
	}

	if stats.ModifiedCoefficients > 0.15 {
		confidence += 2
	}

	if stats.AbnormalDistribution {
		confidence += 2
	}

	// Even/odd ratio close to 1.0 with low variance can indicate Steghide
	if stats.EvenOddRatio > 0.9 && stats.EvenOddRatio < 1.1 {
		confidence += 1
	}

	if entropyAbnormal {
		confidence += 1
	}

	if commentSuspicious {
		confidence += 1
	}

	// Check for other indicators like file size
	if fileHasStegHideSizeCharacteristics(filename) {
		confidence += 1
	}

	stats.ConfidenceScore = confidence

	// Determine if Steghide is detected
	isDetected := confidence >= 4 // Threshold for positive detection
	return isDetected, stats, nil
}

// checkForStegHideSignatures searches for Steghide signatures in the file
func checkForStegHideSignatures(filename string) (bool, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return false, err
	}

	for _, signature := range StegHideSignatures {
		if bytes.Contains(data, signature) {
			return true, nil
		}
	}

	// Check for patterns in the DCT coefficient area
	if containsPatternInDCTArea(data) {
		return true, nil
	}

	return false, nil
}

// containsPatternInDCTArea checks for Steghide patterns in the DCT coefficient area
func containsPatternInDCTArea(data []byte) bool {
	// Find SOS marker (Start of Scan)
	var sosPos int = -1
	for i := 0; i < len(data)-2; i++ {
		if data[i] == 0xFF && data[i+1] == 0xDA {
			sosPos = i
			break
		}
	}

	if sosPos == -1 || sosPos+10 >= len(data) {
		return false
	}

	// Check the area after SOS for Steghide patterns
	// This is a simplified heuristic - real implementation would be more complex
	dctArea := data[sosPos+10:]

	// Check for suspicious patterns in coefficient area
	// These are simplified patterns that might indicate Steghide usage
	patterns := [][]byte{
		{0x00, 0x01, 0xFF, 0x00}, // Example pattern
		{0xFF, 0x00, 0x01, 0xFF}, // Example pattern
	}

	for _, pattern := range patterns {
		if bytes.Contains(dctArea, pattern) {
			return true
		}
	}

	return false
}

// coefficientStatistics holds statistics about DCT coefficients
type coefficientStatistics struct {
	modifiedPercentage   float64
	evenOddRatio         float64
	abnormalDistribution bool
}

// analyzeCoefficients analyzes DCT coefficients for signs of Steghide
// This is a simplified heuristic - real implementation would parse the actual DCT coefficients
func analyzeCoefficients(filename string) (coefficientStatistics, error) {
	result := coefficientStatistics{
		modifiedPercentage:   0,
		evenOddRatio:         1.0, // Default to neutral
		abnormalDistribution: false,
	}

	// Extract JPEG metadata
	metadata, err := ExtractJPEGMetadata(filename)
	if err != nil {
		return result, err
	}

	// Analyze quantization tables for Steghide characteristics
	if len(metadata.QuantizationTables) > 0 {
		var evenCount, oddCount int

		// For each table, check coefficient patterns
		for _, table := range metadata.QuantizationTables {
			// Skip table header byte
			if len(table) < 2 {
				continue
			}

			coefficients := table[1:]
			totalCoeffs := len(coefficients)

			// Count even/odd coefficients
			for _, coeff := range coefficients {
				if coeff%2 == 0 {
					evenCount++
				} else {
					oddCount++
				}
			}

			// In Steghide, certain coefficients get modified to store data
			// This often creates an abnormal statistical pattern

			// Count coefficient values that could indicate modification
			suspiciousValues := 0
			for i := 0; i < totalCoeffs; i++ {
				// Check for coefficient patterns typical of Steghide
				// This is a simplified check - real detection would be more thorough
				if i > 0 && i < totalCoeffs-1 {
					current := int(coefficients[i])
					prev := int(coefficients[i-1])
					next := int(coefficients[i+1])

					// Look for anomalies in coefficient patterns
					if (current == prev+1 || current == prev-1) &&
						(current == next+1 || current == next-1) {
						suspiciousValues++
					}
				}
			}

			if totalCoeffs > 0 {
				// Calculate percentage of coefficients that look modified
				result.modifiedPercentage = float64(suspiciousValues) / float64(totalCoeffs)
			}
		}

		// Calculate even/odd ratio
		totalCoeffs := evenCount + oddCount
		if totalCoeffs > 0 {
			result.evenOddRatio = float64(evenCount) / float64(totalCoeffs)
			// Natural images tend to have a less uniform even/odd ratio
			// Steghide often makes this ratio closer to 0.5
			if result.evenOddRatio > 0.45 && result.evenOddRatio < 0.55 {
				result.abnormalDistribution = true
			}
		}
	}

	return result, nil
}

// checkEntropyDistribution examines entropy distribution in the file
func checkEntropyDistribution(filename string) bool {
	// For real implementation, this would analyze entropy distribution
	// within JPEG segments for Steghide's characteristic patterns

	// Read the file in chunks
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	// For full implementation, we'd calculate entropy of different blocks
	// and look for Steghide's characteristic entropy changes

	// Since this is a simplified implementation, we'll return false
	// A complete implementation would use more sophisticated analysis
	return false
}

// isSuspiciousComment checks if a JPEG comment might contain Steghide data
func isSuspiciousComment(comment string) bool {
	// Check for binary or encoded data in comments
	binaryPattern := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F-\xFF]{4,}`)
	if binaryPattern.MatchString(comment) {
		return true
	}

	// Check for base64-like patterns, which could be encrypted data
	base64Pattern := regexp.MustCompile(`^[A-Za-z0-9+/=]{16,}$`)
	if base64Pattern.MatchString(comment) {
		return true
	}

	// Check for unusually long comments
	if len(comment) > 1000 {
		return true
	}

	return false
}

// fileHasStegHideSizeCharacteristics checks if file size matches Steghide patterns
func fileHasStegHideSizeCharacteristics(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}

	size := info.Size()

	// Steghide often produces files with sizes that are multiples of certain values
	// or have certain characteristics

	// This is a simplified check - real implementation would be more thorough
	// Check if file size is a multiple of 16 bytes (common for encrypted data)
	if size%16 == 0 && size > 1024 {
		return true
	}

	return false
}

// ExtractPotentialStegHidePayload attempts to extract the embedded payload
// using some common Steghide extraction heuristics
func ExtractPotentialStegHidePayload(filename string) ([]byte, error) {
	// Real extraction would require password and proper implementation
	// This is a placeholder that can be expanded if needed

	// For now, just check if the file contains potential payload areas
	// and return information about them

	isDetected, stats, err := DetectStegHide(filename)
	if err != nil {
		return nil, err
	}

	if !isDetected {
		return nil, fmt.Errorf("no Steghide payload detected")
	}

	// Create a report about the potential payload
	report := fmt.Sprintf(
		"Potential Steghide payload detected (confidence: %d/10)\n"+
			"Modified coefficients: %.1f%%\n"+
			"Suspicious header present: %v\n"+
			"For extraction, use the 'steghide' tool with: steghide extract -sf %s -p PASSWORD",
		stats.ConfidenceScore,
		stats.ModifiedCoefficients*100,
		stats.PotentialHeader,
		filename)

	return []byte(report), nil
}
