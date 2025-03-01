package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
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

// scanDirectorySequential processes PNG files one at a time
func scanDirectorySequential(dirPath string) {
	// Gather .png files
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".png" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Printf("[!] Error walking directory %s: %v\n", dirPath, err)
		return
	}

	fmt.Printf("Found %d PNG files to scan\n", len(files))

	// Process files sequentially
	for _, f := range files {
		scanFile(f)
	}
}

// scanDirectoryConcurrent processes PNG files in parallel (original implementation)
func scanDirectoryConcurrent(dirPath string) {
	// Gather .png files
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".png" {
			files = append(files, path)
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

	img, err := LoadPNG(filename)
	if err != nil {
		logger.Printf("[-] Failed to open/parse PNG: %v\n", err)
		return
	}

	// Run multiple detection methods
	runChiSquareAnalysis(img, logger)
	runLSBAnalysis(img, logger)
	runJSLSBDetection(img, logger)
}

// scanFile performs all detection steps on a single file with direct output
func scanFile(filename string) {
	fmt.Printf("\n--- Scanning %s ---\n", filename)

	img, err := LoadPNG(filename)
	if err != nil {
		fmt.Printf("[-] Failed to open/parse PNG: %v\n", err)
		return
	}

	// Run multiple detection methods
	runChiSquareAnalysis(img, nil)
	runLSBAnalysis(img, nil)
	runJSLSBDetection(img, nil)
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
		tryExtractChannelMask(img, mask, true, output)
		tryExtractChannelMask(img, mask, false, output)
	}

	// 2. Then do a more comprehensive brute force if needed
	results := BruteForceLSB(img)
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
func tryExtractChannelMask(img image.Image, mask ChannelMask, useLength bool, output func(string, ...interface{})) {
	var data []byte
	var err error

	if useLength {
		// Try extracting with length prefix
		data, err = ExtractData(img, mask, LSBFirst)
	} else {
		// Try extracting without length prefix
		data = ExtractBitsDirectly(img, mask)
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
			output("[+] Found hidden ASCII text using %s extraction (R:%d G:%d B:%d A:%d):\n%q\n",
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
