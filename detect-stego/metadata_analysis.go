// filepath: /home/thule/Repositories/SteGOC2/detect-stego/metadata_analysis.go

package main

import (
	"encoding/hex"
	"regexp"
	"strings"
)

// IsMetadataString checks if a string is likely standard metadata rather than hidden text
func IsMetadataString(text string) bool {
	// Common metadata markers
	metadataPatterns := []string{
		"Exif", "xmp", "photoshop", "adobe", "ICC_PROFILE", "XMP",
		"JFIF", "ducky", "Created with", "Software:", "Artist:",
		"Make:", "Model:", "Copyright:", "GPS", "Date", "Time",
		"Resolution", "Color", "Profile", "Version", "Camera",
		"metadata", "Metadata", "Author", "Producer", "Creator",
		"Title", "Subject", "Keywords", "Description", "Comment",
	}

	for _, pattern := range metadataPatterns {
		if strings.Contains(strings.ToLower(text), strings.ToLower(pattern)) {
			return true
		}
	}

	// Check for patterns that look like metadata
	metadataRegexes := []*regexp.Regexp{
		regexp.MustCompile(`^\d{4}[-/]\d{2}[-/]\d{2}`),                        // Date pattern
		regexp.MustCompile(`^\d{2}:\d{2}:\d{2}`),                              // Time pattern
		regexp.MustCompile(`^[A-Za-z]+ \d{1,2}, \d{4}`),                       // Date in text format
		regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`), // Email
		regexp.MustCompile(`^https?://`),                                      // URL
		regexp.MustCompile(`^[A-Za-z]+ [A-Za-z]+$`),                           // Two words (likely name)
		regexp.MustCompile(`^v\d+\.\d+\.\d+$`),                                // Version number
	}

	for _, regex := range metadataRegexes {
		if regex.MatchString(strings.TrimSpace(text)) {
			return true
		}
	}

	// Check for hexadecimal data that might be a hash or UUID
	if len(text) >= 32 && isHexString(text) {
		return true
	}

	return false
}

// isHexString checks if a string is primarily hexadecimal
func isHexString(s string) bool {
	// Remove common separators
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, " ", "")

	// Check if it's valid hex
	_, err := hex.DecodeString(s)
	return err == nil
}

// FilterHiddenText takes detected text and filters out normal metadata
func FilterHiddenText(detectedTexts []string) ([]string, float64) {
	var suspiciousTexts []string
	totalTexts := len(detectedTexts)
	metadataCount := 0

	for _, text := range detectedTexts {
		// Skip if empty or very short (less than 3 chars)
		if len(strings.TrimSpace(text)) < 3 {
			metadataCount++
			continue
		}

		// Check if it looks like metadata
		if IsMetadataString(text) {
			metadataCount++
			continue
		}

		// If we get here, it might be hidden text
		suspiciousTexts = append(suspiciousTexts, text)
	}

	// Calculate confidence - if most strings are normal metadata, lower confidence
	confidence := 0.0
	if totalTexts > 0 {
		confidence = float64(len(suspiciousTexts)) / float64(totalTexts)

		// Adjust confidence based on text characteristics
		for _, text := range suspiciousTexts {
			// Increase confidence for longer texts (more likely to be deliberate)
			if len(text) > 20 {
				confidence += 0.1
			}

			// Increase confidence for structured text with punctuation
			if regexp.MustCompile(`[.!?]\s+[A-Z]`).MatchString(text) {
				confidence += 0.1
			}
		}

		// Cap confidence at 1.0
		if confidence > 1.0 {
			confidence = 1.0
		}
	}

	return suspiciousTexts, confidence
}
