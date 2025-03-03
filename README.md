# DeSteGo v1.0.0

DeSteGo is a comprehensive steganography analysis tool designed to detect hidden data across multiple file formats written by Ethan Hulse.

## Overview

DeSteGo provides tools to analyze files for potential steganography techniques. It supports multiple file formats and uses a modular architecture to allow for extensible analysis capabilities.

## Features

- **Multiple Input Methods**: Analyze individual files, directories, URLs, or lists of URLs
- **Format Detection**: Automatically detects file formats or allows manual format specification
- **Modular Analysis**: Uses specialized analyzers for different file formats
- **Comprehensive Results**: Displays detection score, confidence level, findings, and recommendations
- **Extraction Support**: Option to attempt extraction of hidden data
- **Color-coded Output**: Easy-to-read terminal output with color highlighting

## Installation

```bash
# Clone the repository
git clone https://github.com/thvl3/DeSteGo.git

# Change to the project directory
cd DeSteGo

# Build the application
go build -o destego ./cmd/destego
```

## Usage

### Basic Commands

```bash
# Analyze a single file
./destego -file path/to/file.png

# Analyze all files in a directory
./destego -dir path/to/directory

# Download and analyze a file from a URL
./destego -url https://example.com/image.jpg

# Analyze multiple files from a list of URLs
./destego -urlfile path/to/urls.txt
```

### Command-Line Options

| Option | Description |
|--------|-------------|
| `-file <path>` | Path to a single file for analysis |
| `-dir <path>` | Path to directory containing files for analysis |
| `-url <url>` | URL to download and analyze |
| `-urlfile <path>` | Path to file containing URLs to download and analyze |
| `-outdir <path>` | Directory to store results and downloaded files (default: "destego_output") |
| `-format <format>` | Force specific format analysis (png, jpg, gif, svg) (default: "auto") |
| `-verbose` | Enable verbose output |
| `-listformats` | List all supported file formats |
| `-seq` | Use sequential processing (default: true) |
| `-extract` | Attempt to extract hidden data if found |

## Understanding Results

DeSteGo provides a detailed analysis with the following information:

- **Detection Score**: A value between 0.0 and 1.0 indicating the likelihood of steganography
  - 0.0-0.2: No steganography detected
  - 0.2-0.5: LOW probability
  - 0.5-0.8: MEDIUM probability
  - 0.8-1.0: HIGH probability
- **Confidence**: How confident the analyzer is in its detection score (0.0-1.0)
- **Possible Algorithm**: If detected, the likely steganography algorithm used
- **Findings**: Specific anomalies or patterns found during analysis
- **Recommendations**: Suggested next steps for further analysis or extraction

## Examples

### Analyzing a Single File

```bash
./destego -file suspicious_image.png -verbose
```

### Analyzing a Directory of Images

```bash
./destego -dir ./images/ -outdir ./analysis_results
```

### Downloading and Analyzing from the Web

```bash
./destego -url https://example.com/suspicious.jpg -extract
```

### Processing Multiple URLs

Create a file (e.g., `urls.txt`) containing one URL per line:

```
https://example.com/image1.png
https://example.com/image2.jpg
# This is a comment line
https://example.com/image3.gif
```

Then run:

```bash
./destego -urlfile urls.txt -verbose -extract
```

## Supported File Formats

Run `./destego -listformats` to see all supported file formats and their corresponding analyzers.

Current support includes:
- PNG
- JPEG/JPG

## Contributing

Contributions are welcome! The DeSteGo architecture is designed to be modular, making it easy to add support for new file formats or steganography detection techniques.

## License

[License information]