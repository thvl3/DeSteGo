package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// ConvertToPNG converts a JPEG file to PNG and saves it with the same name but .png extension
func ConvertToPNG(filename string) (string, error) {
	// Read the source file
	imgData, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Decode the image
	img, err := jpeg.Decode(bytes.NewReader(imgData))
	if err != nil {
		return "", fmt.Errorf("failed to decode JPEG: %v", err)
	}

	// Create new filename with .png extension
	newFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".png"

	// Create the output file
	outFile, err := os.Create(newFilename)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Encode as PNG
	if err := png.Encode(outFile, img); err != nil {
		os.Remove(newFilename) // Clean up on error
		return "", fmt.Errorf("failed to encode PNG: %v", err)
	}

	return newFilename, nil
}

// ConvertAllJPEGs finds all JPEG files in a directory and converts them to PNG
func ConvertAllJPEGs(dirPath string) ([]string, error) {
	var convertedFiles []string

	// Walk through directory
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is JPEG
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".jpg" || ext == ".jpeg" {
			// Convert the file
			newFile, err := ConvertToPNG(path)
			if err != nil {
				printError("Failed to convert %s: %v\n", path, err)
				return nil // Continue with other files
			}

			convertedFiles = append(convertedFiles, newFile)
			printSuccess("Converted %s -> %s\n", path, filepath.Base(newFile))
		}

		return nil
	})

	return convertedFiles, err
}

// LoadAndConvertJPEG loads a JPEG file and returns it as an image.Image
func LoadAndConvertJPEG(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode the JPEG
	img, err := jpeg.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG: %v", err)
	}

	return img, nil
}
