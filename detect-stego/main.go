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
	"strconv"
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

	// Convert any JPEGs to PNGs before scanning
	if *dirPtr != "" {
		printInfo("Converting JPEG files to PNG format...\n")
		converted, err := ConvertAllJPEGs(*dirPtr)
		if err != nil {
			printError("Error during conversion: %v\n", err)
		} else if len(converted) > 0 {
			printSuccess("Converted %d files to PNG format\n", len(converted))
		}
	} else if *filePtr != "" && (strings.HasSuffix(strings.ToLower(*filePtr), ".jpg") ||
		strings.HasSuffix(strings.ToLower(*filePtr), ".jpeg")) {
		printInfo("Converting JPEG file to PNG format...\n")
		newFile, err := ConvertToPNG(*filePtr)
		if err != nil {
			printError("Failed to convert file: %v\n", err)
			os.Exit(1)
		}
		*filePtr = newFile
		printSuccess("Converted to %s\n", newFile)
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

// Generate a random filename
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
			// Skip "df" and "DF" commands to avoid false positives
			if cmd == "df" || cmd == "DF" {
				continue
			}
			shellCommands = append(shellCommands, regexp.QuoteMeta(cmd))
		}
	}

	// Create regex pattern from commands with word boundaries
	// Add word boundaries to avoid matching substrings inside other words
	pattern := fmt.Sprintf(`(?i)\b(%s)\b`, strings.Join(shellCommands, "|"))
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
			if ext == "..jpg" || ext == ".jpeg" {
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
		runStatisticalAnalysis(img, logger, nil)
		runLSBAnalysis(img, logger, nil)
		runJSLSBDetection(img, logger, nil)
	} else if format == "jpeg" {
		runJPEGAnalysis(filename, logger, nil)
	}
}

// Scan a file for steganography, analyzing both JPEG and PNG aspects
func scanFile(filename string) ScanResult {
	result := ScanResult{Filename: filename}

	fmt.Printf("\n--- Scanning %s ---\n", filename)

	// First analyze original file if it's a JPEG
	if strings.HasSuffix(strings.ToLower(filename), ".jpg") ||
		strings.HasSuffix(strings.ToLower(filename), ".jpeg") {
		runJPEGAnalysis(filename, nil, &result)
	}

	// Then analyze as PNG (either original PNG or converted JPEG)
	img, err := LoadImage(filename)
	if err != nil {
		fmt.Printf("[-] Failed to open/parse image: %v\n", err)
		result.AddFinding("Failed to open/parse image", 10, Suspicious, err.Error())
		return result
	}

	// Run PNG analysis with statistical analysis instead of Chi-Square
	runStatisticalAnalysis(img, nil, &result)
	runLSBAnalysis(img, nil, &result)
	runJSLSBDetection(img, nil, &result)

	// Filter out low-confidence findings
	result.Findings = filterFindings(result.Findings)

	return result
}

// Filter findings to reduce false positives
func filterFindings(findings []Finding) []Finding {
	var filtered []Finding

	for _, f := range findings {
		// Skip low-confidence LSB findings unless they have strong indicators
		if strings.Contains(f.Description, "LSB") {
			// Require higher confidence for LSB-based detections
			if f.Confidence < 9 {
				continue
			}

			// Additional validation for LSB findings
			if !containsStrongIndicators(f.Details) {
				continue
			}
		}

		// Skip findings that are common in normal images
		if isCommonPattern(f.Description, f.Details) {
			continue
		}

		filtered = append(filtered, f)
	}

	return filtered
}

// Check if details contain strong indicators of steganography
func containsStrongIndicators(details string) bool {
	strongIndicators := []string{
		"shell", "password", "secret", "admin",
		"http://", "https://", "ftp://",
		".exe", ".dll", ".sh", ".cmd",
	}

	details = strings.ToLower(details)

	for _, indicator := range strongIndicators {
		if strings.Contains(details, indicator) {
			return true
		}
	}

	if strings.Contains(details, "entropy") {
		if value := extractEntropyValue(details); value > 7.9 {
			return true
		}
	}

	return false
}

// Check if a pattern is common in normal images (to filter false positives)
func isCommonPattern(description, details string) bool {
	commonPatterns := []string{
		"slight variation in LSB",
		"minor statistical anomaly",
		"low entropy distribution",
		"standard JPEG pattern",
	}

	text := strings.ToLower(description + " " + details)
	for _, pattern := range commonPatterns {
		if strings.Contains(text, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// Extract entropy value from text string
func extractEntropyValue(text string) float64 {
	re := regexp.MustCompile(`entropy=(\d+\.\d+)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return value
		}
	}
	return 0
}

// Run LSB analysis on an image
func runLSBAnalysis(img image.Image, logger *Logger, result *ScanResult) {
	output := func(format string, args ...interface{}) {
		if logger != nil {
			logger.Printf(format, args...)
		} else {
			fmt.Printf(format, args...)
		}
	}

	output("\n=== LSB Brute Force Analysis ===\n")

	// Add more comprehensive LSB extraction attempts
	extractLSB := func(mask ChannelMask) []byte {
		// Try different bit orders and methods
		data := ExtractBitsDirectly(img, mask)
		if len(data) > 0 && IsASCIIPrintable(data) {
			return data
		}

		// Try reverse bit order
		data = ExtractBitsReverse(img, mask)
		if len(data) > 0 && IsASCIIPrintable(data) {
			return data
		}

		return nil
	}

	// Try common LSB patterns
	masks := []ChannelMask{
		{RBits: 1, GBits: 0, BBits: 0, ABits: 0}, // R channel
		{RBits: 0, GBits: 1, BBits: 0, ABits: 0}, // G channel
		{RBits: 0, GBits: 0, BBits: 1, ABits: 0}, // B channel
		{RBits: 1, GBits: 1, BBits: 1, ABits: 0}, // All channels
		{RBits: 1, GBits: 1, BBits: 0, ABits: 0}, // RG channels
		{RBits: 0, GBits: 1, BBits: 1, ABits: 0}, // GB channels
	}

	foundData := false
	for _, mask := range masks {
		data := extractLSB(mask)
		if data != nil {
			foundData = true
			output("[+] Found hidden data using mask R:%d G:%d B:%d:\n",
				mask.RBits, mask.GBits, mask.BBits)
			output("    %q\n", string(data))

			// Check for C2 traffic or other suspicious content
			if isC2, reason := IsLikelyC2Traffic(string(data)); isC2 {
				result.AddFinding(
					"Found potential C2 traffic in LSB data",
					10,
					ConfirmedC2,
					reason,
				)
			} else if IsSuspiciousText(string(data)) {
				result.AddFinding(
					"Found suspicious text in LSB data",
					7,
					Suspicious,
					fmt.Sprintf("Text: %s", string(data)),
				)
			}
		}
	}

	if !foundData {
		output("[ ] No readable LSB data found\n")
	}

	// Create a progress callback function that only prints on major milestones
	progressCb := func(percentComplete float64, message string) {
		// Only print progress at 25%, 50%, 75% and 100%
		if percentComplete == 0 || percentComplete >= 100 ||
			int(percentComplete)%25 == 0 {
			if logger != nil {
				logger.Printf("\rLSB Analysis: %.0f%% complete", percentComplete)
			} else {
				fmt.Printf("\rLSB Analysis: %.0f%% complete", percentComplete)
			}
			if percentComplete >= 100 {
				fmt.Println() // Add newline at end
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

// Run JavaScript LSB detection
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

// Run JPEG-specific analysis
func runJPEGAnalysis(filename string, logger *Logger, result *ScanResult) {
	output := func(format string, args ...interface{}) {
		if logger != nil {
			logger.Printf(format, args...)
		} else {
			fmt.Printf(format, args...)
		}
	}

	output("\n=== JPEG Analysis ===\n")
	output("Starting analysis of %s\n", filepath.Base(filename))

	// Extract metadata first
	metadata, err := ExtractJPEGMetadata(filename)
	if err != nil {
		output("[-] Failed to extract JPEG metadata: %v\n", err)
		result.AddFinding("Failed to extract JPEG metadata", 5, Suspicious, err.Error())
		return
	}

	// Basic file analysis
	output("Dimensions: %dx%d\n", metadata.Width, metadata.Height)
	output("Color components: %d\n", metadata.Components)
	output("Progressive: %v\n", metadata.IsProgressive)

	// Check for appended data
	if metadata.HasAppendedData {
		output("%s[!] Found %d bytes of appended data after JPEG EOF marker%s\n",
			colorYellow, metadata.AppendedDataSize, colorReset)
		result.AddFinding("Found appended data after EOF", 8, Suspicious,
			fmt.Sprintf("Found %d bytes of appended data", metadata.AppendedDataSize))
	}

	// Report quantization table analysis
	output("\nAnalyzing quantization tables...\n")
	modified, numTables := CheckQuantizationTables(metadata)
	if modified {
		output("%s[!] Detected modified quantization tables (%d tables)%s\n",
			colorYellow, numTables, colorReset)
		result.AddFinding("Modified quantization tables detected", 7, Suspicious,
			fmt.Sprintf("Found %d modified tables", numTables))
	} else {
		output("[+] Quantization tables appear normal (%d tables)\n", numTables)
	}

	// Run steganography detection
	output("\nRunning steganalysis...\n")
	stegResults, err := AnalyzeJPEG(filename)
	if err != nil {
		output("[-] Steganalysis failed: %v\n", err)
	} else {
		// Report detection results with confidence scores
		if stegResults.JStegProbability > 0.2 {
			output("%s[!] JSteg probability: %.1f%%%s\n",
				colorYellow, stegResults.JStegProbability*100, colorReset)
			if details, ok := stegResults.Details["JSteg"]; ok {
				output("    %s\n", details)
				if stegResults.JStegProbability > 0.4 {
					result.AddFinding("JSteg steganography detected",
						int(7+stegResults.JStegProbability*3), Suspicious,
						fmt.Sprintf("Confidence: %.1f%%", stegResults.JStegProbability*100))
				}
			}
		}

		if stegResults.F5Probability > 0.2 {
			output("%s[!] F5 probability: %.1f%%%s\n",
				colorYellow, stegResults.F5Probability*100, colorReset)
			if details, ok := stegResults.Details["F5"]; ok {
				output("    %s\n", details)
				if stegResults.F5Probability > 0.4 {
					result.AddFinding("F5 steganography detected",
						int(7+stegResults.F5Probability*3), Suspicious,
						fmt.Sprintf("Confidence: %.1f%%", stegResults.F5Probability*100))
				}
			}
		}

		if stegResults.OutGuessProbability > 0.2 {
			output("%s[!] OutGuess probability: %.1f%%%s\n",
				colorYellow, stegResults.OutGuessProbability*100, colorReset)
			if details, ok := stegResults.Details["OutGuess"]; ok {
				output("    %s\n", details)
				if stegResults.OutGuessProbability > 0.4 {
					result.AddFinding("OutGuess steganography detected",
						int(7+stegResults.OutGuessProbability*3), Suspicious,
						fmt.Sprintf("Confidence: %.1f%%", stegResults.OutGuessProbability*100))
				}
			}
		}
	}

	// Look for hidden text
	output("\nScanning for hidden text...\n")
	findings, err := ScanForPlaintextStego(filename)
	if err == nil && len(findings) > 0 {
		output("%s[!] Found potential hidden text:%s\n", colorYellow, colorReset)
		for _, text := range findings {
			output("    > %s\n", text)

			// Check if the found text contains C2 indicators
			if isC2, reason := IsLikelyC2Traffic(text); isC2 {
				result.AddFinding(
					"Found potential C2 traffic in hidden text",
					10,
					ConfirmedC2,
					reason,
				)
			} else {
				result.AddFinding(
					"Found hidden text",
					7,
					Suspicious,
					fmt.Sprintf("Text: %s", text),
				)
			}
		}
	}

	// Check for polyglot files
	if isPolyglot, format := ScanForPolyglotFile(filename); isPolyglot {
		output("%s[!] File appears to be a polyglot - also contains %s format%s\n",
			colorYellow, format, colorReset)
		result.AddFinding(
			"Polyglot file detected",
			8,
			Suspicious,
			fmt.Sprintf("File also contains %s format", format))
	}

	output("\nAnalysis complete.\n")
}

// Check if text is suspicious
func IsSuspiciousText(text string) bool {
	lowered := strings.ToLower(text)

	// Create word boundaries for these patterns to avoid false matches
	suspiciousPatterns := []string{
		"\\bpassword\\b", "\\bsecret\\b", "\\bkey\\b", "\\btoken\\b",
		"\\badmin\\b", "\\broot\\b", "\\bshell\\b", "\\bbash\\b",
		"http://", "https://", "ftp://",
		"\\.exe\\b", "\\.dll\\b", "\\.sh\\b", "\\.bat\\b",
	}

	for _, pattern := range suspiciousPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(lowered) {
			return true
		}
	}

	// Check for very high entropy sections only
	if ComputeEntropy([]byte(text)) > 7.2 { // Increased from 6.5
		return true
	}

	return false
}

// Extract bits in reverse order
func ExtractBitsReverse(img image.Image, mask ChannelMask) []byte {
	bounds := img.Bounds()
	var bytesBuilder bytes.Buffer
	currentByte := byte(0)
	bitCount := 0

	// Extract bits in reverse order
	for y := bounds.Max.Y - 1; y >= bounds.Min.Y; y-- {
		for x := bounds.Max.X - 1; x >= bounds.Min.X; x-- {
			r, g, b, a := img.At(x, y).RGBA()

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

			if bitCount >= 8 {
				bytesBuilder.WriteByte(currentByte)
				currentByte = 0
				bitCount = 0
			}
		}
	}

	return bytesBuilder.Bytes()
}

// Try to extract data using specific channel mask
func tryExtractChannelMask(img image.Image, mask ChannelMask, useLength bool,
	progressCb ProgressCallback, output func(string, ...interface{})) {
	var data []byte
	var err error

	// Don't print the starting message
	if useLength {
		data, err = ExtractData(img, mask, LSBFirst, nil) // Pass nil for progress callback
	} else {
		data = ExtractLSBNoLength(img, mask, LSBFirst, nil) // Pass nil for progress callback
		if len(data) == 0 {
			err = fmt.Errorf("no data extracted")
		}
	}

	// Only output if we found something
	if err == nil && len(data) > 0 && IsASCIIPrintable(data) {
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

// Extract bits directly from image
func ExtractBitsDirectly(img image.Image, mask ChannelMask) []byte {
	bounds := img.Bounds()
	var bytesBuilder bytes.Buffer
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

// Helper function to update scan results
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

// Print summary of scan results
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

// Check if text appears to be C2 traffic
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

// Get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Assess how likely a text is to be an intentional hidden message
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

// Run the new statistical analysis
func runStatisticalAnalysis(img image.Image, logger *Logger, result *ScanResult) {
	defer (func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in statistical analysis: %v\n", r)
			fmt.Println("Skipping statistical analysis for this image")
			// Add error to results
			if result != nil {
				result.AddFinding("Statistical analysis error", 5, Suspicious, fmt.Sprintf("Panic: %v", r))
			}
		}
	})()

	output := func(format string, args ...interface{}) {
		if logger != nil {
			logger.Printf(format, args...)
		} else {
			fmt.Printf(format, args...)
		}
	}

	output("\n=== Advanced Statistical Analysis ===\n")

	// Run the new statistical analysis
	anomalyScore, dist, err := DetectSteganoAnomaly(img)
	if err != nil {
		output("[-] Failed to perform statistical analysis: %v\n", err)
		return
	}

	// Output detailed statistics
	output("LSB Distribution Analysis:\n")
	output("- Overall entropy: %.4f (closer to 1.0 = more random)\n", dist.Entropy)
	output("- Channel statistics:\n")
	output("  - Red:   Entropy=%.4f\n", dist.ChannelStats["R"].Entropy)
	output("  - Green: Entropy=%.4f\n", dist.ChannelStats["G"].Entropy)
	output("  - Blue:  Entropy=%.4f\n", dist.ChannelStats["B"].Entropy)

	// Interpret the results
	output("\nFinal anomaly score: %.4f ", anomalyScore)

	// Only flag images with very high anomaly scores
	if anomalyScore > 0.85 {
		output("(HIGHLY SUSPICIOUS)\n")
		if result != nil {
			result.AddFinding(
				"Highly anomalous LSB distribution",
				9, // High confidence
				Suspicious,
				fmt.Sprintf("Statistical anomaly score=%.4f (>0.85 is suspicious)", anomalyScore),
			)
		}
	} else if anomalyScore > 0.75 {
		output("(SOMEWHAT SUSPICIOUS)\n")
		if result != nil {
			result.AddFinding(
				"Unusual LSB distribution",
				7, // Medium confidence
				Suspicious,
				fmt.Sprintf("Statistical anomaly score=%.4f (>0.75 is unusual)", anomalyScore),
			)
		}
	} else {
		output("(NORMAL RANGE)\n")
	}

	// Also check for extreme entropy values which can indicate steganography
	if dist.Entropy > 0.99 {
		output("[!] Perfect entropy detected (%.4f) - highly suspicious\n", dist.Entropy)
		if result != nil {
			result.AddFinding(
				"Perfect LSB entropy",
				9,
				Suspicious,
				fmt.Sprintf("LSB entropy=%.4f (unnaturally perfect randomness)", dist.Entropy),
			)
		}
	} else if dist.Entropy < 0.3 {
		output("[!] Extremely low entropy detected (%.4f) - suspicious\n", dist.Entropy)
		if result != nil {
			result.AddFinding(
				"Abnormally low LSB entropy",
				8,
				Suspicious,
				fmt.Sprintf("LSB entropy=%.4f (unnaturally low randomness)", dist.Entropy),
			)
		}
	}
}
