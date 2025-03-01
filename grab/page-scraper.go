package grab

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

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

func main() {
	// Check for required arguments
	if len(os.Args) < 3 {
		fmt.Println("Usage: go-run <url> <download_directory>")
		os.Exit(1)
	}

	url := os.Args[1]
	downloadDir := os.Args[2]

	// Run gallery-dl command
	fmt.Println("Running gallery-dl...")
	cmd := exec.Command("./gallery-dl.bin", url, "-d", downloadDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Error running gallery-dl: %v", err)
	}

	// Move files
	fmt.Println("Moving image files...")
	if err := moveFiles(downloadDir); err != nil {
		log.Fatalf("Error moving files: %v", err)
	}

	// Remove empty directories recursively
	fmt.Println("Cleaning up empty directories...")
	if err := removeEmptyDirsRecursively(downloadDir); err != nil {
		log.Fatalf("Error removing empty directories: %v", err)
	}

	fmt.Println("Process completed successfully!")
}
