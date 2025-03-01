package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Color constants
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

// Colored output helpers
func printInfo(format string, args ...interface{}) {
	fmt.Printf(colorBlue+"[*] "+format+colorReset, args...)
}

func printSuccess(format string, args ...interface{}) {
	fmt.Printf(colorGreen+"[+] "+format+colorReset, args...)
}

func printWarning(format string, args ...interface{}) {
	fmt.Printf(colorYellow+"[!] "+format+colorReset, args...)
}

func printError(format string, args ...interface{}) {
	fmt.Printf(colorRed+"[-] "+format+colorReset, args...)
}

func printAlert(format string, args ...interface{}) {
	fmt.Printf(colorRed+colorBold+"[!!!] "+format+colorReset, args...)
}

// moveFiles moves all PNG and JPG files from subdirectories to the target directory
func moveFiles(rootDir string) error {
	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if file is PNG or JPG
		if !info.IsDir() && (filepath.Ext(path) == ".png" || filepath.Ext(path) == ".jpg" || filepath.Ext(path) == ".jpeg") {
			destPath := filepath.Join(rootDir, filepath.Base(path))
			fmt.Printf("Moving: %s -> %s\n", path, destPath)
			err := os.Rename(path, destPath) // Move file
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// removeEmptyDirsRecursively keeps deleting empty folders until none remain
func removeEmptyDirsRecursively(rootDir string) error {
	removed := true
	for removed { // Keep looping until no more empty dirs are deleted
		removed = false
		filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				files, err := os.ReadDir(path)
				if err != nil {
					return err
				}
				if len(files) == 0 && path != rootDir {
					fmt.Printf("Removing empty folder: %s\n", path)
					os.Remove(path) // Delete empty directory
					removed = true  // Mark as removed so we check again
				}
			}
			return nil
		})
	}
	return nil
}

// GalleryDownload uses gallery-dl to download images from a URL
func GalleryDownload(url string, targetDir string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Run gallery-dl command
	printInfo("Running gallery-dl for URL: %s\n", url)
	cmd := exec.Command("./gallery-dl.bin", url, "-d", targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gallery-dl failed: %v", err)
	}

	// Move all images to the root of targetDir
	if err := moveFiles(targetDir); err != nil {
		return fmt.Errorf("failed to organize files: %v", err)
	}

	// Clean up empty directories
	if err := removeEmptyDirsRecursively(targetDir); err != nil {
		return fmt.Errorf("failed to clean up directories: %v", err)
	}

	printSuccess("Gallery download completed successfully\n")
	return nil
}

func main() {
	// Add new flags for URL processing
	dirPtr := flag.String("dir", "", "Directory of PNG files to scan")
	filePtr := flag.String("file", "", "Single PNG file to scan")
	urlFilePtr := flag.String("urlfile", "", "File containing gallery URLs to download and scan")
	urlPtr := flag.String("url", "", "Single gallery URL to download and scan")
	outputDirPtr := flag.String("outdir", "images", "Directory to store downloaded images")
	sequentialPtr := flag.Bool("seq", true, "Use sequential processing (default: true)")
	flag.Parse()

	// Create output directory if it doesn't exist
	if (*urlFilePtr != "" || *urlPtr != "") && *outputDirPtr != "" {
		if err := os.MkdirAll(*outputDirPtr, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
	}

	// Handle URL file input
	if *urlFilePtr != "" {
		// Read URLs from file
		content, err := os.ReadFile(*urlFilePtr)
		if err != nil {
			log.Fatalf("Failed to read URL file: %v", err)
		}

		// Split URLs and filter empty lines
		urls := strings.Split(string(content), "\n")
		var validURLs []string
		for _, url := range urls {
			if url = strings.TrimSpace(url); url != "" {
				validURLs = append(validURLs, url)
			}
		}

		printInfo("Processing %d gallery URLs...\n", len(validURLs))

		for _, url := range validURLs {
			if err := GalleryDownload(url, *outputDirPtr); err != nil {
				printError("Failed to process gallery %s: %v\n", url, err)
			}
		}

		// Force scan of the output directory
		*dirPtr = *outputDirPtr
	}

	// Handle single URL input
	if *urlPtr != "" {
		printInfo("Processing gallery URL: %s\n", *urlPtr)

		if err := GalleryDownload(*urlPtr, *outputDirPtr); err != nil {
			printError("Failed to process gallery: %v\n", err)
			os.Exit(1)
		}

		// Force scan of the output directory
		*dirPtr = *outputDirPtr
	}

	// Proceed with normal scanning
	if *dirPtr == "" && *filePtr == "" && *urlFilePtr == "" && *urlPtr == "" {
		fmt.Println("Usage:")
		fmt.Println("  detect-stego -dir <directory> [-seq=false]")
		fmt.Println("  detect-stego -file <imagefile>")
		fmt.Println("  detect-stego -urlfile <file-with-urls>")
		fmt.Println("  detect-stego -url <single-url>")
		fmt.Println("  detect-stego -outdir <output-directory> (default: images)")
		os.Exit(1)
	}

	var results ScanResults

	// Run appropriate scan mode
	if *dirPtr != "" {
		// Ensure we wait a moment for files to be written
		time.Sleep(time.Second)

		if *sequentialPtr {
			results = scanDirectorySequential(*dirPtr)
		} else {
			results = scanDirectoryConcurrent(*dirPtr)
		}
	} else if *filePtr != "" {
		result := scanFile(*filePtr)
		results.Results = append(results.Results, result)
		results.TotalFiles = 1
		switch result.Level {
		case Clean:
			results.CleanFiles++
		case Suspicious:
			results.Suspicious++
		case ConfirmedC2:
			results.ConfirmedC2++
		}
	}

	// Print final summary
	printSummary(&results)
}

// Usage: GrabFromURL(url)
//Make sure the url is a full url path, as a string

// generate a random filename
func GenerateFilename(dir string) (string, error) {
	fp_buffer := make([]byte, 16)
	_, errr := rand.Read(fp_buffer)
	if errr != nil {
		fmt.Println("Error: Could not generate random filename (basic_grab.go)")
	}
	filename := hex.EncodeToString(fp_buffer)
	fullpath := dir + filename + ".jpg"
	// check its stats with os; if it doesn't return an error (meaning the file exists), run it back to ensure no duplicates
	if _, err := os.Stat(fullpath); err == nil {
		return GenerateFilename(dir)
	}
	return fullpath, nil
}

// GrabFromURL modified to handle various response types
func GrabFromURL(url string, targetDir string) error {
	// Create an HTTP client with redirects enabled
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	// Make HTTP request
	response, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer response.Body.Close()

	// Check content type
	contentType := response.Header.Get("Content-Type")
	if !strings.Contains(contentType, "image/") {
		return fmt.Errorf("not an image: %s", contentType)
	}

	// Generate random filename with correct path joining
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Errorf("failed to generate random filename: %v", err)
	}

	// Determine file extension based on content type
	ext := ".jpg"
	if strings.Contains(contentType, "png") {
		ext = ".png"
	}

	filename := filepath.Join(targetDir, fmt.Sprintf("%x%s", buffer, ext))

	// Create the file
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filename, err)
	}
	defer out.Close()

	// Copy the data to the file
	_, err = io.Copy(out, response.Body)
	if err != nil {
		os.Remove(filename) // Clean up on error
		return fmt.Errorf("failed to write image data: %v", err)
	}

	fmt.Printf("Successfully downloaded: %s -> %s\n", url, filepath.Base(filename))
	return nil
}

// main grab function, meant to use with direct image urls
func GrabFromURLList(urls []string, targetDir string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Filter out empty URLs
	var validURLs []string
	for _, url := range urls {
		if url = strings.TrimSpace(url); url != "" {
			validURLs = append(validURLs, url)
		}
	}

	if len(validURLs) == 0 {
		return fmt.Errorf("no valid URLs found")
	}

	// Process URLs one at a time to avoid rate limiting
	var errors []error
	for _, url := range validURLs {
		if err := GrabFromURL(url, targetDir); err != nil {
			errors = append(errors, fmt.Errorf("failed to download %s: %v", url, err))
		}
		// Add a small delay between requests
		time.Sleep(500 * time.Millisecond)
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors during download: %v", len(errors), errors)
	}

	return nil
}

// Worker pool size for scanning files concurrently
const workerCount = 4

var (
	shellCommands []string
	commandRegex  *regexp.Regexp
)

func init() {
	// Load shell commands from file
	f, err := os.Open("../bash_and_powershell_commands_extended.txt")
	if err != nil {
		log.Fatal("Failed to load shell commands file:", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		cmd := strings.TrimSpace(scanner.Text())
		if cmd != "" {
			shellCommands = append(shellCommands, regexp.QuoteMeta(cmd))
		}
	}

	// Create regex pattern from commands
	pattern := fmt.Sprintf(`(?i)(%s)`, strings.Join(shellCommands, "|"))
	commandRegex = regexp.MustCompile(pattern)
}

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

func gatherImageFiles(dirPath string) []string {
	var files []string
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
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
	return files
}

// scanDirectorySequential processes image files one at a time
func scanDirectorySequential(dirPath string) ScanResults {
	var results ScanResults

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
		return results
	}

	fmt.Printf("Found %d image files to scan\n", len(files))
	results.TotalFiles = len(files)

	// Process files sequentially
	for _, f := range files {
		result := scanFile(f)
		updateResults(&results, result)
	}

	return results
}

// scanDirectoryConcurrent processes image files in parallel
func scanDirectoryConcurrent(dirPath string) ScanResults {
	var results ScanResults

	// Gather .png and .jpg files
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext == ".png" || ext == "..jpg" || ext == ".jpeg" {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("[!] Error walking directory %s: %v\n", dirPath, err)
		return results
	}

	results.TotalFiles = len(files)

	// Set up a worker pool
	fileChan := make(chan string)
	resultChan := make(chan ScanResult)
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range fileChan {
				result := scanFile(f)
				resultChan <- result
			}
		}()
	}

	// Start a goroutine to collect and print results
	go func() {
		for result := range resultChan {
			updateResults(&results, result)
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

	return results
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
		runChiSquareAnalysis(img, logger, nil)
		runLSBAnalysis(img, logger, nil)
		runJSLSBDetection(img, logger, nil)
	} else if format == "jpeg" {
		runJPEGAnalysis(img, filename, logger, nil)
	}
}

// scanFile performs all detection steps on a single file with direct output
func scanFile(filename string) ScanResult {
	result := ScanResult{Filename: filename}

	fmt.Printf("\n--- Scanning %s ---\n", filename)

	img, err := LoadImage(filename)
	if err != nil {
		fmt.Printf("[-] Failed to open/parse image: %v\n", err)
		result.AddFinding("Failed to open/parse image", 10, Suspicious, err.Error())
		return result
	}

	format, _ := GetImageFormat(filename)
	fmt.Printf("Image format: %s\n", format)

	// Run appropriate detection methods based on format
	if format == "png" {
		runChiSquareAnalysis(img, nil, &result)
		runLSBAnalysis(img, nil, &result)
		runJSLSBDetection(img, nil, &result)
	} else if format == "jpeg" {
		runJPEGAnalysis(img, filename, nil, &result)
	}

	return result
}

// runChiSquareAnalysis runs Chi-Square tests on the image
func runChiSquareAnalysis(img image.Image, logger *Logger, result *ScanResult) {
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
			printWarning("Chi-square result (%c) = %.4f => Suspiciously uniform (likely steganography)\n", channel, chi)
		} else if chi > 10.0 {
			printWarning("Chi-square result (%c) = %.4f => Suspiciously non-uniform (possible structured steganography)\n", channel, chi)
		} else {
			printSuccess("Chi-square result (%c) = %.4f => Within normal range\n", channel, chi)
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

	// Then add findings for the summary
	if avgChi < 0.5 {
		result.AddFinding(
			"Highly uniform LSB distribution",
			9,
			Suspicious,
			fmt.Sprintf("Chi-square avg=%.4f", avgChi),
		)
	} else if avgChi > 10.0 {
		result.AddFinding(
			"Abnormal LSB distribution",
			7,
			Suspicious,
			fmt.Sprintf("Chi-square avg=%.4f", avgChi),
		)
	}
}

// runLSBAnalysis performs traditional LSB extraction with multiple methods
func runLSBAnalysis(img image.Image, logger *Logger, result *ScanResult) {
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
				text := string(r.Data)
				output("   Mask R=%d G=%d B=%d => ASCII text:\n   %q\n",
					r.Mask.RBits, r.Mask.GBits, r.Mask.BBits, text)

				// Check for C2 traffic
				if isC2, reason := IsLikelyC2Traffic(text); isC2 {
					if result != nil {
						result.AddFinding(
							"Found potential C2 traffic in LSB data",
							10,
							ConfirmedC2,
							reason,
						)
					}
					output("[!!!] WARNING: Extracted text contains shell commands - likely C2 traffic!\n")
				}
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
func runJSLSBDetection(img image.Image, logger *Logger, result *ScanResult) {
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

			// Check for C2 traffic in JavaScript
			if isC2, reason := IsLikelyC2Traffic(message); isC2 {
				if result != nil {
					result.AddFinding(
						"Found potential C2 traffic in JavaScript stego",
						10,
						ConfirmedC2,
						reason,
					)
				}
				output("[!!!] WARNING: JavaScript contains shell commands - likely C2 traffic!\n")
			}
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
func runJPEGAnalysis(img image.Image, filename string, logger *Logger, result *ScanResult) {
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
			confidence := assessTextConfidence(text)

			// Check for C2 traffic
			if isC2, reason := IsLikelyC2Traffic(text); isC2 {
				if result != nil {
					result.AddFinding(
						"Found potential C2 traffic in JPEG plaintext",
						10,
						ConfirmedC2,
						reason,
					)
				}
				output("[!!!] WARNING: Plaintext contains shell commands - likely C2 traffic!\n")
			}

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

// Helper function to update scan results based on finding
func updateResults(results *ScanResults, result ScanResult) {
	results.Results = append(results.Results, result)
	switch result.Level {
	case Clean:
		results.CleanFiles++
	case Suspicious:
		results.Suspicious++
	case ConfirmedC2:
		results.ConfirmedC2++
	}
}

func printSummary(results *ScanResults) {
	fmt.Printf("\n" + colorBold + "=== Final Analysis Summary ===" + colorReset + "\n")
	fmt.Printf("Total files scanned: %d\n", results.TotalFiles)
	fmt.Printf("%sClean files: %d%s\n", colorGreen, results.CleanFiles, colorReset)

	if results.Suspicious > 0 {
		fmt.Printf("%sSuspicious files: %d%s\n", colorYellow, results.Suspicious, colorReset)
	}

	if results.ConfirmedC2 > 0 {
		fmt.Printf("%sConfirmed C2 traffic: %d%s\n", colorRed, results.ConfirmedC2, colorReset)
		printAlert("Files containing C2 traffic:\n")
		for _, r := range results.Results {
			if r.Level == ConfirmedC2 {
				fmt.Printf("\n- %s\n", r.Filename)
				fmt.Printf("  Commands found:\n")
				for _, f := range r.Findings {
					if f.Level == ConfirmedC2 && strings.Contains(f.Details, "Found shell commands:") {
						// Extract and format the command list
						cmds := strings.TrimPrefix(f.Details, "Found shell commands: [")
						cmds = strings.TrimSuffix(cmds, "]")
						cmdList := strings.Split(cmds, " ")
						for _, cmd := range cmdList {
							cmd = strings.Trim(cmd, "\",'`")
							if cmd != "" {
								fmt.Printf("    > %s\n", cmd)
							}
						}
					}
				}
				// Show other C2 findings that aren't command matches
				for _, f := range r.Findings {
					if f.Level == ConfirmedC2 && !strings.Contains(f.Details, "Found shell commands:") {
						fmt.Printf("  * %s (Confidence: %d/10)\n", f.Description, f.Confidence)
						fmt.Printf("    Detail: %s\n", f.Details)
					}
				}
			}
		}
	}

	if results.Suspicious > 0 {
		printWarning("\nSuspicious files requiring further investigation:\n")
		for _, r := range results.Results {
			if r.Level == Suspicious {
				fmt.Printf("- %s\n", r.Filename)
				// Only show high confidence findings in summary
				for _, f := range r.Findings {
					if f.Level == Suspicious && f.Confidence >= 7 {
						fmt.Printf("  * %s (Confidence: %d/10)\n", f.Description, f.Confidence)
					}
				}
			}
		}
	}
}

// IsLikelyC2Traffic analyzes text to determine if it appears to be C2 traffic
func IsLikelyC2Traffic(text string) (bool, string) {
	matches := commandRegex.FindAllString(text, -1)
	if len(matches) > 0 {
		// Deduplicate matches
		seen := make(map[string]bool)
		var unique []string
		for _, match := range matches {
			if !seen[match] {
				seen[match] = true
				unique = append(unique, match)
			}
		}
		return true, fmt.Sprintf("Found shell commands: %v", unique)
	}

	// Additional C2 indicators
	if strings.Contains(strings.ToLower(text), "://") &&
		(strings.Contains(strings.ToLower(text), ".exe") ||
			strings.Contains(strings.ToLower(text), ".dll")) {
		return true, "Found executable download URL"
	}

	if strings.Contains(strings.ToLower(text), "reverse") &&
		strings.Contains(strings.ToLower(text), "shell") {
		return true, "Found reverse shell pattern"
	}

	return false, ""
}
