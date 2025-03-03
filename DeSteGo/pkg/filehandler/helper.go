package filehandler

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// GatherFiles collects all files in a directory (non-recursive)
func GatherFiles(dirPath string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		filePath := filepath.Join(dirPath, entry.Name())
		files = append(files, filePath)
	}

	return files, nil
}

// ReadLines reads a file and returns its lines
func ReadLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// DownloadFromURL downloads a file from a URL to the specified directory
func DownloadFromURL(url, outputDir string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Extract filename from URL
	urlParts := strings.Split(url, "/")
	filename := urlParts[len(urlParts)-1]
	if filename == "" {
		filename = "downloaded_file"
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	// Create output file
	outputPath := filepath.Join(outputDir, filename)
	out, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

// IsImageFile checks if a file is an image based on extension
func IsImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := SupportedImageFormats[ext]
	return ok
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filePath string) (int64, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}
