package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"

	//"image/jpeg" // Add JPEG support
	_ "image/jpeg"
	//"image/png"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Worker pool size for scanning files concurrently
const workerCount = 4

// Logger is a custom logger that can buffer output
type Logger struct {
	buf    bytes.Buffer
	prefix string
	mu     sync.Mutex
}

// NewLogger creates a new buffered logger
func NewLogger(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

// Printf writes a formatted message to the buffer
func (l *Logger) Printf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(&l.buf, format, args...)
}

// FlushTo outputs the buffered content to a writer
func (l *Logger) FlushTo(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	w.Write(l.buf.Bytes())
	l.buf.Reset()
}

// main orchestrates everything from the command line.
//
// Usage examples:
//
//	go run testmain.go -dir ./images
//	go run testmain.go -file sample.png
func main() {
	dirPtr := flag.String("dir", "", "Directory of PNG files to scan")
	filePtr := flag.String("file", "", "Single PNG file to scan")
	sequentialPtr := flag.Bool("seq", true, "Use sequential processing (default: true)")
	flag.Parse()

	if *dirPtr == "" && *filePtr == "" {
		fmt.Println("Usage:")
		fmt.Println("  testmain -dir <directory> [-seq=false]")
		fmt.Println("  testmain -file <pngfile>")
		os.Exit(1)
	}

	if *dirPtr != "" {
		if *sequentialPtr {
			scanDirectorySequential(*dirPtr)
		} else {
			scanDirectoryConcurrent(*dirPtr)
		}
	} else {
		// Single-file scan
		scanFile(*filePtr)
	}
}

// scanDirectorySequential processes image files one at a time
func scanDirectorySequential(dirPath string) {
	// Gather .png and .jpg files
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("[!] Error walking directory %s: %v\n", dirPath, err)
		return
	}

	fmt.Printf("Found %d image files to scan\n", len(files))

	// Process files sequentially
	for _, f := range files {
		scanFile(f)
	}
}

// scanDirectoryConcurrent processes image files in parallel
func scanDirectoryConcurrent(dirPath string) {
	// Gather .png and .jpg files
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("[!] Error walking directory %s: %v\n", dirPath, err)
		return
	}

	// Set up a worker pool
	fileChan := make(chan string)
	resultChan := make(chan *Logger)
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range fileChan {
				logger := NewLogger(f)
				scanFileBuffered(f, logger)
				resultChan <- logger
			}
		}()
	}

	// Start a goroutine to collect and print results
	go func() {
		for logger := range resultChan {
			logger.FlushTo(os.Stdout)
		}
	}()

	// Feed the files to the workers
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)
}

// scanFileBuffered performs all detection steps on a single file and logs to a buffer
func scanFileBuffered(filename string, logger *Logger) {
	logger.Printf("\n--- Scanning %s ---\n", filename)

	img, err := LoadImage(filename)
	if err != nil {
		logger.Printf("[-] Failed to open/parse image: %v\n", err)
		return
	}

	format, _ := GetImageFormat(filename)
	logger.Printf("Image format: %s\n", format)

	// Run appropriate detection methods based on format
	if format == "png" {
		runChiSquareAnalysis(img, logger)
		runLSBAnalysis(img, logger)
		runJSLSBDetection(img, logger)
	} else if format == "jpeg" {
		runJPEGAnalysis(img, filename, logger)
	}
}

// scanFile performs all detection steps on a single file with direct output
func scanFile(filename string) {
	fmt.Printf("\n--- Scanning %s ---\n", filename)

	img, err := LoadImage(filename)
	if err != nil {
		fmt.Printf("[-] Failed to open/parse image: %v\n", err)
		return
	}

	format, _ := GetImageFormat(filename)
	fmt.Printf("Image format: %s\n", format)

	// Run appropriate detection methods based on format
	if format == "png" {
		runChiSquareAnalysis(img, nil)
		runLSBAnalysis(img, nil)
		runJSLSBDetection(img, nil)
	} else if format == "jpeg" {
		runJPEGAnalysis(img, filename, nil)
	}
}

// runChiSquareAnalysis runs Chi-Square tests on the image
func runChiSquareAnalysis(img image.Image, logger *Logger) {
	output := func(format string, args ...interface{}) {
		if logger != nil {
			logger.Printf(format, args...)
		} else {
			fmt.Printf(format, args...)
		}
	}

	//
	// 1) Run a Chi-Square analysis on R, G, B channels separately.
	//
	chiR := ChiSquareLSB(img, 'R')
	chiG := ChiSquareLSB(img, 'G')
	chiB := ChiSquareLSB(img, 'B')

	output("\n=== Chi-Square Analysis ===\n")

	// Better interpretation of chi-square values
	// Very low or very high values can both indicate steganography
	interpretChiSquare := func(chi float64, channel byte) {
		if chi < 0.5 {
			output("[!] Chi-square result (%c) = %.4f => Suspiciously uniform (likely steganography)\n", channel, chi)
		} else if chi > 10.0 {
			output("[!] Chi-square result (%c) = %.4f => Suspiciously non-uniform (possible structured steganography)\n", channel, chi)
		} else {
			output("[ ] Chi-square result (%c) = %.4f => Within normal range\n", channel, chi)
		}
	}

	interpretChiSquare(chiR, 'R')
	interpretChiSquare(chiG, 'G')
	interpretChiSquare(chiB, 'B')

	// Calculate the average chi-square across channels
	avgChi := (chiR + chiG + chiB) / 3.0
	if avgChi < 0.7 || avgChi > 8.0 {
		output("[!] Average Chi-square = %.4f => Suspicious LSB distributions detected\n", avgChi)
	}
}

// runLSBAnalysis performs traditional LSB extraction with multiple methods
func runLSBAnalysis(img image.Image, logger *Logger) {
	output := func(format string, args ...interface{}) {
		if logger != nil {
			logger.Printf(format, args...)
		} else {
			fmt.Printf(format, args...)
		}
	}

	output("\n=== LSB Brute Force Analysis ===\n")

	// Create a progress callback function
	progressCb := func(percentComplete float64, message string) {
		if logger != nil {
			logger.Printf("\r[%.1f%%] %s", percentComplete, message)
		} else {
			fmt.Printf("\r[%.1f%%] %s", percentComplete, message)
			// Ensure console flushes the output
			if message == "" || message[len(message)-1] == '\n' {
				fmt.Print("\n")
			}
		}
	}

	// 1. First try specific, targeted masks that are commonly used in steganography
	commonMasks := []ChannelMask{
		{RBits: 1, GBits: 0, BBits: 0, ABits: 0}, // R only
		{RBits: 0, GBits: 1, BBits: 0, ABits: 0}, // G only
		{RBits: 0, GBits: 0, BBits: 1, ABits: 0}, // B only
		{RBits: 1, GBits: 1, BBits: 1, ABits: 0}, // RGB equal
		{RBits: 1, GBits: 1, BBits: 0, ABits: 0}, // RG only
	}

	for _, mask := range commonMasks {
		// Try both with and without length prefix
		tryExtractChannelMask(img, mask, true, progressCb, output)
		tryExtractChannelMask(img, mask, false, progressCb, output)
	}

	output("\n") // Ensure we have a clean line after progress updates

	// 2. Then do a more comprehensive brute force
	output("Starting comprehensive LSB brute force scan...\n")
	results := BruteForceLSB(img, progressCb)
	output("\n") // Ensure clean line after progress

	if len(results) > 0 {
		output("[+] Found %d potential embedded payload(s) via LSB brute force:\n", len(results))
		for _, r := range results {
			if IsASCIIPrintable(r.Data) {
				output("   Mask R=%d G=%d B=%d => ASCII text:\n   %q\n",
					r.Mask.RBits, r.Mask.GBits, r.Mask.BBits, string(r.Data))
			} else {
				entropy := ComputeEntropy(r.Data)
				output("   Mask R=%d G=%d B=%d => Non-printable data (%d bytes), entropy: %.4f\n",
					r.Mask.RBits, r.Mask.GBits, r.Mask.BBits, len(r.Data), entropy)

				// Check for encrypted/compressed data signatures
				if entropy > 7.8 {
					output("   [!] High entropy suggests encrypted or compressed data\n")
				} else if entropy < 6.0 {
					output("   [!] Medium entropy may indicate encoded data\n")
				}
			}
		}
	} else {
		output("[ ] No embedded data found with traditional LSB methods\n")
	}
}

// tryExtractChannelMask tries to extract data using a specific channel mask
func tryExtractChannelMask(img image.Image, mask ChannelMask, useLength bool,
	progressCb ProgressCallback, output func(string, ...interface{})) {

	var data []byte
	var err error

	if useLength {
		// Try extracting with length prefix
		data, err = ExtractData(img, mask, LSBFirst, progressCb)
	} else {
		// Try extracting without length prefix
		data = ExtractLSBNoLength(img, mask, LSBFirst, progressCb)
		if len(data) == 0 {
			err = fmt.Errorf("no data extracted")
		}
	}

	if err == nil && len(data) > 0 {
		if IsASCIIPrintable(data) {
			extractionType := "direct"
			if useLength {
				extractionType = "length-based"
			}
			output("\n[+] Found hidden ASCII text using %s extraction (R:%d G:%d B:%d A:%d):\n%q\n",
				extractionType,
				mask.RBits, mask.GBits, mask.BBits, mask.ABits,
				string(data))
		}
	}
}

// runJSLSBDetection runs specialized detection for JavaScript LSB embedding
func runJSLSBDetection(img image.Image, logger *Logger) {
	output := func(format string, args ...interface{}) {
		if logger != nil {
			logger.Printf(format, args...)
		} else {
			fmt.Printf(format, args...)
		}
	}

	output("\n=== JavaScript LSB Detection ===\n")

	// Check if the LSB distribution suggests JS LSB encoding
	if DetectJSLSB(img) {
		output("[!] LSB distribution suggests JavaScript steganography\n")

		// Try to extract the message
		message, err := TryExtractJSLSB(img)
		if err == nil && len(message) > 0 {
			output("[+] Extracted potential JavaScript LSB message:\n%s\n", message)
		} else {
			output("[-] Detected JavaScript LSB pattern but could not extract a message\n")
		}
	} else {
		output("[ ] No JavaScript LSB pattern detected\n")
	}

	// Additional forensic analysis
	dist := AnalyzeLSBDistribution(img)
	output("\n=== LSB Distribution Analysis ===\n")
	output("Total entropy: %.4f (0=uniform, 1=random)\n", dist.Entropy)
	output("Channel entropies: R=%.4f, G=%.4f, B=%.4f, A=%.4f\n",
		dist.ChannelStats["R"].Entropy,
		dist.ChannelStats["G"].Entropy,
		dist.ChannelStats["B"].Entropy,
		dist.ChannelStats["A"].Entropy)

	// Final determination
	if dist.Entropy > 0.95 || dist.Entropy < 0.6 {
		output("[!] LSB entropy analysis suggests hidden data (entropy=%.4f)\n", dist.Entropy)
	}
}

// runJPEGAnalysis performs JPEG-specific steganalysis
func runJPEGAnalysis(img image.Image, filename string, logger *Logger) {
	output := func(format string, args ...interface{}) {
		if logger != nil {
			logger.Printf(format, args...)
		} else {
			fmt.Printf(format, args...)
		}
	}

	output("\n=== JPEG Analysis ===\n")

	// 1. Analyze JPEG metadata
	jfifData, err := ExtractJPEGMetadata(filename)
	if err != nil {
		output("[-] Failed to extract JPEG metadata: %v\n", err)
		return
	}

	// 2. Check for signs of steganography
	if isJPEGModified := DetectJPEGSteganography(jfifData); isJPEGModified {
		output("[!] JPEG analysis suggests possible steganography\n")

		// Check for specific steganography tools
		if DetectJSteg(jfifData) {
			output("[!] Detected possible JSteg steganography\n")
		}

		if DetectF5(jfifData) {
			output("[!] Detected possible F5 steganography\n")
		}

		if DetectOutguess(jfifData) {
			output("[!] Detected possible Outguess steganography\n")
		}
	} else {
		output("[ ] No obvious signs of JPEG steganography detected in file structure\n")
	}

	// 2.5 NEW: Specific Steghide detection
	output("\n=== StegHide Detection ===\n")
	isStegHide, stegStats, err := DetectStegHide(filename)
	if err != nil {
		output("[-] Error during StegHide detection: %v\n", err)
	} else if isStegHide {
		output("[!] StegHide steganography detected (confidence: %d/10)\n", stegStats.ConfidenceScore)
		output("    - Modified coefficients: %.1f%%\n", stegStats.ModifiedCoefficients*100)
		output("    - Even/Odd coefficient ratio: %.2f (normal ~1.0)\n", stegStats.EvenOddRatio)
		output("    - StegHide header detected: %v\n", stegStats.PotentialHeader)

		// Try to extract information about the payload
		payloadInfo, err := ExtractPotentialStegHidePayload(filename)
		if err == nil && len(payloadInfo) > 0 {
			output("[+] Payload information:\n%s\n", string(payloadInfo))
		}
	} else {
		if stegStats.ConfidenceScore > 0 {
			output("[-] Some StegHide indicators found, but below detection threshold (confidence: %d/10)\n",
				stegStats.ConfidenceScore)
		} else {
			output("[ ] No evidence of StegHide steganography\n")
		}
	}

	// 3. Look for plaintext steganography
	output("\n=== JPEG Plaintext Search ===\n")
	plaintextFindings, err := ScanForPlaintextStego(filename)
	if err != nil {
		output("[-] Error scanning for plaintext: %v\n", err)
	} else if len(plaintextFindings) > 0 {
		output("[!] Found %d potential plaintext message(s) in unexpected locations:\n", len(plaintextFindings))

		for i, text := range plaintextFindings {
			// Add a score to each finding based on how confident we are
			confidence := assessTextConfidence(text)

			// Limit output length to avoid flooding the console
			displayText := text
			if len(displayText) > 100 {
				displayText = displayText[:97] + "..."
			}

			output("    [%d] (Confidence: %d/10) %s\n", i+1, confidence, displayText)
		}
	} else {
		output("[ ] No suspicious plaintext found in JPEG structure\n")
	}

	// 4. Check for polyglot files (JPEG combined with another format)
	isPolyglot, otherFormat := ScanForPolyglotFile(filename)
	if isPolyglot {
		output("[!] File appears to be a polyglot - both JPEG and %s format\n", otherFormat)
		output("    Polyglot files are often used to hide data\n")
	}

	// 5. Other JPEG-specific checks
	output("\n=== JPEG File Structure Analysis ===\n")

	// Check for appended data
	if hasAppendedData, dataSize := CheckAppendedData(filename); hasAppendedData {
		output("[!] Found %d bytes of data appended after JPEG EOI marker\n", dataSize)

		// Try to extract and analyze the appended data
		appendedData, err := ExtractAppendedData(filename)
		if err == nil && len(appendedData) > 0 {
			if IsASCIIPrintable(appendedData) {
				output("[+] Appended data appears to be ASCII text:\n%s\n",
					string(appendedData[:min(100, len(appendedData))]))
				if len(appendedData) > 100 {
					output("... (%d more bytes)\n", len(appendedData)-100)
				}
			} else {
				entropy := ComputeEntropy(appendedData)
				output("[+] Appended data is binary (%d bytes, entropy: %.2f)\n",
					len(appendedData), entropy)

				// Check for encoded text in the binary data
				if containsEncodedBytes(appendedData) {
					output("[!] Appended data appears to contain encoded text (possible base64/hex)\n")
				}
			}
		}
	} else {
		output("[ ] No data appended after JPEG EOI marker\n")
	}

	// Check for mismatched quantization tables
	if hasModifiedTables, tableCount := CheckQuantizationTables(jfifData); hasModifiedTables {
		output("[!] Detected non-standard quantization tables (%d tables)\n", tableCount)
		output("    This may indicate steganographic manipulation\n")
	} else {
		output("[ ] Quantization tables appear standard\n")
	}

	// Check comment sections
	if len(jfifData.Comments) > 0 {
		output("\n=== JPEG Comments Analysis ===\n")

		for i, comment := range jfifData.Comments {
			if len(comment) > 100 {
				output("[!] Comment %d: Length=%d (suspicious if >100): %.100s...\n",
					i+1, len(comment), comment)
			} else {
				output("[+] Comment %d: %s\n", i+1, comment)
			}
		}
	}
}

// assessTextConfidence rates how likely a string is to be an intentional hidden message
func assessTextConfidence(text string) int {
	// Start with baseline score
	score := 5

	// Longer text is more likely to be meaningful
	if len(text) > 20 {
		score++
	}

	// Text with spaces is more likely to be natural language
	if strings.Contains(text, " ") {
		score++
	}

	// Text with normal punctuation is more likely to be meaningful
	if strings.ContainsAny(text, ".,:;?!") {
		score++
	}

	// Check for keywords that suggest a hidden message
	if strings.Contains(strings.ToLower(text), "secret") ||
		strings.Contains(strings.ToLower(text), "password") ||
		strings.Contains(strings.ToLower(text), "key") ||
		strings.Contains(strings.ToLower(text), "confidential") {
		score += 2
	}

	// Check for high numbers of special characters/digits (suspicious)
	specCount := 0
	for _, c := range text {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == ' ') {
			specCount++
		}
	}

	if float64(specCount)/float64(len(text)) > 0.4 {
		score -= 2
	}

	// Clamp score between 1-10
	if score < 1 {
		score = 1
	}
	if score > 10 {
		score = 10
	}

	return score
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ExtractBitsDirectly extracts bits directly from the image without assuming a length prefix
// This is now exported for use in multiple places
func ExtractBitsDirectly(img image.Image, mask ChannelMask) []byte {
	bounds := img.Bounds()
	//var bits []byte
	bytesBuilder := bytes.Buffer{}
	currentByte := byte(0)
	bitCount := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			// Extract LSB from each channel according to mask
			if mask.RBits > 0 {
				currentByte = (currentByte << 1) | byte(r>>8)&1
				bitCount++
			}
			if mask.GBits > 0 {
				currentByte = (currentByte << 1) | byte(g>>8)&1
				bitCount++
			}
			if mask.BBits > 0 {
				currentByte = (currentByte << 1) | byte(b>>8)&1
				bitCount++
			}
			if mask.ABits > 0 {
				currentByte = (currentByte << 1) | byte(a>>8)&1
				bitCount++
			}

			// When we have 8 bits, add the byte to our result
			if bitCount >= 8 {
				// If the byte contains all nulls, we might be at the end
				bytesBuilder.WriteByte(currentByte)

				// Reset for next byte
				currentByte = 0
				bitCount = 0
			}
		}
	}

	return bytesBuilder.Bytes()
}
