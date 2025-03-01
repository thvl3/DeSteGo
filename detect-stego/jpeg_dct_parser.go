package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

// DCTCoefficientBlock represents an 8×8 block of DCT coefficients
type DCTCoefficientBlock struct {
	Coefficients [64]int16  // 64 coefficients in zigzag order
	Original     [64]uint16 // Original values for comparison
	BlockX       int        // X position of block in image
	BlockY       int        // Y position of block in image
	Component    int        // Color component (0=Y, 1=Cb, 2=Cr)
}

// JPEGDCTData holds all DCT coefficient data from a JPEG file
type JPEGDCTData struct {
	Blocks        []DCTCoefficientBlock
	Width         int
	Height        int
	Components    int
	QuantTables   [][]uint16
	HuffmanTables map[int]HuffmanTable
}

// HuffmanTable represents a JPEG Huffman table
type HuffmanTable struct {
	TableClass int    // 0=DC, 1=AC
	TableID    int    // Table identifier
	Codes      []byte // Huffman codes
	Values     []byte // Values for the codes
}

// ParseJPEGDCTCoefficients extracts DCT coefficients from a JPEG file
func ParseJPEGDCTCoefficients(filename string) (*JPEGDCTData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Check JPEG signature
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, fmt.Errorf("not a valid JPEG file")
	}

	result := &JPEGDCTData{
		Blocks:        make([]DCTCoefficientBlock, 0),
		QuantTables:   make([][]uint16, 0),
		HuffmanTables: make(map[int]HuffmanTable),
	}

	// Parse JPEG segments to find SOF and SOS segments
	var pos int = 2 // Skip JPEG header
	var foundSOF bool
	var compInfo []struct {
		ID                int
		HSamplingFactor   int
		VSamplingFactor   int
		QuantizationTable int
	}
	var dcHuffmanTables [4][]byte
	var acHuffmanTables [4][]byte
	var quantTables [4][]uint16

	for pos < len(data)-1 {
		// All markers start with 0xFF
		if data[pos] != 0xFF {
			pos++
			continue
		}

		markerType := data[pos+1]
		pos += 2 // Move past marker

		// Skip 0xFF padding
		if markerType == 0x00 {
			continue
		}

		// Check for SOS (Start of Scan) - we'll process the scan data separately
		if markerType == 0xDA {
			if !foundSOF {
				return nil, fmt.Errorf("found SOS marker before SOF marker")
			}

			// Here's where we'd decode the scan data to extract DCT coefficients
			// This is complex and requires implementing the JPEG decoding process
			// to extract coefficients

			// Parse scan header
			//segmentLength := int(binary.BigEndian.Uint16(data[pos : pos+2]))
			numComponents := int(data[pos+2])
			pos += 3 // Skip length and numComponents

			// Read component info for scan
			componentSelectors := make([]int, numComponents)
			dcTableSelectors := make([]int, numComponents)
			acTableSelectors := make([]int, numComponents)

			for i := 0; i < numComponents; i++ {
				componentSelectors[i] = int(data[pos])
				dcTableSelectors[i] = int(data[pos+1] >> 4)
				acTableSelectors[i] = int(data[pos+1] & 0x0F)
				pos += 2
			}

			// Skip spectral selection bytes (3 bytes)
			pos += 3

			// Now extract entropy-coded data until we find next marker
			entropyData := extractEntropyCodedData(data, pos)

			// Process entropy coded data to extract DCT coefficients
			coeffs, err := decodeDCTCoefficients(
				entropyData,
				dcHuffmanTables,
				acHuffmanTables,
				quantTables,
				result.Width,
				result.Height,
				compInfo,
				componentSelectors,
				dcTableSelectors,
				acTableSelectors)

			if err != nil {
				return nil, fmt.Errorf("failed to decode DCT coefficients: %v", err)
			}

			result.Blocks = coeffs

			// Find the next marker after the entropy coded data
			pos += len(entropyData)
			continue
		}

		// EOI (End of Image)
		if markerType == 0xD9 {
			break
		}

		// All other marker segments include a length field
		if pos+2 > len(data) {
			return nil, fmt.Errorf("unexpected end of file while reading marker length")
		}

		segmentLength := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		if segmentLength < 2 {
			return nil, fmt.Errorf("invalid segment length")
		}

		// Check bounds
		if pos+segmentLength > len(data) {
			return nil, fmt.Errorf("segment length exceeds file size")
		}

		// Process SOF segments (Start of Frame) - 0xC0 for baseline DCT
		if markerType == 0xC0 || markerType == 0xC1 || markerType == 0xC2 {
			foundSOF = true
			//precision := int(data[pos+2])
			height := int(binary.BigEndian.Uint16(data[pos+3 : pos+5]))
			width := int(binary.BigEndian.Uint16(data[pos+5 : pos+7]))
			numComponents := int(data[pos+7])

			result.Width = width
			result.Height = height
			result.Components = numComponents

			// Read component info
			compInfo = make([]struct {
				ID                int
				HSamplingFactor   int
				VSamplingFactor   int
				QuantizationTable int
			}, numComponents)

			for i := 0; i < numComponents; i++ {
				offset := pos + 8 + i*3
				compInfo[i].ID = int(data[offset])
				compInfo[i].HSamplingFactor = int(data[offset+1] >> 4)
				compInfo[i].VSamplingFactor = int(data[offset+1] & 0x0F)
				compInfo[i].QuantizationTable = int(data[offset+2])
			}
		}

		// Process DQT segments (Define Quantization Table)
		if markerType == 0xDB {
			// Process quantization tables
			tableOffset := pos + 2
			for tableOffset < pos+segmentLength {
				tableInfo := data[tableOffset]
				precision := (tableInfo >> 4) & 0x0F // 0 = 8 bit, 1 = 16 bit
				tableID := tableInfo & 0x0F

				tableOffset++

				// Read the table data
				tableData := make([]uint16, 64)
				if precision == 0 {
					// 8-bit table
					for i := 0; i < 64; i++ {
						tableData[i] = uint16(data[tableOffset+i])
					}
					tableOffset += 64
				} else {
					// 16-bit table
					for i := 0; i < 64; i++ {
						tableData[i] = binary.BigEndian.Uint16(data[tableOffset+i*2 : tableOffset+i*2+2])
					}
					tableOffset += 128
				}

				// Store the quantization table
				if int(tableID) < len(quantTables) {
					quantTables[tableID] = tableData
					result.QuantTables = append(result.QuantTables, tableData)
				}
			}
		}

		// Process DHT segments (Define Huffman Table)
		if markerType == 0xC4 {
			tableOffset := pos + 2
			for tableOffset < pos+segmentLength {
				tableInfo := data[tableOffset]
				tableClass := (tableInfo >> 4) & 0x0F // 0 = DC, 1 = AC
				tableID := tableInfo & 0x0F

				tableOffset++

				// Read code counts for each bit length (1-16)
				codeCounts := make([]byte, 16)
				copy(codeCounts, data[tableOffset:tableOffset+16])
				tableOffset += 16

				// Calculate total number of codes
				totalCodes := 0
				for _, count := range codeCounts {
					totalCodes += int(count)
				}

				// Read the values
				values := make([]byte, totalCodes)
				copy(values, data[tableOffset:tableOffset+totalCodes])
				tableOffset += totalCodes

				// Store the Huffman table
				huffKey := (int(tableClass) << 4) | int(tableID)
				result.HuffmanTables[huffKey] = HuffmanTable{
					TableClass: int(tableClass),
					TableID:    int(tableID),
					Codes:      codeCounts,
					Values:     values,
				}

				// Also store for easier access during decoding
				if tableClass == 0 { // DC table
					dcHuffmanTables[tableID] = values
				} else { // AC table
					acHuffmanTables[tableID] = values
				}
			}
		}

		// Move to next segment
		pos += segmentLength

	}

	return result, nil
}

// extractEntropyCodedData extracts the bitstream between SOS and the next marker
// This is a simplification as real JPEG data can have stuffed bytes
func extractEntropyCodedData(data []byte, startPos int) []byte {
	var entropyData []byte
	pos := startPos

	// Find the next marker
	for pos < len(data)-1 {
		if data[pos] == 0xFF {
			// Check for stuffed byte (FF followed by 00)
			if pos+1 < len(data) && data[pos+1] == 0x00 {
				entropyData = append(entropyData, 0xFF)
				pos += 2
				continue
			}

			// Check if it's a real marker
			if pos+1 < len(data) && data[pos+1] >= 0xC0 {
				break
			}
		}

		entropyData = append(entropyData, data[pos])
		pos++
	}

	return entropyData
}

// decodeDCTCoefficients decodes the entropy-coded data to extract DCT coefficients
// This is a simplified stub function - a full implementation would be quite complex
func decodeDCTCoefficients(
	entropyData []byte,
	dcHuffmanTables [4][]byte,
	acHuffmanTables [4][]byte,
	quantTables [4][]uint16,
	width, height int,
	compInfo []struct {
		ID                int
		HSamplingFactor   int
		VSamplingFactor   int
		QuantizationTable int
	},
	componentSelectors []int,
	dcTableSelectors []int,
	acTableSelectors []int) ([]DCTCoefficientBlock, error) {

	// This is a placeholder for a real implementation
	// For detecting steganography, we could use a dedicated JPEG library
	// to extract the coefficients

	// For now, return an error indicating this is not implemented yet
	return nil, fmt.Errorf("DCT coefficient decoding not fully implemented")
}

// AnalyzeDCTCoefficientHistogram analyzes the histogram of DCT coefficients
// to detect anomalies that might indicate steganography
func AnalyzeDCTCoefficientHistogram(blocks []DCTCoefficientBlock) map[string]float64 {
	results := make(map[string]float64)

	// Count coefficient statistics
	evenCount := 0
	oddCount := 0
	zeroCount := 0
	oneCount := 0
	minusOneCount := 0
	totalCoeffs := 0

	// Analyze only AC coefficients (skip DC coefficient which is the first one)
	for _, block := range blocks {
		for i := 1; i < 64; i++ { // Skip DC coefficient (index 0)
			coeff := block.Coefficients[i]
			totalCoeffs++

			if coeff == 0 {
				zeroCount++
			} else if coeff == 1 {
				oneCount++
			} else if coeff == -1 {
				minusOneCount++
			}

			if coeff%2 == 0 {
				evenCount++
			} else {
				oddCount++
			}
		}
	}

	// Calculate statistics
	if totalCoeffs > 0 {
		results["even_ratio"] = float64(evenCount) / float64(totalCoeffs)
		results["odd_ratio"] = float64(oddCount) / float64(totalCoeffs)
		results["zero_ratio"] = float64(zeroCount) / float64(totalCoeffs)
		results["one_ratio"] = float64(oneCount) / float64(totalCoeffs)
		results["minus_one_ratio"] = float64(minusOneCount) / float64(totalCoeffs)

		// Calculate entropy of coefficient values
		valueCounts := make(map[int16]int)
		for _, block := range blocks {
			for i := 1; i < 64; i++ { // Skip DC
				coeff := block.Coefficients[i]
				valueCounts[coeff]++
			}
		}

		entropy := 0.0
		for _, count := range valueCounts {
			p := float64(count) / float64(totalCoeffs)
			entropy -= p * math.Log2(p)
		}
		results["entropy"] = entropy

		// Calculate suspiciousness score based on known steganography patterns
		// Different algorithms have different fingerprints
		suspiciousness := 0.0

		// JSteg often creates an abnormal ratio of even vs. odd coefficients
		evenOddDiff := math.Abs(results["even_ratio"] - 0.5)
		if evenOddDiff < 0.02 { // Too perfectly balanced
			suspiciousness += (0.02 - evenOddDiff) * 50
		}

		// F5 tends to reduce the number of 1/-1 coefficients
		expectedOneRatio := 0.15 // Approximate for natural images
		if results["one_ratio"] < expectedOneRatio*0.7 {
			suspiciousness += (expectedOneRatio - results["one_ratio"]) * 10
		}

		// StegHide often affects the distribution of zero coefficients
		if results["zero_ratio"] > 0.6 {
			suspiciousness += (results["zero_ratio"] - 0.6) * 5
		}

		results["suspiciousness"] = suspiciousness
	}

	return results
}

// DetectStegoByCoefficientAnalysis analyzes DCT coefficients to detect steganography
func DetectStegoByCoefficientAnalysis(dctData *JPEGDCTData) (bool, map[string]float64, string) {
	if len(dctData.Blocks) == 0 {
		return false, nil, "no DCT coefficient data available"
	}

	results := AnalyzeDCTCoefficientHistogram(dctData.Blocks)

	// Determine which algorithm it might be
	algorithm := "unknown"
	threshold := 0.5 // Default threshold for detection

	// Check for different steganography algorithms

	// JSteg characteristics
	if results["suspiciousness"] > threshold &&
		math.Abs(results["even_ratio"]-0.5) < 0.02 {
		algorithm = "JSteg"
	}

	// F5 characteristics
	if results["suspiciousness"] > threshold &&
		results["one_ratio"] < 0.08 &&
		results["minus_one_ratio"] < 0.08 {
		algorithm = "F5"
	}

	// StegHide characteristics
	if results["suspiciousness"] > threshold &&
		results["zero_ratio"] > 0.65 &&
		results["entropy"] < 2.0 {
		algorithm = "StegHide"
	}

	// OutGuess characteristics
	if results["suspiciousness"] > threshold &&
		math.Abs(results["even_ratio"]-results["odd_ratio"]) > 0.2 {
		algorithm = "OutGuess"
	}

	return results["suspiciousness"] > threshold, results, algorithm
}

// ExtractSteganographicData attempts to extract hidden data from DCT coefficients
func ExtractSteganographicData(dctData *JPEGDCTData, algorithm string) ([]byte, error) {
	var extractedData []byte

	// For most algorithms, we need to look at LSBs of non-zero coefficients
	switch algorithm {
	case "JSteg":
		// JSteg embeds data in LSBs of non-zero AC DCT coefficients
		extractedData = extractJStegData(dctData.Blocks)

	case "F5":
		// F5 changes DCT coefficients to encode bits through specific patterns
		extractedData = extractF5Data(dctData.Blocks)

	case "StegHide":
		// StegHide uses a pseudorandom order of coefficients and may use permutation
		extractedData = extractStegHideData(dctData.Blocks)

	case "OutGuess":
		// OutGuess preserves the statistical properties while embedding data
		extractedData = extractOutGuessData(dctData.Blocks)

	default:
		// Try generic LSB extraction from DCT coefficients
		extractedData = extractGenericLSB(dctData.Blocks)
	}

	// Check if the extracted data seems valid
	if len(extractedData) == 0 {
		return nil, fmt.Errorf("no data could be extracted")
	}

	// Try to detect if there's a length header
	if len(extractedData) > 4 {
		possibleLength := binary.BigEndian.Uint32(extractedData[:4])
		if possibleLength < uint32(len(extractedData)) && possibleLength > 0 {
			// Likely has a length header
			return extractedData[4 : 4+possibleLength], nil
		}
	}

	return extractedData, nil
}

// extractJStegData extracts data embedded using the JSteg algorithm
func extractJStegData(blocks []DCTCoefficientBlock) []byte {
	var bits []byte

	// JSteg embeds data in LSBs of non-zero AC coefficients
	for _, block := range blocks {
		for i := 1; i < 64; i++ { // Skip DC coefficient (index 0)
			coeff := block.Coefficients[i]
			if coeff != 0 {
				// Extract the LSB
				bit := byte(coeff & 1)
				bits = append(bits, bit)
			}
		}
	}

	// Convert bits to bytes
	return convertBitsToBytes(bits)
}

// extractF5Data attempts to extract data embedded using the F5 algorithm
func extractF5Data(blocks []DCTCoefficientBlock) []byte {
	// F5 is more complex and uses matrix encoding
	// This is a simplified extraction that might not work with all F5 variants
	var bits []byte

	for _, block := range blocks {
		for i := 1; i < 64; i++ {
			coeff := block.Coefficients[i]
			if coeff != 0 {
				// F5 uses a more complex scheme, but LSBs still carry significant information
				bit := byte(coeff & 1)
				bits = append(bits, bit)
			}
		}
	}

	return convertBitsToBytes(bits)
}

// extractStegHideData attempts to extract data embedded using the StegHide algorithm
func extractStegHideData(blocks []DCTCoefficientBlock) []byte {
	// StegHide uses a pseudo-random walk through coefficients
	// This is a simplified extraction that won't work for password-protected content
	var bits []byte

	for _, block := range blocks {
		for i := 0; i < 64; i++ {
			coeff := block.Coefficients[i]
			// StegHide tends to modify values to even/odd
			bit := byte(coeff & 1)
			bits = append(bits, bit)
		}
	}

	return convertBitsToBytes(bits)
}

// extractOutGuessData attempts to extract data embedded using the OutGuess algorithm
func extractOutGuessData(blocks []DCTCoefficientBlock) []byte {
	// OutGuess uses a subset of coefficients to maintain histogram statistics
	// This is a simplified approach that might not work for all cases
	var bits []byte

	for _, block := range blocks {
		for i := 1; i < 64; i++ {
			coeff := block.Coefficients[i]
			if coeff != 0 {
				// Extract LSB
				bit := byte(coeff & 1)
				bits = append(bits, bit)
			}
		}
	}

	return convertBitsToBytes(bits)
}

// extractGenericLSB attempts a generic LSB extraction from coefficients
func extractGenericLSB(blocks []DCTCoefficientBlock) []byte {
	var bits []byte

	// Try different coefficient selections

	// 1. All non-zero coefficients
	for _, block := range blocks {
		for i := 0; i < 64; i++ {
			coeff := block.Coefficients[i]
			if coeff != 0 {
				bits = append(bits, byte(coeff&1))
			}
		}
	}

	result1 := convertBitsToBytes(bits)
	if IsASCIIPrintable(result1) {
		return result1
	}

	// 2. All coefficients
	bits = []byte{}
	for _, block := range blocks {
		for i := 0; i < 64; i++ {
			coeff := block.Coefficients[i]
			bits = append(bits, byte(coeff&1))
		}
	}

	result2 := convertBitsToBytes(bits)
	if IsASCIIPrintable(result2) {
		return result2
	}

	// 3. Only AC coefficients (no DC)
	bits = []byte{}
	for _, block := range blocks {
		for i := 1; i < 64; i++ { // Skip DC coefficient (index 0)
			coeff := block.Coefficients[i]
			bits = append(bits, byte(coeff&1))
		}
	}

	result3 := convertBitsToBytes(bits)
	if IsASCIIPrintable(result3) {
		return result3
	}

	// Return the longest result
	candidates := [][]byte{result1, result2, result3}
	longest := candidates[0]
	for _, candidate := range candidates[1:] {
		if len(candidate) > len(longest) {
			longest = candidate
		}
	}

	return longest
}

// convertBitsToBytes converts a slice of bits (0/1) to bytes
func convertBitsToBytes(bits []byte) []byte {
	var result []byte

	// Need at least 8 bits to form a byte
	if len(bits) < 8 {
		return result
	}

	// Process 8 bits at a time
	for i := 0; i < len(bits)/8; i++ {
		var b byte
		for j := 0; j < 8; j++ {
			b = (b << 1) | bits[i*8+j]
		}
		result = append(result, b)

		// Stop at null byte or after reasonable length
		if b == 0 && len(result) > 10 {
			break
		}
	}

	return result
}

// GetStegoCoefficientCount estimates how many coefficients were modified for steganography
func GetStegoCoefficientCount(blocks []DCTCoefficientBlock) int {
	// A heuristic approach to estimate the number of modified coefficients
	// Most stego algorithms can't use more than ~50-60% of available coefficients

	// Get statistics on zero values, which often indicate available capacity
	zeroCount := 0
	totalCoeffs := 0

	for _, block := range blocks {
		for i := 1; i < 64; i++ { // Skip DC
			totalCoeffs++
			if block.Coefficients[i] == 0 {
				zeroCount++
			}
		}
	}

	// Estimate potentially modified coefficients
	nonZeroCoeffs := totalCoeffs - zeroCount
	potentiallyModified := int(float64(nonZeroCoeffs) * 0.8)

	// Constraints based on JPEG file size
	// Assumes ~1 bit per modified coefficient
	maxPayloadBytes := potentiallyModified / 8

	return maxPayloadBytes
}
