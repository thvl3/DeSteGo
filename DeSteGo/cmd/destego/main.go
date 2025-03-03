package main

import (
	"DeSteGo/pkg/analyzer"
	jpeganalyzer "DeSteGo/pkg/analyzer/image/jpeg"
	pnganalyzer "DeSteGo/pkg/analyzer/image/png"
	"DeSteGo/pkg/filehandler"
	"DeSteGo/pkg/models"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	// Color printers
	infoColor    = color.New(color.FgBlue).SprintFunc()
	successColor = color.New(color.FgGreen).SprintFunc()
	warningColor = color.New(color.FgYellow).SprintFunc()
	errorColor   = color.New(color.FgRed).SprintFunc()
	alertColor   = color.New(color.FgRed, color.Bold).SprintFunc()
)

func printInfo(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", infoColor("[*]"), fmt.Sprintf(format, args...))
}

func printSuccess(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", successColor("[+]"), fmt.Sprintf(format, args...))
}

func printWarning(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", warningColor("[!]"), fmt.Sprintf(format, args...))
}

func printError(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", errorColor("[-]"), fmt.Sprintf(format, args...))
}

func printAlert(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", alertColor("[!!!]"), fmt.Sprintf(format, args...))
}

func main() {
	// Parse command line arguments
	var (
		filePath    = flag.String("file", "", "Path to a single file for analysis")
		dirPath     = flag.String("dir", "", "Path to directory of files for analysis")
		urlPath     = flag.String("url", "", "URL to download and analyze")
		urlFilePath = flag.String("urlfile", "", "Path to file containing URLs to download and analyze")
		outputDir   = flag.String("outdir", "destego_output", "Directory to store results and downloaded files")
		format      = flag.String("format", "auto", "Force specific format analysis (png, jpg, gif, svg)")
		verbose     = flag.Bool("verbose", false, "Enable verbose output")
		listFormats = flag.Bool("listformats", false, "List all supported file formats")
		sequential  = flag.Bool("seq", true, "Use sequential processing (default: true)")
		extractFlag = flag.Bool("extract", false, "Attempt to extract hidden data if found")
	)

	flag.Parse()

	// Banner and version info
	fmt.Println("DeSteGo v1.0.0")
	fmt.Println("A wide net steganography analysis tool")
	fmt.Println("Developed by Ethan Hulse")
	fmt.Println("---------------------------------")

	// Create registry and register analyzers
	registry := analyzer.NewRegistry()
	registerAnalyzers(registry)

	// Handle list formats flag
	if *listFormats {
		fmt.Println("Supported file formats:")
		formats := registry.GetSupportedFormats()
		for _, format := range formats {
			analyzers := registry.GetAnalyzersForFormat(format)
			names := make([]string, 0, len(analyzers))
			for _, a := range analyzers {
				names = append(names, a.Name())
			}
			fmt.Printf("- %s: %s\n", format, strings.Join(names, ", "))
		}
		return
	}

	// Ensure we have at least one input method
	if *filePath == "" && *dirPath == "" && *urlPath == "" && *urlFilePath == "" {
		fmt.Println("Usage:")
		fmt.Println("  destego -file <filepath>")
		fmt.Println("  destego -dir <directory>")
		fmt.Println("  destego -url <url>")
		fmt.Println("  destego -urlfile <file-with-urls>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		printError("Failed to create output directory: %v", err)
		os.Exit(1)
	}

	// Process URL file if specified
	if *urlFilePath != "" {
		printInfo("Processing URLs from file: %s", *urlFilePath)
		urls, err := filehandler.ReadLines(*urlFilePath)
		if err != nil {
			printError("Failed to read URL file: %v", err)
			os.Exit(1)
		}

		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url == "" || strings.HasPrefix(url, "#") {
				continue // Skip empty lines and comments
			}

			downloadDir := filepath.Join(*outputDir, "downloads")
			printInfo("Downloading from %s", url)
			filePath, err := filehandler.DownloadFromURL(url, downloadDir)
			if err != nil {
				printError("Failed to download from %s: %v", url, err)
				continue
			}
			printSuccess("Downloaded to %s", filePath)

			// Analyze the downloaded file
			analyzeFile(filePath, registry, *format, *verbose, *extractFlag)
		}
	}

	// Process single URL if specified
	if *urlPath != "" {
		printInfo("Downloading from URL: %s", *urlPath)
		downloadDir := filepath.Join(*outputDir, "downloads")
		filePath, err := filehandler.DownloadFromURL(*urlPath, downloadDir)
		if err != nil {
			printError("Failed to download from URL: %v", err)
			os.Exit(1)
		}
		printSuccess("Downloaded to %s", filePath)

		// Analyze the downloaded file
		analyzeFile(filePath, registry, *format, *verbose, *extractFlag)
	}

	// Process single file if specified
	if *filePath != "" {
		printInfo("Analyzing file: %s", *filePath)
		analyzeFile(*filePath, registry, *format, *verbose, *extractFlag)
	}

	// Process directory if specified
	if *dirPath != "" {
		printInfo("Analyzing directory: %s", *dirPath)
		files, err := filehandler.GatherFiles(*dirPath)
		if err != nil {
			printError("Failed to read directory: %v", err)
			os.Exit(1)
		}

		printInfo("Found %d files to analyze", len(files))

		var results []models.AnalysisResult

		if *sequential {
			for _, file := range files {
				result := analyzeFile(file, registry, *format, *verbose, *extractFlag)
				if result != nil {
					results = append(results, *result)
				}
			}
		} else {
			// TODO: Implement parallel processing
			printWarning("Parallel processing not yet implemented, using sequential")
			for _, file := range files {
				result := analyzeFile(file, registry, *format, *verbose, *extractFlag)
				if result != nil {
					results = append(results, *result)
				}
			}
		}

		// Print summary
		printSummary(results)
	}
}

func registerAnalyzers(registry *analyzer.Registry) {
	// Register all available analyzers
	registry.Register(pnganalyzer.NewPNGAnalyzer())
	registry.Register(jpeganalyzer.NewJPEGAnalyzer())
	// Add more analyzers as they become available
}

func analyzeFile(filePath string, registry *analyzer.Registry, formatHint string, verbose bool, extract bool) *models.AnalysisResult {
	// Detect file format
	format := formatHint
	if format == "auto" {
		detectedFormat, err := filehandler.DetectFileFormat(filePath)
		if err != nil {
			printError("Failed to detect file format: %v", err)
			return nil
		}
		format = detectedFormat
	}

	// Get appropriate analyzers
	analyzers := registry.GetAnalyzersForFormat(format)
	if len(analyzers) == 0 {
		printWarning("No analyzers available for format: %s", format)
		return nil
	}

	printInfo("Analyzing %s as %s format", filePath, format)
	startTime := time.Now()

	var finalResult *models.AnalysisResult

	// Run all applicable analyzers
	for _, a := range analyzers {
		printInfo("Running %s analyzer", a.Name())

		// Setup options
		options := analyzer.AnalysisOptions{
			Verbose: verbose,
			Format:  format,
			Extract: extract,
		}

		// Run analysis
		result, err := a.Analyze(filePath, options)
		if err != nil {
			printError("Analysis with %s failed: %v", a.Name(), err)
			continue
		}

		// Display results
		displayAnalysisResult(result, verbose)

		// Keep the result with highest detection score
		if finalResult == nil || result.DetectionScore > finalResult.DetectionScore {
			finalResult = result
		}
	}

	duration := time.Since(startTime)
	printInfo("Analysis completed in %v", duration)

	return finalResult
}

func displayAnalysisResult(result *models.AnalysisResult, verbose bool) {
	fmt.Println("\n--- Analysis Results ---")

	// Basic info
	fmt.Printf("File: %s\n", result.Filename)
	fmt.Printf("Format: %s\n", result.FileType)

	// Detection results
	if result.DetectionScore > 0.8 {
		printAlert("HIGH probability of steganography detected (%.2f)", result.DetectionScore)
	} else if result.DetectionScore > 0.5 {
		printWarning("MEDIUM probability of steganography detected (%.2f)", result.DetectionScore)
	} else if result.DetectionScore > 0.2 {
		printInfo("LOW probability of steganography detected (%.2f)", result.DetectionScore)
	} else {
		printSuccess("No steganography detected (%.2f)", result.DetectionScore)
	}

	// Confidence score
	fmt.Printf("Detection confidence: %.2f\n", result.Confidence)

	// Algorithm detection
	if result.PossibleAlgorithm != "" {
		fmt.Printf("Possible algorithm: %s\n", result.PossibleAlgorithm)
	}

	// Findings
	if len(result.Findings) > 0 {
		fmt.Println("\nFindings:")
		for i, finding := range result.Findings {
			fmt.Printf("%d. %s (Confidence: %.2f)\n", i+1, finding.Description, finding.Confidence)
			if verbose && finding.Details != "" {
				fmt.Printf("   Details: %s\n", finding.Details)
			}
		}
	}

	// Recommendations
	if len(result.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for i, rec := range result.Recommendations {
			fmt.Printf("%d. %s\n", i+1, rec)
		}
	}

	fmt.Println("-------------------------")
}

func printSummary(results []models.AnalysisResult) {
	var clean, suspicious, confirmed int

	for _, result := range results {
		if result.DetectionScore < 0.2 {
			clean++
		} else if result.DetectionScore < 0.7 {
			suspicious++
		} else {
			confirmed++
		}
	}

	fmt.Println("\n=== Analysis Summary ===")
	fmt.Printf("Total files analyzed: %d\n", len(results))
	fmt.Printf("%sClean files: %d%s\n", successColor("[+]"), clean, "")

	if suspicious > 0 {
		fmt.Printf("%sSuspicious files: %d%s\n", warningColor("[!]"), suspicious, "")
	}

	if confirmed > 0 {
		fmt.Printf("%sConfirmed steganography: %d%s\n", alertColor("[!!!]"), confirmed, "")

		fmt.Println("\nFiles with high probability of steganography:")
		for _, result := range results {
			if result.DetectionScore >= 0.7 {
				fmt.Printf("- %s (Score: %.2f)\n", result.Filename, result.DetectionScore)
			}
		}
	}
}
