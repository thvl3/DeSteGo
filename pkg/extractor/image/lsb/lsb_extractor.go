package lsb

import (
	//"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"

	//"image/color"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"

	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"DeSteGo/pkg/extractor"
	"DeSteGo/pkg/models"
	//_ "golang.org/x/image/bmp"
	//_ "golang.org/x/image/tiff"
)

const (
	// Common file signatures/magic numbers
	pngSignature   = "\x89PNG"
	jpgSignature   = "\xff\xd8\xff"
	pdfSignature   = "%PDF"
	zipSignature   = "PK\x03\x04"
	gifSignature   = "GIF8"
	bmpSignature   = "BM"
	nullTerminator = "\x00"
)

// LSBExtractor implements the ImageExtractor interface for LSB steganography
type LSBExtractor struct {
	extractor.BaseExtractor
}

// MaxExtractSize is the maximum size of data to extract (prevent excessive memory usage)
const MaxExtractSize = 50 * 1024 * 1024 // 50MB

// NewLSBExtractor creates a new LSB extractor
func NewLSBExtractor() *LSBExtractor {
	formats := []string{"png", "bmp", "tiff", "jpg", "jpeg", "gif"}
	algorithms := []string{"lsb-basic", "lsb-sequential", "lsb-rgb"}
	base := extractor.NewBaseExtractor("LSB Extractor", formats, algorithms)

	return &LSBExtractor{
		BaseExtractor: base,
	}
}

// Extract implements the DataExtractor interface
func (e *LSBExtractor) Extract(filePath string, options extractor.ExtractionOptions) (*models.ExtractionResult, error) {
	// Open the image file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Call the image-specific extraction method
	return e.ExtractFromImage(img, options)
}

// ExtractFromImage implements the ImageExtractor interface
func (e *LSBExtractor) ExtractFromImage(img image.Image, options extractor.ExtractionOptions) (*models.ExtractionResult, error) {
	if img == nil {
		return nil, errors.New("nil image provided")
	}

	// Try multiple extraction techniques and return the best result
	results := make(map[string]*ExtractionCandidate)
	var bestResult *ExtractionCandidate

	// Try different extraction methods
	extractionMethods := []struct {
		name   string
		method func(image.Image) *ExtractionCandidate
	}{
		{"sequential-rgb", extractSequentialRGB},
		{"sequential-rgba", extractSequentialRGBA},
		{"sequential-r", extractSequentialR},
		{"sequential-g", extractSequentialG},
		{"sequential-b", extractSequentialB},
		{"planes-rgb", extractPlanesRGB},
	}

	verbose := options.Verbose

	for _, method := range extractionMethods {
		if verbose {
			fmt.Printf("Trying extraction method: %s\n", method.name)
		}

		candidate := method.method(img)
		results[method.name] = candidate

		// Evaluate if this is the best result so far
		if bestResult == nil || candidate.Score > bestResult.Score {
			bestResult = candidate
		}
	}

	if bestResult == nil || bestResult.Data == nil || len(bestResult.Data) == 0 {
		return nil, errors.New("failed to extract any hidden data")
	}

	// Process extracted data to determine file type and save output
	return processExtractedData(bestResult, options)
}

// ExtractionCandidate represents a possible extraction result with quality metrics
type ExtractionCandidate struct {
	Data        []byte
	Method      string
	Score       float64
	FileType    string
	TextQuality float64
}

// extractSequentialRGB extracts LSB data sequentially from R, G, B channels
func extractSequentialRGB(img image.Image) *ExtractionCandidate {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Maximum possible size of hidden data (1 bit per channel per pixel)
	maxBytes := (width * height * 3) / 8
	if maxBytes > MaxExtractSize {
		maxBytes = MaxExtractSize
	}

	// Pre-allocate result buffer
	result := make([]byte, 0, maxBytes)

	var currentByte byte = 0
	bitIndex := 0

	// Extract LSBs sequentially from each RGB channel
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			// Extract 1 bit from each channel
			pixels := []uint32{r, g, b}
			for _, p := range pixels {
				// Get LSB from the current channel
				bit := byte(p>>8) & 1

				// Add bit to current byte
				currentByte |= bit << uint(7-bitIndex)
				bitIndex++

				// When we have a complete byte, add it to the result
				if bitIndex == 8 {
					result = append(result, currentByte)
					currentByte = 0
					bitIndex = 0

					// Check if we've reached the maximum size
					if len(result) >= maxBytes {
						score := evaluateExtraction(result)
						return &ExtractionCandidate{
							Data:   result,
							Method: "sequential-rgb",
							Score:  score,
						}
					}
				}
			}
		}
	}

	// Add the final partial byte if there is one
	if bitIndex > 0 {
		result = append(result, currentByte)
	}

	score := evaluateExtraction(result)
	return &ExtractionCandidate{
		Data:   result,
		Method: "sequential-rgb",
		Score:  score,
	}
}

// extractSequentialRGBA extracts LSB data sequentially from R, G, B, A channels
func extractSequentialRGBA(img image.Image) *ExtractionCandidate {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Pre-allocate result buffer
	maxBytes := (width * height * 4) / 8
	if maxBytes > MaxExtractSize {
		maxBytes = MaxExtractSize
	}
	result := make([]byte, 0, maxBytes)

	var currentByte byte = 0
	bitIndex := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			// Extract 1 bit from each channel, including alpha
			pixels := []uint32{r, g, b, a}
			for _, p := range pixels {
				bit := byte(p>>8) & 1
				currentByte |= bit << uint(7-bitIndex)
				bitIndex++

				if bitIndex == 8 {
					result = append(result, currentByte)
					currentByte = 0
					bitIndex = 0

					if len(result) >= maxBytes {
						score := evaluateExtraction(result)
						return &ExtractionCandidate{
							Data:   result,
							Method: "sequential-rgba",
							Score:  score,
						}
					}
				}
			}
		}
	}

	if bitIndex > 0 {
		result = append(result, currentByte)
	}

	score := evaluateExtraction(result)
	return &ExtractionCandidate{
		Data:   result,
		Method: "sequential-rgba",
		Score:  score,
	}
}

// extractSequentialR extracts LSB data from the R channel only
func extractSequentialR(img image.Image) *ExtractionCandidate {
	return extractSingleChannel(img, 0, "sequential-r")
}

// extractSequentialG extracts LSB data from the G channel only
func extractSequentialG(img image.Image) *ExtractionCandidate {
	return extractSingleChannel(img, 1, "sequential-g")
}

// extractSequentialB extracts LSB data from the B channel only
func extractSequentialB(img image.Image) *ExtractionCandidate {
	return extractSingleChannel(img, 2, "sequential-b")
}

// extractSingleChannel extracts LSB data from a single color channel
func extractSingleChannel(img image.Image, channel int, methodName string) *ExtractionCandidate {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Pre-allocate result buffer
	maxBytes := (width * height) / 8
	if maxBytes > MaxExtractSize {
		maxBytes = MaxExtractSize
	}
	result := make([]byte, 0, maxBytes)

	var currentByte byte = 0
	bitIndex := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			// Select the right channel
			var p uint32
			switch channel {
			case 0:
				p = r
			case 1:
				p = g
			case 2:
				p = b
			}

			// Extract LSB
			bit := byte(p>>8) & 1
			currentByte |= bit << uint(7-bitIndex)
			bitIndex++

			if bitIndex == 8 {
				result = append(result, currentByte)
				currentByte = 0
				bitIndex = 0

				if len(result) >= maxBytes {
					score := evaluateExtraction(result)
					return &ExtractionCandidate{
						Data:   result,
						Method: methodName,
						Score:  score,
					}
				}
			}
		}
	}

	if bitIndex > 0 {
		result = append(result, currentByte)
	}

	score := evaluateExtraction(result)
	return &ExtractionCandidate{
		Data:   result,
		Method: methodName,
		Score:  score,
	}
}

// extractPlanesRGB extracts LSB data by collecting all bits from R channel first,
// then G channel, then B channel
func extractPlanesRGB(img image.Image) *ExtractionCandidate {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	pixelCount := width * height

	// Pre-allocate bit arrays for each channel
	rBits := make([]byte, pixelCount)
	gBits := make([]byte, pixelCount)
	bBits := make([]byte, pixelCount)

	// Extract LSBs from each channel
	i := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			rBits[i] = byte(r>>8) & 1
			gBits[i] = byte(g>>8) & 1
			bBits[i] = byte(b>>8) & 1

			i++
		}
	}

	// Calculate number of whole bytes we can extract
	byteCount := pixelCount / 8
	if byteCount > MaxExtractSize {
		byteCount = MaxExtractSize
	}

	// Combine bits into bytes
	result := make([]byte, 0, byteCount*3) // R + G + B channels

	// Process each channel separately
	channels := [][]byte{rBits, gBits, bBits}
	for _, channel := range channels {
		var currentByte byte = 0

		for i := 0; i < pixelCount; i++ {
			bitIndex := i % 8
			currentByte |= channel[i] << uint(7-bitIndex)

			if bitIndex == 7 {
				result = append(result, currentByte)
				currentByte = 0

				if len(result) >= byteCount*3 {
					break
				}
			}
		}
	}

	score := evaluateExtraction(result)
	return &ExtractionCandidate{
		Data:   result,
		Method: "planes-rgb",
		Score:  score,
	}
}

// evaluateExtraction scores the quality of extracted data
func evaluateExtraction(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	score := 0.0

	// Check for known file signatures
	if detectFileSignature(data) != "" {
		score += 0.5 // Strong indicator of successful extraction
	}

	// Check text quality if it might be text data
	textScore := evaluateAsText(data)
	score += textScore * 0.3

	// Check entropy - good steganography data often has high entropy
	entropy := calculateDataEntropy(data)

	// Adjust score based on entropy
	// Too low entropy might be just zeros, too high might be random noise
	if entropy > 3.5 && entropy < 7.5 {
		score += 0.2
	}

	// Check if the data contains long sequences of the same byte
	// Natural files rarely have long sequences of identical bytes
	repetitionPenalty := calculateRepetitionPenalty(data)
	score -= repetitionPenalty

	return score
}

// detectFileSignature checks if the data starts with a known file signature
func detectFileSignature(data []byte) string {
	if len(data) < 8 {
		return ""
	}

	prefix := string(data[:8])

	// Check for common file signatures
	if strings.HasPrefix(prefix, pngSignature) {
		return "png"
	} else if strings.HasPrefix(prefix, jpgSignature) {
		return "jpg"
	} else if strings.HasPrefix(prefix, pdfSignature) {
		return "pdf"
	} else if strings.HasPrefix(prefix, zipSignature) {
		return "zip"
	} else if strings.HasPrefix(prefix, gifSignature) {
		return "gif"
	} else if strings.HasPrefix(prefix, bmpSignature) {
		return "bmp"
	}

	return ""
}

// evaluateAsText determines if the data is likely to be text
func evaluateAsText(data []byte) float64 {
	// Skip evaluation if too short
	if len(data) < 10 {
		return 0.0
	}

	// Check if the data is valid UTF-8
	if !utf8.Valid(data) {
		return 0.0
	}

	// Count printable ASCII characters
	printable := 0
	control := 0
	total := len(data)

	for _, b := range data {
		if b >= 32 && b <= 126 {
			printable++
		} else if b < 32 || b == 127 {
			// Control characters (except common ones like newline, tab)
			if b != 9 && b != 10 && b != 13 {
				control++
			}
		}
	}

	// Calculate percentage of printable chars
	printableRatio := float64(printable) / float64(total)
	controlRatio := float64(control) / float64(total)

	// Text typically has high printable ratio and low control char ratio
	textScore := printableRatio - (controlRatio * 2)

	if textScore < 0 {
		return 0.0
	} else if textScore > 1.0 {
		return 1.0
	}

	return textScore
}

// calculateDataEntropy calculates Shannon entropy of the data
func calculateDataEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	// Count occurrences of each byte value
	counts := make(map[byte]int, 256)
	for _, b := range data {
		counts[b]++
	}

	// Calculate entropy
	entropy := 0.0
	for _, count := range counts {
		probability := float64(count) / float64(len(data))
		entropy -= probability * (logBase2(probability))
	}

	return entropy
}

// logBase2 calculates log base 2 of a number
func logBase2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return float64(binary.Size(nil))
}

// calculateRepetitionPenalty detects unnatural byte repetitions
func calculateRepetitionPenalty(data []byte) float64 {
	if len(data) < 20 {
		return 0.0
	}

	// Check for long sequences of identical bytes
	maxRepeatLength := 0
	currentRepeat := 1

	for i := 1; i < len(data); i++ {
		if data[i] == data[i-1] {
			currentRepeat++
		} else {
			if currentRepeat > maxRepeatLength {
				maxRepeatLength = currentRepeat
			}
			currentRepeat = 1
		}
	}

	// Update max if the last sequence was the longest
	if currentRepeat > maxRepeatLength {
		maxRepeatLength = currentRepeat
	}

	// Apply penalty for very long repetitions
	if maxRepeatLength > 20 {
		return 0.3
	} else if maxRepeatLength > 10 {
		return 0.1
	}

	return 0.0
}

// processExtractedData analyzes the extracted data and saves it appropriately
func processExtractedData(candidate *ExtractionCandidate, options extractor.ExtractionOptions) (*models.ExtractionResult, error) {
	data := candidate.Data
	if data == nil || len(data) == 0 {
		return nil, errors.New("no data extracted")
	}

	// Try to detect the file type
	fileType := detectFileSignature(data)

	// Determine appropriate file extension
	extension := "bin"
	mimeType := "application/octet-stream"

	if fileType != "" {
		extension = fileType
		switch fileType {
		case "png":
			mimeType = "image/png"
		case "jpg":
			mimeType = "image/jpeg"
		case "pdf":
			mimeType = "application/pdf"
		case "zip":
			mimeType = "application/zip"
		case "gif":
			mimeType = "image/gif"
		case "bmp":
			mimeType = "image/bmp"
		}
	} else if evaluateAsText(data) > 0.7 {
		// Likely text data
		extension = "txt"
		mimeType = "text/plain"
	}

	// Create output filename
	filename := fmt.Sprintf("extracted_%s.%s", candidate.Method, extension)
	outputPath := filepath.Join(options.OutputDir, filename)

	// Write the extracted data to a file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write extracted data: %w", err)
	}

	// Create the extraction result
	result := &models.ExtractionResult{
		Algorithm:     "lsb-" + candidate.Method,
		Success:       true,
		FileType:      fileType,
		ExtractedData: data,
		DataSize:      len(data),
		Details: map[string]interface{}{
			"extraction_method": candidate.Method,
			"text_quality":      evaluateAsText(data),
			"entropy":           calculateDataEntropy(data),
		},
		OutputFiles: []string{outputPath},
		MimeType:    mimeType,
		DataType:    "binary",
	}

	return result, nil
}
