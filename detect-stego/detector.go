package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Main entry point.
// Usage examples:
//
//	go run detector.go -file sample.png
//	go run detector.go -dir images/
func main() {
	dirPtr := flag.String("dir", "", "Directory of PNG files to scan")
	filePtr := flag.String("file", "", "Single PNG file to scan")
	flag.Parse()

	// We require either -dir or -file
	if *dirPtr == "" && *filePtr == "" {
		fmt.Println("Usage: detector -dir <directory> OR -file <pngfile>")
		os.Exit(1)
	}

	if *dirPtr != "" {
		scanDirectory(*dirPtr)
	} else {
		scanFile(*filePtr)
	}
}

// scanDirectory walks through 'dirPath' and scans every .png file
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
		log.Printf("Error walking directory %s: %v\n", dirPath, err)
	}
}

// scanFile loads a single PNG and tries to extract LSB-embedded data.
func scanFile(filename string) {
	img, err := LoadPNG(filename)
	if err != nil {
		log.Printf("[!] Failed to open/parse PNG %s: %v\n", filename, err)
		return
	}

	// Create a default mask that extracts LSB from each channel
	mask := ChannelMask{
		RBits: 1,
		GBits: 1,
		BBits: 1,
		ABits: 0, // Optional: set to 1 if you want to use alpha channel too
	}

	data, err := ExtractData(img, mask)
	if err != nil {
		// Typically means there's no stego data or we can't parse it
		log.Printf("[?] No valid LSB data in %s: %v\n", filename, err)
		return
	}

	// If we have data, check if it's mostly ASCII
	if len(data) > 0 {
		if IsASCIIPrintable(data) {
			// Potential plaintext
			fmt.Printf("[+] Possible plaintext found in %s:\n%s\n\n", filename, string(data))
		} else {
			// Could be encrypted or binary
			fmt.Printf("[*] Non-printable data found in %s (%d bytes)\n", filename, len(data))
		}
	}
}
