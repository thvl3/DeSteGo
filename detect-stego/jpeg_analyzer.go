package main

import (
	"bufio"
	//"bytes"
	"fmt"
	//"image"
	//"image/jpeg"
	"io"
	"os"
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
		}
	}

	// Analyze quantization tables for signs of manipulation
	for _, table := range metadata.QuantizationTables {
		// Check if the table has been manipulated (specific checks would depend on knowledge
		// of standard quantization tables and how stego tools modify them)
		if hasModifiedQuantizationValues(table) {
			return true
		}
	}

	return false
}

// hasModifiedQuantizationValues checks if a quantization table has been manipulated
// This is a simplified example and would need to be enhanced with actual knowledge of
// how specific steganography tools modify quantization tables
func hasModifiedQuantizationValues(table []byte) bool {
	// This is a simplified check
	// In real implementation, you'd compare against standard tables or check for patterns

	// Check for unusual patterns in the table
	zeroCount := 0
	for _, val := range table {
		if val == 0 {
			zeroCount++
		}
	}

	// If more than 10% of values are zero, it might indicate manipulation
	if float64(zeroCount)/float64(len(table)) > 0.1 {
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
				return data[i+2:], nil
			}
			break
		}
	}

	return nil, fmt.Errorf("no appended data found")
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
