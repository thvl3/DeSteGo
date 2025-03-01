package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Worker pool size for scanning files concurrently
const workerCount = 4

// main orchestrates everything from the command line.
//
// Usage examples:
//
//	go run testmain.go -dir ./images
//	go run testmain.go -file sample.png
func main() {
	dirPtr := flag.String("dir", "", "Directory of PNG files to scan")
	filePtr := flag.String("file", "", "Single PNG file to scan")
	flag.Parse()

	if *dirPtr == "" && *filePtr == "" {
		fmt.Println("Usage:")
		fmt.Println("  testmain -dir <directory>")
		fmt.Println("  testmain -file <pngfile>")
		os.Exit(1)
	}

	if *dirPtr != "" {
		scanDirectory(*dirPtr)
	} else {
		// Single-file scan
		scanFile(*filePtr)
	}
}

// scanDirectory enumerates PNG files and processes them in parallel.
func scanDirectory(dirPath string) {
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
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range fileChan {
				scanFile(f)
			}
		}()
	}

	// Feed the files to the workers
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	// Wait for all workers to finish
	wg.Wait()
}

// scanFile performs all detection steps on a single file.
func scanFile(filename string) {
	fmt.Printf("\n--- Scanning %s ---\n", filename)
	img, err := LoadPNG(filename)
	if err != nil {
		fmt.Printf("[-] Failed to open/parse PNG: %v\n", err)
		return
	}

	//
	// 1) Run a Chi-Square analysis on R, G, B channels separately.
	//    We assume you've modified ChiSquareLSB(img, channel rune) to handle 'R','G','B'.
	//
	chiR := ChiSquareLSB(img, 'R')
	if IsSuspiciousChiSquare(chiR) {
		fmt.Printf("[!] Chi-square result (R) = %.4f => Suspiciously uniform distribution!\n", chiR)
	} else {
		fmt.Printf("[ ] Chi-square result (R) = %.4f => Not strongly suspicious.\n", chiR)
	}

	chiG := ChiSquareLSB(img, 'G')
	if IsSuspiciousChiSquare(chiG) {
		fmt.Printf("[!] Chi-square result (G) = %.4f => Suspiciously uniform distribution!\n", chiG)
	} else {
		fmt.Printf("[ ] Chi-square result (G) = %.4f => Not strongly suspicious.\n", chiG)
	}

	chiB := ChiSquareLSB(img, 'B')
	if IsSuspiciousChiSquare(chiB) {
		fmt.Printf("[!] Chi-square result (B) = %.4f => Suspiciously uniform distribution!\n", chiB)
	} else {
		fmt.Printf("[ ] Chi-square result (B) = %.4f => Not strongly suspicious.\n", chiB)
	}

	//
	// 2) Brute Force LSB extraction with multiple channel/bit combos
	//    (assuming you have a function BruteForceLSB(img) that returns
	//    a slice of { Mask, Data } or similar).
	//
	results := BruteForceLSB(img)
	if len(results) == 0 {
		fmt.Println("[ ] No LSB-embedded data found with the tested channel masks.")
		return
	}

	//
	// 3) Print out extracted data or note if it's likely encrypted.
	//
	fmt.Printf("[+] Found %d potential embedded payload(s) via LSB brute force:\n", len(results))
	for _, r := range results {
		if IsASCIIPrintable(r.Data) {
			fmt.Printf("   Mask R=%d G=%d B=%d => ASCII text:\n   %q\n",
				r.Mask.RBits, r.Mask.GBits, r.Mask.BBits, string(r.Data))
		} else {
			fmt.Printf("   Mask R=%d G=%d B=%d => Non-printable or encrypted data (%d bytes)\n",
				r.Mask.RBits, r.Mask.GBits, r.Mask.BBits, len(r.Data))
		}
	}
}
