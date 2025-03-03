package filehandler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

/*
File explanation:
This file contains utility functions for file handling, such as detecting file formats, reading files, downloading files, and saving files.
The DetectFileFormat function detects the format of a file by checking the extension and content type.
The ReadFileBytes function reads a file and returns its content as a byte array.
The IsURL function checks if a string is a URL.
The DownloadFile function downloads a file from a URL and saves it to a temporary file.
The SaveFile function saves data to a file.
The FilesInDirectory function returns a list of files in a directory with the given extensions.
*/

// SupportedImageFormats is a map of file extensions to their format names
var SupportedImageFormats = map[string]string{
	".png":  "png",
	".jpg":  "jpeg",
	".jpeg": "jpeg",
	".gif":  "gif",
	".bmp":  "bmp",
	".webp": "webp",
	".svg":  "svg",
}

// DetectFileFormat detects the format of a file
func DetectFileFormat(filePath string) (string, error) {
	// First check extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if format, ok := SupportedImageFormats[ext]; ok {
		return format, nil
	}

	// If extension not recognized, try to detect by content
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	contentType := http.DetectContentType(buffer)

	// Map content types to our formats
	switch {
	case strings.Contains(contentType, "image/png"):
		return "png", nil
	case strings.Contains(contentType, "image/jpeg"):
		return "jpeg", nil
	case strings.Contains(contentType, "image/gif"):
		return "gif", nil
	case strings.Contains(contentType, "image/bmp"):
		return "bmp", nil
	case strings.Contains(contentType, "image/webp"):
		return "webp", nil
	case strings.Contains(contentType, "image/svg+xml"):
		return "svg", nil
	default:
		return "", fmt.Errorf("unsupported file format: %s", contentType)
	}
}

// ReadFileBytes reads a file and returns its content as a byte array
func ReadFileBytes(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	size := info.Size()
	if size > 100*1024*1024 { // 100MB limit
		return nil, fmt.Errorf("file too large (max 100MB)")
	}

	content := make([]byte, size)
	_, err = io.ReadFull(file, content)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}

// IsURL checks if the given string is a URL
func IsURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// DownloadFile downloads a file from a URL and saves it to a temporary file
func DownloadFile(url string) (string, error) {
	// Create a temporary file
	tempDir := os.TempDir()
	fileName := fmt.Sprintf("stegoc2_download_%d", time.Now().UnixNano())
	tempFilePath := filepath.Join(tempDir, fileName)

	// Create output file
	out, err := os.Create(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Send GET request
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Check content length
	if resp.ContentLength > 100*1024*1024 { // 100MB limit
		return "", fmt.Errorf("file too large (max 100MB)")
	}

	// Write response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save downloaded file: %w", err)
	}

	return tempFilePath, nil
}

// SaveFile saves data to a file
func SaveFile(data []byte, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create or truncate the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write data to file
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// FilesInDirectory returns a list of files in a directory with the given extensions
func FilesInDirectory(dirPath string, extensions []string) ([]string, error) {
	var files []string

	// Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dirPath)
	}

	// Walk the directory
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check file extension
		if len(extensions) > 0 {
			ext := strings.ToLower(filepath.Ext(path))
			for _, validExt := range extensions {
				if ext == validExt {
					files = append(files, path)
					break
				}
			}
		} else {
			// If no extensions provided, include all files
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}
