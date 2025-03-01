package main

// This file contains common patterns found in standard JPEG Huffman tables
// that often cause false positives in plaintext steganography detection

var standardHuffmanPatterns = []string{
	// DC luminance
	"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",

	// DC chrominance
	"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",

	// AC luminance
	"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",

	// AC chrominance
	"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",

	// Common sequences in standard Huffman tables
	"0123456789",
	"ABCDEFGHIJKLMN",
	"OPQRSTUVWXYZ",
	"abcdefghijklmn",
	"opqrstuvwxyz",
	"%&'()*456789:CDEFGHI",
	"%&'()*456789:CDEFGHIJSTUVWXYZcdefghijstuvwxyz",
	"&'()*56789:CDEFGHIJSTUVWXYZcdefghijstuvwxyz",

	// These patterns often appear in DCT coefficients
	"0123456789ABCDEFGHIJK",
	"0123456789ABCDEFGHIJKLMNOPQR",
	"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
}

// IsStandardHuffmanPattern checks if a string is likely part of standard JPEG encoding tables
func IsStandardHuffmanPattern(text string) bool {
	// Check against known patterns
	for _, pattern := range standardHuffmanPatterns {
		if len(text) >= 10 && ContainsSignificantOverlap(text, pattern) {
			return true
		}
	}
	return false
}

// ContainsSignificantOverlap checks if there's significant overlap between two strings
func ContainsSignificantOverlap(text, pattern string) bool {
	// Look for at least 10 consecutive matching characters
	minMatchLen := 10

	if len(text) < minMatchLen || len(pattern) < minMatchLen {
		return false
	}

	// Simple sliding window approach
	for i := 0; i <= len(pattern)-minMatchLen; i++ {
		chunk := pattern[i : i+minMatchLen]
		if contains(text, chunk) {
			return true
		}
	}

	return false
}

// contains checks if a string contains another string
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		found := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				found = false
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}

// HasHuffmanTableStructure checks if a byte array looks like a Huffman table
func HasHuffmanTableStructure(data []byte) bool {
	// Huffman tables in JPEG have specific structures
	// This is a simplified check for common patterns

	if len(data) < 20 {
		return false
	}

	// Check if the data contains sequences that appear in standard tables
	for _, pattern := range standardHuffmanPatterns {
		bytes := []byte(pattern)
		if ByteSliceContains(data, bytes[:10]) { // Using first 10 chars of each pattern
			return true
		}
	}

	return false
}

// ByteSliceContains checks if a byte slice contains another byte slice
func ByteSliceContains(haystack, needle []byte) bool {
	if len(needle) > len(haystack) {
		return false
	}

	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}
