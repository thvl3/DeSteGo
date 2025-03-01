package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strings"
)

// JPEG markers
const (
	markerPrefix = 0xFF
	markerSOI    = 0xD8 // Start Of Image
	markerAPP0   = 0xE0 // JFIF APP0 marker
	markerAPP1   = 0xE1 // EXIF APP1 marker
	markerDQT    = 0xDB // Define Quantization Table
	markerSOF0   = 0xC0 // Start Of Frame (baseline DCT)
	markerSOF2   = 0xC2 // Start Of Frame (progressive DCT)
	markerDHT    = 0xC4 // Define Huffman Table
	markerSOS    = 0xDA // Start Of Scan
	markerCOM    = 0xFE // Comment
	markerEOI    = 0xD9 // End Of Image
)

// JPEGMetadata holds information extracted from a JPEG file's structure
type JPEGMetadata struct {
	QuantizationTables [][]byte
	HuffmanTables      [][]byte
	Comments           []string
	Width              int
	Height             int
	Components         int
	HasAppendedData    bool
	AppendedDataSize   int
	IsProgressive      bool
	MarkerSequence     []byte
}

// ExtractJPEGMetadata analyzes a JPEG file and extracts its metadata
func ExtractJPEGMetadata(filename string) (*JPEGMetadata, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	// Check for JPEG signature (FF D8)
	b1, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	b2, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	if b1 != 0xFF || b2 != markerSOI {
		return nil, fmt.Errorf("not a valid JPEG file")
	}

	metadata := &JPEGMetadata{
		QuantizationTables: make([][]byte, 0),
		HuffmanTables:      make([][]byte, 0),
		Comments:           make([]string, 0),
		MarkerSequence:     []byte{b1, b2},
	}

	// Read and process JPEG segments
	for {
		// Every marker starts with 0xFF
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Look for marker prefix (0xFF)
		if b != 0xFF {
			// Skip to next possible marker
			continue
		}

		// Read marker identifier
		marker, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}

		metadata.MarkerSequence = append(metadata.MarkerSequence, 0xFF, marker)

		// Process marker
		switch marker {
		case markerEOI:
			// End of image, check for appended data
			remainingBytes, _ := io.ReadAll(reader)
			if len(remainingBytes) > 0 {
				metadata.HasAppendedData = true
				metadata.AppendedDataSize = len(remainingBytes)
			}
			return metadata, nil

		case markerDQT:
			// Quantization table
			lengthBytes := make([]byte, 2)
			if _, err := io.ReadFull(reader, lengthBytes); err != nil {
				return nil, err
			}
			length := int(lengthBytes[0])<<8 | int(lengthBytes[1])

			// Read the quantization table data
			tableData := make([]byte, length-2)
			if _, err := io.ReadFull(reader, tableData); err != nil {
				return nil, err
			}

			// Store the quantization table
			metadata.QuantizationTables = append(metadata.QuantizationTables, tableData)

		case markerSOF0:
			// Baseline DCT frame header
			metadata.IsProgressive = false
			lengthBytes := make([]byte, 2)
			if _, err := io.ReadFull(reader, lengthBytes); err != nil {
				return nil, err
			}
			length := int(lengthBytes[0])<<8 | int(lengthBytes[1])

			// Read frame header data
			frameData := make([]byte, length-2)
			if _, err := io.ReadFull(reader, frameData); err != nil {
				return nil, err
			}

			// Extract image dimensions and color components
			metadata.Height = int(frameData[1])<<8 | int(frameData[2])
			metadata.Width = int(frameData[3])<<8 | int(frameData[4])
			metadata.Components = int(frameData[5])

		case markerSOF2:
			// Progressive DCT frame header
			metadata.IsProgressive = true
			lengthBytes := make([]byte, 2)
			if _, err := io.ReadFull(reader, lengthBytes); err != nil {
				return nil, err
			}
			length := int(lengthBytes[0])<<8 | int(lengthBytes[1])

			// Read frame header data
			frameData := make([]byte, length-2)
			if _, err := io.ReadFull(reader, frameData); err != nil {
				return nil, err
			}

			// Extract image dimensions and color components
			metadata.Height = int(frameData[1])<<8 | int(frameData[2])
			metadata.Width = int(frameData[3])<<8 | int(frameData[4])
			metadata.Components = int(frameData[5])

		case markerDHT:
			// Huffman table
			lengthBytes := make([]byte, 2)
			if _, err := io.ReadFull(reader, lengthBytes); err != nil {
				return nil, err
			}
			length := int(lengthBytes[0])<<8 | int(lengthBytes[1])

			// Read the Huffman table data
			tableData := make([]byte, length-2)
			if _, err := io.ReadFull(reader, tableData); err != nil {
				return nil, err
			}

			// Store the Huffman table
			metadata.HuffmanTables = append(metadata.HuffmanTables, tableData)

		case markerCOM:
			// Comment segment
			lengthBytes := make([]byte, 2)
			if _, err := io.ReadFull(reader, lengthBytes); err != nil {
				return nil, err
			}
			length := int(lengthBytes[0])<<8 | int(lengthBytes[1]) - 2

			commentData := make([]byte, length)
			if _, err := io.ReadFull(reader, commentData); err != nil {
				return nil, err
			}

			metadata.Comments = append(metadata.Comments, string(commentData))

		case markerSOS:
			// Start of scan - skip compressed data until next marker
			lengthBytes := make([]byte, 2)
			if _, err := io.ReadFull(reader, lengthBytes); err != nil {
				return nil, err
			}
			length := int(lengthBytes[0])<<8 | int(lengthBytes[1])

			// Skip the SOS header
			headerData := make([]byte, length-2)
			if _, err := io.ReadFull(reader, headerData); err != nil {
				return nil, err
			}

			// Now skip compressed data until we find the next marker
			// This is tricky because 0xFF bytes in the compressed data are escaped with a 0x00 byte
			for {
				b, err := reader.ReadByte()
				if err != nil {
					break
				}
				if b != 0xFF {
					continue
				}

				nextByte, err := reader.ReadByte()
				if err != nil {
					break
				}

				// If the next byte is 0x00, it's an escaped 0xFF in the data
				if nextByte == 0x00 {
					continue
				}

				// We found a marker - push it back and break
				reader.UnreadByte()
				reader.UnreadByte()
				break
			}

		default:
			// Skip other segments
			lengthBytes := make([]byte, 2)
			if _, err := io.ReadFull(reader, lengthBytes); err != nil {
				// If we can't read a length, we might be at the end or in compressed data
				continue
			}
			length := int(lengthBytes[0])<<8 | int(lengthBytes[1])

			// Skip the segment data
			if length > 2 {
				segmentData := make([]byte, length-2)
				if _, err := io.ReadFull(reader, segmentData); err != nil {
					continue
				}
			}
		}
	}

	return metadata, nil
}

// DetectJPEGSteganography analyzes JPEG metadata to detect signs of steganography
func DetectJPEGSteganography(metadata *JPEGMetadata) bool {
	// Check for appended data after EOI
	if metadata.HasAppendedData {
		return true
	}

	// Check for unusual comments that might contain steganographic data
	if len(metadata.Comments) > 0 {
		for _, comment := range metadata.Comments {
			// Check for suspiciously long comments
			if len(comment) > 1000 {
				return true
			}

			// Check for binary data in comments
			binaryCount := 0
			for _, b := range comment {
				if b < 32 && b != 9 && b != 10 && b != 13 {
					binaryCount++
				}
			}
			if float64(binaryCount)/float64(len(comment)) > 0.1 {
				return true
			}

			// Check for encoded text in comments (base64, hex, etc.)
			if containsEncodedText(comment) {
				return true
			}
		}
	}

	// Check for abnormal marker sequences (unusual order or repetition)
	if hasAbnormalMarkerSequence(metadata.MarkerSequence) {
		return true
	}

	// Analyze quantization tables for signs of manipulation
	for _, table := range metadata.QuantizationTables {
		if hasModifiedQuantizationValues(table) {
			return true
		}
	}

	return false
}

// containsEncodedText checks if the string might contain encoded data
func containsEncodedText(text string) bool {
	// Check for Base64 pattern (at least 20 chars, mostly A-Za-z0-9+/=)
	if len(text) > 20 {
		base64Pattern := regexp.MustCompile(`^[A-Za-z0-9+/=]{20,}$`)
		if base64Pattern.MatchString(text) {
			return true
		}

		// Check for long hex string
		hexPattern := regexp.MustCompile(`^[A-Fa-f0-9]{20,}$`)
		if hexPattern.MatchString(text) {
			return true
		}

		// Check for high entropy in the text
		if textEntropy(text) > 4.5 {
			return true
		}
	}

	return false
}

// textEntropy calculates the Shannon entropy of a text string
func textEntropy(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	// Count character frequencies
	freqs := make(map[rune]int)
	for _, char := range text {
		freqs[char]++
	}

	// Calculate entropy
	entropy := 0.0
	textLen := float64(len(text))
	for _, count := range freqs {
		p := float64(count) / textLen
		entropy -= p * logBase2(p)
	}

	return entropy
}

// logBase2 calculates log base 2 of a value
func logBase2(x float64) float64 {
	return math.Log(x) / math.Log(2)
}

// hasAbnormalMarkerSequence checks for unusual marker sequences
func hasAbnormalMarkerSequence(markers []byte) bool {
	// Check for marker sequences that shouldn't occur in standard JPEGs
	// This is a simplified check - would need more complex analysis in production

	for i := 0; i < len(markers)-3; i += 2 {
		// Check for repeated markers in sequence
		if i > 0 && markers[i] == 0xFF && markers[i+1] == markers[i-1] &&
			markers[i+1] != 0x00 && markers[i-1] != 0x00 {
			return true
		}
	}

	return false
}

// hasModifiedQuantizationValues checks if a quantization table has been manipulated
func hasModifiedQuantizationValues(table []byte) bool {
	// Improved version with better heuristics

	// Check for unusual patterns in the table
	zeroCount := 0
	oneCount := 0
	evenOddDiffs := 0 // Count differences between adjacent values

	for i, val := range table {
		if val == 0 {
			zeroCount++
		}
		if val == 1 {
			oneCount++
		}

		// Compare with adjacent values
		if i > 0 && (val%2) != (table[i-1]%2) {
			evenOddDiffs++
		}
	}

	// If more than 10% of values are zero or 1, might indicate manipulation
	if float64(zeroCount)/float64(len(table)) > 0.1 || float64(oneCount)/float64(len(table)) > 0.15 {
		return true
	}

	// If there's a highly regular pattern of even/odd values
	if float64(evenOddDiffs)/float64(len(table)-1) > 0.8 {
		return true
	}

	return false
}

// DetectJSteg checks for signs of JSteg steganography
func DetectJSteg(metadata *JPEGMetadata) bool {
	// JSteg modifies the DCT coefficients to hide data
	// This would require a more sophisticated analysis of the actual DCT coefficients

	// For this simplified version, we'll check for patterns in quantization tables
	// that might indicate JSteg usage
	if len(metadata.QuantizationTables) > 0 {
		// JSteg usually doesn't modify the quantization tables directly,
		// but the pattern of DCT coefficients would be affected
		// For a simple check, see if tables look unusual
		for _, table := range metadata.QuantizationTables {
			// Count values that are 1, which is unusual in standard tables
			oneCount := 0
			for _, val := range table {
				if val == 1 {
					oneCount++
				}
			}

			// If many values are 1, it might indicate tampering
			if oneCount > 10 {
				return true
			}
		}
	}

	return false
}

// DetectF5 checks for signs of F5 steganography
func DetectF5(metadata *JPEGMetadata) bool {
	// F5 has distinct patterns in how it modifies DCT coefficients

	// One indicator of F5 is a specific marker in the Huffman tables
	// This is a highly simplified check and would need deeper analysis
	for _, table := range metadata.HuffmanTables {
		if len(table) > 0 && table[0] == 0xF5 {
			return true
		}
	}

	// F5 often results in a specific distribution pattern in the DCT coefficients
	// without access to those coefficients directly, we can only approximate

	return false
}

// DetectOutguess checks for signs of Outguess steganography
func DetectOutguess(metadata *JPEGMetadata) bool {
	// Outguess modifies DCT coefficients in a way that preserves histogram statistics

	// Simple check: Outguess often modifies images to be progressive for compatibility
	if metadata.IsProgressive {
		// This alone is not sufficient, but could be an indicator
		// when combined with other checks
		return true
	}

	return false
}

// CheckAppendedData checks if there is data appended after the JPEG EOI marker
func CheckAppendedData(filename string) (bool, int) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return false, 0
	}

	// Find the last EOI marker (FF D9)
	for i := len(data) - 2; i >= 0; i-- {
		if data[i] == 0xFF && data[i+1] == 0xD9 {
			if i+2 < len(data) {
				return true, len(data) - (i + 2)
			}
			break
		}
	}

	return false, 0
}

// ExtractAppendedData extracts any data appended after the EOI marker
func ExtractAppendedData(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Find the last EOI marker (FF D9)
	for i := len(data) - 2; i >= 0; i-- {
		if data[i] == 0xFF && data[i+1] == 0xD9 {
			if i+2 < len(data) {
				appendedData := data[i+2:]

				// Try to detect if the appended data contains plaintext
				if IsASCIIPrintable(appendedData) {
					return appendedData, nil
				}

				// Check if it's an encoded message (base64, hex, etc)
				if containsEncodedBytes(appendedData) {
					return appendedData, nil
				}

				return appendedData, nil
			}
			break
		}
	}

	return nil, fmt.Errorf("no appended data found")
}

// containsEncodedBytes checks if byte array might contain encoded data
func containsEncodedBytes(data []byte) bool {
	// Convert to string for regex checks
	str := string(data)

	// Check for Base64 pattern (at least 20 chars, mostly A-Za-z0-9+/=)
	base64Pattern := regexp.MustCompile(`[A-Za-z0-9+/=]{20,}`)
	if base64Pattern.MatchString(str) {
		return true
	}

	// Check for hex encoded data
	hexPattern := regexp.MustCompile(`[A-Fa-f0-9]{20,}`)
	if hexPattern.MatchString(str) {
		return true
	}

	// Check entropy - high entropy suggests encoded or encrypted data
	if ComputeEntropy(data) > 3.5 {
		return true
	}

	return false
}

// ScanForPlaintextStego searches for plaintext hidden in various JPEG segments
func ScanForPlaintextStego(filename string) ([]string, error) {
	var findings []string

	// Open the JPEG file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the entire file to search through
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Gather positions of all known segments to avoid false positives
	knownSegments := identifyKnownSegments(data)

	// Check for ASCII text throughout the file
	// Use sliding window approach to detect potential plaintext
	for i := 0; i < len(data)-20; i++ {
		// Skip if we're in a known segment that typically contains structured data
		if isInKnownSegment(i, knownSegments) {
			continue
		}

		// Check 40-byte windows
		end := min(i+40, len(data))
		window := data[i:end]

		// If window has high percentage of printable ASCII
		printableCount := 0
		for _, b := range window {
			if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
				printableCount++
			}
		}

		// If more than 95% is printable (increased from 90%)
		if float64(printableCount)/float64(len(window)) > 0.95 {
			// Filter out false positives by checking for common JPEG patterns
			text := string(window)

			if !isCommonJPEGPattern(text) && isLikelyPlaintext(text) {
				// Clean up the text by removing non-printable characters
				cleanText := cleanupText(text)

				// Only include if it looks meaningful (more strict check)
				if len(cleanText) >= 10 && containsMeaningfulText(cleanText) {
					findings = append(findings, cleanText)
					// Skip ahead to avoid duplicate detections
					i += len(window)
				}
			}
		}
	}

	// Check for text/messages in EXIF data - improved version
	exifText, err := extractImprovedEXIFText(filename)
	if err == nil && len(exifText) > 0 {
		findings = append(findings, exifText...)
	}

	// Filter out any remaining false positives
	return filterFalsePositives(findings), nil
}

// identifyKnownSegments identifies positions of standard JPEG segments
func identifyKnownSegments(data []byte) []segment {
	var segments []segment

	for i := 0; i < len(data)-2; i++ {
		// Look for marker prefix
		if data[i] == 0xFF && data[i+1] != 0x00 && data[i+1] != 0xFF {
			markerType := data[i+1]

			// Skip markers that don't have a length field
			if markerType == markerSOI || markerType == markerEOI {
				continue
			}

			// Get segment length for markers that have length fields
			if i+3 < len(data) {
				length := int(data[i+2])<<8 | int(data[i+3])
				if length >= 2 && i+length < len(data) {
					segments = append(segments, segment{
						start:  i,
						end:    i + length + 2,
						marker: markerType,
					})
					i = i + length + 1 // Skip to end of segment
				}
			}
		}
	}

	return segments
}

// segment represents a JPEG segment
type segment struct {
	start  int
	end    int
	marker byte
}

// isInKnownSegment checks if a position is part of a known JPEG segment
func isInKnownSegment(pos int, segments []segment) bool {
	for _, seg := range segments {
		if pos >= seg.start && pos < seg.end {
			// Skip checking text in certain segments that can contain
			// legitimate text like comments or EXIF
			if seg.marker == markerCOM || seg.marker == markerAPP1 {
				return false // Allow text detection in these segments
			}
			return true // In other segments, skip text detection
		}
	}
	return false
}

// isCommonJPEGPattern identifies common strings in JPEG files that aren't steganography
func isCommonJPEGPattern(text string) bool {
	commonPatterns := []string{
		// Huffman tables often contain these patterns
		"%&'()*456789:CDEFGHIJ",
		"&'()*56789:CDEFGHIJSTUVWXYZcdefghij",
		"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklm",

		// JFIF/EXIF standard strings
		"JFIF", "Exif", "http://", "Adobe", "ICC_PROFILE",

		// Common EXIF tags as strings
		"Nikon", "Canon", "OLYMPUS", "PENTAX", "SONY",
	}

	for _, pattern := range commonPatterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}

	// Also check for repetitive sequences that appear in huffman tables
	if containsRepetitivePattern(text) {
		return true
	}

	return false
}

// containsRepetitivePattern checks if the text has repetitive sequences
// like "2222222222" which are common in JPEG encoding tables
func containsRepetitivePattern(text string) bool {
	// Check for sequences of the same character repeating more than 5 times
	for i := 0; i < len(text)-5; i++ {
		if text[i] == text[i+1] && text[i] == text[i+2] &&
			text[i] == text[i+3] && text[i] == text[i+4] {
			return true
		}
	}

	// Check for repeating 2-char patterns
	if len(text) > 10 {
		for i := 0; i < len(text)-6; i += 2 {
			if text[i] == text[i+2] && text[i] == text[i+4] &&
				text[i+1] == text[i+3] && text[i+1] == text[i+5] {
				return true
			}
		}
	}

	return false
}

// isLikelyPlaintext performs more precise checks to determine if text is likely
// to be actual human-readable content rather than encoded data
func isLikelyPlaintext(text string) bool {
	// Must have at least 10 chars to be considered
	if len(text) < 10 {
		return false
	}

	// Check for common word separators (spaces, punctuation)
	spaceCount := 0
	for _, c := range text {
		if c == ' ' || c == '.' || c == ',' || c == '!' || c == '?' || c == ';' || c == ':' {
			spaceCount++
		}
	}

	// Real text usually has word separators (unless it's a password or key)
	// but we'll accept it even without spaces as it could be concatenated words

	// Check for a reasonable distribution of characters
	// Binary data often has unusual distributions
	freqs := make(map[byte]int)
	for i := 0; i < len(text); i++ {
		freqs[text[i]]++
	}

	// If a single character dominates (>50%), it's probably not meaningful text
	for _, count := range freqs {
		if float64(count)/float64(len(text)) > 0.5 {
			return false
		}
	}

	// Natural language often follows certain patterns
	// like having vowels mixed with consonants
	vowelCount := 0
	for _, c := range strings.ToLower(text) {
		if c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u' {
			vowelCount++
		}
	}

	// Most languages have 25-60% vowels in normal text
	// Allow a bit wider range for potentially encoded but human-readable text
	vowelRatio := float64(vowelCount) / float64(len(text))
	if vowelRatio < 0.1 || vowelRatio > 0.7 {
		// Unless it's all uppercase, which might be a key or code
		if strings.ToUpper(text) != text {
			return false
		}
	}

	return true
}

// cleanupText removes non-printable characters and trims the text
func cleanupText(text string) string {
	var result strings.Builder

	for _, c := range text {
		if (c >= 32 && c <= 126) || c == '\n' || c == '\r' || c == '\t' {
			result.WriteRune(c)
		}
	}

	return strings.TrimSpace(result.String())
}

// containsMeaningfulText checks if the text is likely to be meaningful
// rather than random sequences or encoding artifacts
func containsMeaningfulText(text string) bool {
	// English words usually have 4+ letters
	// Look for at least a few letters in sequence
	letterGroups := 0
	currentGroup := 0

	for _, c := range text {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			currentGroup++
			if currentGroup >= 4 {
				letterGroups++
			}
		} else {
			currentGroup = 0
		}
	}

	// Require at least one group of 4+ letters
	// or special cases like "SECRET:" or "PASSWORD:"
	keywords := []string{"secret", "password", "key", "login", "user", "admin", "credit"}
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(text), keyword) {
			return true
		}
	}

	return letterGroups > 0
}

// extractImprovedEXIFText is an improved version of ExtractEXIFText
func extractImprovedEXIFText(filename string) ([]string, error) {
	var results []string

	// Use a placeholder implementation since ExtractEXIFText is missing
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file content
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Simplified EXIF text extraction
	rawResults := []string{}
	// Look for APP1 segment which typically contains EXIF
	for i := 0; i < len(data)-10; i++ {
		if data[i] == 0xFF && data[i+1] == markerAPP1 {
			// Extract segment length
			length := int(data[i+2])<<8 | int(data[i+3])
			if i+2+length <= len(data) && length > 10 {
				// Extract the segment data
				segment := data[i+4 : i+2+length]
				// Look for text strings within the segment
				for j := 0; j < len(segment)-10; j++ {
					end := min(j+40, len(segment))
					window := segment[j:end]

					// Check if window has printable ASCII
					printableCount := 0
					for _, b := range window {
						if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
							printableCount++
						}
					}

					// If most chars are printable
					if float64(printableCount)/float64(len(window)) > 0.9 {
						text := string(window)
						rawResults = append(rawResults, cleanupText(text))
						j += len(window) - 1
					}
				}
			}
		}
	}

	// Apply more strict filtering
	for _, text := range rawResults {
		if !isCommonJPEGPattern(text) && isLikelyPlaintext(text) && containsMeaningfulText(text) {
			results = append(results, text)
		}
	}

	return results, nil
}

// filterFalsePositives removes likely false positive results
func filterFalsePositives(findings []string) []string {
	var filtered []string

	for _, text := range findings {
		// Skip very short strings
		if len(text) < 10 {
			continue
		}

		// Skip strings with too many numbers or symbols
		alphaCount := 0
		for _, c := range text {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				alphaCount++
			}
		}

		if float64(alphaCount)/float64(len(text)) < 0.4 {
			continue
		}

		// Additional checks for Huffman table patterns
		if strings.Contains(text, "UVWXYZ") && strings.Contains(text, "cdefghi") {
			continue
		}

		filtered = append(filtered, text)
	}

	return filtered
}

// CheckQuantizationTables analyzes quantization tables for suspicious patterns
func CheckQuantizationTables(metadata *JPEGMetadata) (bool, int) {
	if len(metadata.QuantizationTables) == 0 {
		return false, 0
	}

	// Standard luminance and chrominance quantization tables for quality 50
	// Source: JPEG standard
	standardLumTable := []byte{
		16, 11, 10, 16, 24, 40, 51, 61,
		12, 12, 14, 19, 26, 58, 60, 55,
		14, 13, 16, 24, 40, 57, 69, 56,
		14, 17, 22, 29, 51, 87, 80, 62,
		18, 22, 37, 56, 68, 109, 103, 77,
		24, 35, 55, 64, 81, 104, 113, 92,
		49, 64, 78, 87, 103, 121, 120, 101,
		72, 92, 95, 98, 112, 100, 103, 99,
	}

	standardChrTable := []byte{
		17, 18, 24, 47, 99, 99, 99, 99,
		18, 21, 26, 66, 99, 99, 99, 99,
		24, 26, 56, 99, 99, 99, 99, 99,
		47, 66, 99, 99, 99, 99, 99, 99,
		99, 99, 99, 99, 99, 99, 99, 99,
		99, 99, 99, 99, 99, 99, 99, 99,
		99, 99, 99, 99, 99, 99, 99, 99,
		99, 99, 99, 99, 99, 99, 99, 99,
	}

	// Check each quantization table for significant deviations
	modified := false

	for _, table := range metadata.QuantizationTables {
		// Extract precision and table ID from the first byte
		if len(table) < 65 {
			continue // Not enough data for a full table
		}

		// Check precision and table ID
		precisionAndID := table[0]
		precision := (precisionAndID >> 4) & 0x0F // 0 = 8-bit, 1 = 16-bit
		tableID := precisionAndID & 0x0F

		// Get just the quantization values
		var qTable []byte
		if precision == 0 {
			// 8-bit precision
			qTable = table[1:65] // 64 values, each 1 byte
		} else {
			// 16-bit precision
			if len(table) < 129 {
				continue // Not enough data
			}
			// Skip 16-bit table analysis for simplicity
			continue
		}

		// Compare with standard tables
		standardTable := standardLumTable
		if tableID > 0 {
			standardTable = standardChrTable
		}

		// Count deviations from standard values
		deviations := 0
		for i := 0; i < 64 && i < len(qTable); i++ {
			// Allow for some variation due to quality settings
			diff := int(qTable[i]) - int(standardTable[i])
			if diff < -10 || diff > 10 {
				deviations++
			}
		}

		// If more than 25% of values deviate significantly, consider it modified
		if float64(deviations)/64.0 > 0.25 {
			modified = true
		}
	}

	return modified, len(metadata.QuantizationTables)
}

// ScanForPolyglotFile checks if the JPEG file is also another valid file format
func ScanForPolyglotFile(filename string) (bool, string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return false, ""
	}

	// Check for ZIP signature at the end of the file
	if len(data) > 30 && bytes.Contains(data[len(data)-30:], []byte("PK\x03\x04")) {
		return true, "ZIP"
	}

	// Check for PDF signature
	if bytes.Contains(data, []byte("%PDF-")) {
		return true, "PDF"
	}

	// Check for RAR signature
	if bytes.Contains(data, []byte("Rar!\x1A\x07")) {
		return true, "RAR"
	}

	// Check for PNG signature
	if bytes.Contains(data, []byte("\x89PNG\r\n\x1A\n")) {
		return true, "PNG"
	}

	return false, ""
}
