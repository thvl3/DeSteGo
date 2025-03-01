package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// StegoExtractionResult holds the result of an extraction attempt
type StegoExtractionResult struct {
	Algorithm       string
	Data            []byte
	ConfidenceScore int
	PayloadSize     int
	Description     string
	IsASCII         bool
	Entropy         float64
}

// ExtractJPEGSteganography attempts to extract steganographic content from a JPEG file
// using various methods and algorithms
func ExtractJPEGSteganography(filename string) ([]StegoExtractionResult, error) {
	var results []StegoExtractionResult

	// 1. First try to extract using DCT coefficient analysis
	dctResults, err := extractUsingDCTAnalysis(filename)
	if err == nil {
		results = append(results, dctResults...)
	}

	// 2. Try using the dedicated StegHide detector
	stegHideResult, err := extractUsingStegHide(filename)
	if err == nil && len(stegHideResult.Data) > 0 {
		results = append(results, stegHideResult)
	}

	// 3. Check for appended data
	appendedDataResult, err := extractAppendedData(filename)
	if err == nil && len(appendedDataResult.Data) > 0 {
		results = append(results, appendedDataResult)
	}

	// 4. Try using external tools if available
	externalResults, _ := extractUsingExternalTools(filename)
	results = append(results, externalResults...)

	return results, nil
}

// extractUsingDCTAnalysis extracts data using DCT coefficient analysis
func extractUsingDCTAnalysis(filename string) ([]StegoExtractionResult, error) {
	var results []StegoExtractionResult

	// Parse DCT coefficients from the JPEG file
	dctData, err := ParseJPEGDCTCoefficients(filename)
	if err != nil {
		return nil, err
	}

	// Detect which steganography algorithm might be used
	detected, metrics, algorithm := DetectStegoByCoefficientAnalysis(dctData)
	if !detected {
		return nil, fmt.Errorf("no steganography detected in DCT coefficients")
	}

	// Try to extract the data using the detected algorithm
	extractedData, err := ExtractSteganographicData(dctData, algorithm)
	if err != nil || len(extractedData) == 0 {
		return nil, fmt.Errorf("failed to extract data: %v", err)
	}

	// Create result
	result := StegoExtractionResult{
		Algorithm:   algorithm,
		Data:        extractedData,
		PayloadSize: len(extractedData),
		Description: fmt.Sprintf("Extracted using %s algorithm from DCT coefficients", algorithm),
		IsASCII:     IsASCIIPrintable(extractedData),
		Entropy:     ComputeEntropy(extractedData),
	}

	// Calculate confidence score based on various factors
	confidenceScore := 5 // Start with neutral confidence

	// Adjust based on detection metrics
	if metrics["suspiciousness"] > 0.8 {
		confidenceScore += 2
	}

	// Higher confidence if the data looks like valid content
	if result.IsASCII {
		confidenceScore += 2
	} else if result.Entropy > 7.0 {
		confidenceScore += 1 // High entropy could be encrypted data
	}

	// Payload size reasonability check
	maxPossiblePayload := GetStegoCoefficientCount(dctData.Blocks)
	if result.PayloadSize > maxPossiblePayload {
		confidenceScore -= 3 // Unlikely to be valid if too large
	}

	result.ConfidenceScore = confidenceScore
	if confidenceScore > 0 {
		results = append(results, result)
	}

	// Try alternative algorithms if the detected one didn't work well
	if confidenceScore < 6 && algorithm != "unknown" {
		alternativeAlgorithms := []string{
			"JSteg",
			"F5",
			"StegHide",
			"OutGuess",
		}

		for _, altAlgo := range alternativeAlgorithms {
			if altAlgo == algorithm {
				continue // Skip the one we already tried
			}

			altData, err := ExtractSteganographicData(dctData, altAlgo)
			if err == nil && len(altData) > 0 {
				altResult := StegoExtractionResult{
					Algorithm:       altAlgo,
					Data:            altData,
					PayloadSize:     len(altData),
					Description:     fmt.Sprintf("Alternative extraction using %s", altAlgo),
					IsASCII:         IsASCIIPrintable(altData),
					Entropy:         ComputeEntropy(altData),
					ConfidenceScore: 4, // Lower confidence for alternative algorithms
				}

				results = append(results, altResult)
			}
		}
	}

	return results, nil
}

// extractUsingStegHide tries to extract data using specialized StegHide detection
func extractUsingStegHide(filename string) (StegoExtractionResult, error) {
	result := StegoExtractionResult{
		Algorithm:   "StegHide",
		Description: "Extracted using StegHide algorithm",
	}

	// First check if this looks like StegHide
	isStegHide, stats, err := DetectStegHide(filename)
	if err != nil || !isStegHide {
		return result, fmt.Errorf("file doesn't appear to use StegHide: %v", err)
	}

	// Try to extract the payload
	data, err := ExtractPotentialStegHidePayload(filename)
	if err != nil {
		return result, err
	}

	result.Data = data
	result.PayloadSize = len(data)
	result.IsASCII = IsASCIIPrintable(data)
	result.Entropy = ComputeEntropy(data)
	result.ConfidenceScore = stats.ConfidenceScore

	return result, nil
}

// extractAppendedData creates a result from data appended after the JPEG EOF marker
func extractAppendedData(filename string) (StegoExtractionResult, error) {
	result := StegoExtractionResult{
		Algorithm:   "Appended",
		Description: "Data appended after JPEG EOI marker",
	}

	// Check if there's appended data
	hasAppended, size := CheckAppendedData(filename)
	if !hasAppended || size == 0 {
		return result, fmt.Errorf("no appended data found")
	}

	// Extract the appended data
	data, err := ExtractAppendedData(filename)
	if err != nil {
		return result, err
	}

	result.Data = data
	result.PayloadSize = len(data)
	result.IsASCII = IsASCIIPrintable(data)
	result.Entropy = ComputeEntropy(data)

	// Calculate confidence score based on data properties
	confidence := 5 // Start with neutral score

	// ASCII text is highly likely to be intentional
	if result.IsASCII {
		confidence += 3
	}

	// Very high entropy might indicate encrypted content
	if result.Entropy > 7.5 {
		confidence += 2
	}

	// Small payload size might be incidental data
	if result.PayloadSize < 10 {
		confidence -= 2
	}

	result.ConfidenceScore = confidence

	return result, nil
}

// extractUsingExternalTools attempts to use external steganography tools if available
func extractUsingExternalTools(filename string) ([]StegoExtractionResult, error) {
	var results []StegoExtractionResult

	// Check if the steghide command-line tool is available
	if hasStegHideCommand() {
		result, err := tryExtractUsingStegHideCommand(filename)
		if err == nil && len(result.Data) > 0 {
			results = append(results, result)
		}
	}

	// Check if the outguess command-line tool is available
	if hasOutguessCommand() {
		result, err := tryExtractUsingOutguessCommand(filename)
		if err == nil && len(result.Data) > 0 {
			results = append(results, result)
		}
	}

	// Add more external tools as needed...

	return results, nil
}

// hasStegHideCommand checks if the steghide command-line tool is available
func hasStegHideCommand() bool {
	cmd := exec.Command("which", "steghide")
	err := cmd.Run()
	return err == nil
}

// tryExtractUsingStegHideCommand attempts extraction using the steghide command
// This will only work with empty passwords or if we attempt common passwords
func tryExtractUsingStegHideCommand(filename string) (StegoExtractionResult, error) {
	result := StegoExtractionResult{
		Algorithm:   "StegHide-External",
		Description: "Extracted using steghide command-line tool",
	}

	// Create a temporary output file
	tmpDir := os.TempDir()
	outFile := filepath.Join(tmpDir, "steghide_extract.tmp")
	defer os.Remove(outFile) // Clean up when done

	// Try with empty password first
	cmd := exec.Command("steghide", "extract", "-sf", filename, "-xf", outFile, "-p", "", "-f")

	// Ignore stderr since it will error if password is wrong
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		// Try with some common passwords
		commonPasswords := []string{"password", "123456", "steg", "steghide", "secret"}

		for _, passwd := range commonPasswords {
			cmd := exec.Command("steghide", "extract", "-sf", filename, "-xf", outFile, "-p", passwd, "-f")
			cmd.Stderr = nil

			err = cmd.Run()
			if err == nil {
				// Successfully extracted
				break
			}
		}
	}

	// Check if we managed to extract anything
	if _, statErr := os.Stat(outFile); statErr == nil {
		data, readErr := os.ReadFile(outFile)
		if readErr == nil && len(data) > 0 {
			result.Data = data
			result.PayloadSize = len(data)
			result.IsASCII = IsASCIIPrintable(data)
			result.Entropy = ComputeEntropy(data)
			result.ConfidenceScore = 9 // Very high confidence since an actual tool extracted it

			return result, nil
		}
	}

	return result, fmt.Errorf("failed to extract data with steghide command")
}

// hasOutguessCommand checks if the outguess command-line tool is available
func hasOutguessCommand() bool {
	cmd := exec.Command("which", "outguess")
	err := cmd.Run()
	return err == nil
}

// tryExtractUsingOutguessCommand attempts extraction using the outguess command
func tryExtractUsingOutguessCommand(filename string) (StegoExtractionResult, error) {
	result := StegoExtractionResult{
		Algorithm:   "Outguess-External",
		Description: "Extracted using outguess command-line tool",
	}

	// Create a temporary output file
	tmpDir := os.TempDir()
	outFile := filepath.Join(tmpDir, "outguess_extract.tmp")
	defer os.Remove(outFile) // Clean up when done

	// Try extraction with no password
	cmd := exec.Command("outguess", "-r", filename, outFile)
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		// Try with some common passwords (if outguess supports passwords)
		commonPasswords := []string{"password", "123456", "outguess", "secret"}

		for _, passwd := range commonPasswords {
			cmd := exec.Command("outguess", "-k", passwd, "-r", filename, outFile)
			cmd.Stderr = nil

			err = cmd.Run()
			if err == nil {
				// Successfully extracted
				break
			}
		}
	}

	// Check if we managed to extract anything
	if _, statErr := os.Stat(outFile); statErr == nil {
		data, readErr := os.ReadFile(outFile)
		if readErr == nil && len(data) > 0 {
			result.Data = data
			result.PayloadSize = len(data)
			result.IsASCII = IsASCIIPrintable(data)
			result.Entropy = ComputeEntropy(data)
			result.ConfidenceScore = 9 // Very high confidence since an actual tool extracted it

			return result, nil
		}
	}

	return result, fmt.Errorf("failed to extract data with outguess command")
}

// analyzeExtractedData performs detailed analysis on extracted data to determine its nature
func analyzeExtractedData(data []byte) map[string]interface{} {
	results := make(map[string]interface{})

	// Check if it's printable ASCII
	isASCII := IsASCIIPrintable(data)
	results["is_ascii"] = isASCII

	// Calculate entropy
	entropy := ComputeEntropy(data)
	results["entropy"] = entropy

	// Determine likely data type
	dataType := "unknown"

	// Check for common file signatures
	if len(data) > 4 {
		// Check for PNG
		if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
			dataType = "png_image"
		}
		// Check for JPEG
		if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			dataType = "jpeg_image"
		}
		// Check for ZIP
		if data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04 {
			dataType = "zip_archive"
		}
		// Check for PDF
		if data[0] == '%' && data[1] == 'P' && data[2] == 'D' && data[3] == 'F' {
			dataType = "pdf_document"
		}
	}

	// If no specific signature was found but it's ASCII
	if dataType == "unknown" && isASCII {
		dataType = "text"

		// Try to determine text type
		text := string(data)

		// Check if it might be JSON
		if (text[0] == '{' && text[len(text)-1] == '}') ||
			(text[0] == '[' && text[len(text)-1] == ']') {
			dataType = "json_text"
		}

		// Check if it might be XML/HTML
		if text[0] == '<' && text[len(text)-1] == '>' {
			dataType = "xml_html_text"
		}
	}

	// If no specific type found, but high entropy
	if dataType == "unknown" && entropy > 7.5 {
		dataType = "encrypted_or_compressed"
	}

	results["data_type"] = dataType

	return results
}

// saveExtractedData saves the data to a file with an appropriate extension
func saveExtractedData(data []byte, algorithm string, outputDir string) (string, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	// Determine appropriate extension
	ext := ".bin" // Default

	// Check for common file signatures
	if len(data) > 4 {
		if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
			ext = ".png"
		} else if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			ext = ".jpg"
		} else if data[0] == 'P' && data[1] == 'K' {
			ext = ".zip"
		} else if data[0] == '%' && data[1] == 'P' && data[2] == 'D' && data[3] == 'F' {
			ext = ".pdf"
		} else if IsASCIIPrintable(data) {
			ext = ".txt"
		}
	}
	// Create filename
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	filename := filepath.Join(outputDir,
		fmt.Sprintf("extracted_%s_%s%s", algorithm, timestamp, ext))

	// Save the file
	err := os.WriteFile(filename, data, 0644)
	if err != nil {
		return "", err
	}

	return filename, nil
}
