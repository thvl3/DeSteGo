package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// main orchestrates everything from a command-line perspective.
// Usage: go run main.go -file <pngfile> OR -dir <directory-of-pngs>
func main() {
	dirPtr := flag.String("dir", "", "Directory of PNG files to scan")
	filePtr := flag.String("file", "", "Single PNG file to scan")
	flag.Parse()

	if *dirPtr == "" && *filePtr == "" {
		fmt.Println("Usage:")
		fmt.Println("  stegodetect -dir <directory>")
		fmt.Println("  stegodetect -file <pngfile>")
		os.Exit(1)
	}

	if *dirPtr != "" {
		scanDirectory(*dirPtr)
	} else {
		scanFile(*filePtr)
	}
}

// scanDirectory recursively scans all .png files in dirPath.
func scanDirectory(dirPath string) {
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".png" {
			scanFile(path)
		}
		return nil
	})
	if err != nil {
		log.Printf("[!] Error walking directory %s: %v\n", dirPath, err)
	}
}

// scanFile applies our detection methods to a single PNG file.
func scanFile(filename string) {
	fmt.Printf("\n--- Scanning %s ---\n", filename)
	img, err := LoadPNG(filename)
	if err != nil {
		fmt.Printf("[-] Failed to open/parse PNG: %v\n", err)
		return
	}

	// 1) Run a Chi-Square analysis on the red-channel LSB distribution
	chiR := ChiSquareLSB(img, 'R')
	if IsSuspiciousChiSquare(chiR) {
		fmt.Printf("[!] Chi-square result (R) = %.4f => Suspiciously uniform distribution!\n", chiR)
	} else {
		fmt.Printf("[ ] Chi-square result (R) = %.4f => Not strongly suspicious.\n", chiR)
	}
	// 1b) Run a Chi-Square analysis on the green-channel LSB distribution
	chiG := ChiSquareLSB(img, 'G')
	if IsSuspiciousChiSquare(chiG) {
		fmt.Printf("[!] Chi-square result (G) = %.4f => Suspiciously uniform distribution!\n", chiG)
	} else {
		fmt.Printf("[ ] Chi-square result (G) = %.4f => Not strongly suspicious.\n", chiG)
	}
	// 1c) Run a Chi-Square analysis on the blue-channel LSB distribution
	chiB := ChiSquareLSB(img, 'B')
	if IsSuspiciousChiSquare(chiB) {
		fmt.Printf("[!] Chi-square result (B) = %.4f => Suspiciously uniform distribution!\n", chiB)
	} else {
		fmt.Printf("[ ] Chi-square result (B) = %.4f => Not strongly suspicious.\n", chiB)
	}
	// 2) Brute Force LSB extraction with multiple channel/bit combos
	results := BruteForceLSB(img)
	if len(results) == 0 {
		fmt.Println("[ ] No LSB-embedded data found with the tested channel masks.")
		return
	}

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
