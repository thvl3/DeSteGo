# SteGOC2 Codebase Summary

This document provides a comprehensive summary of all Go files and functions in the SteGOC2 codebase.

## File Structure

The SteGOC2 codebase is organized into the following structure:
- `/home/thule/SteGOC2/detect-stego/` - Main package for steganography detection
- `/home/thule/SteGOC2/DeSteGo/` - Simple package that currently only contains a hello world program

## Go Files and Functions

### detect-stego/main.go

Package: main

Purpose: Entry point for the steganography detection tool. Contains the main function and command-line interface.

#### Constants
- `colorReset`, `colorRed`, `colorGreen`, etc. - ANSI color codes for terminal output
- `workerCount = 4` - Number of concurrent workers for parallel scanning

#### Functions
- `main()` - Entry point, parses command line arguments and runs the appropriate scan mode
- `printInfo()`, `printSuccess()`, `printWarning()`, `printError()`, `printAlert()` - Utility functions for colored console output
- `moveFiles()` - Moves PNG and JPG files from subdirectories to the target directory
- `removeEmptyDirsRecursively()` - Recursively removes empty directories
- `GalleryDownload()` - Downloads images from a URL using gallery-dl
- `GenerateFilename()` - Generates a random filename for downloaded files
- `GrabFromURL()` - Downloads an image from a direct image URL
- `GrabFromURLList()` - Downloads images from a list of URLs
- `gatherImageFiles()` - Finds all image files in a directory
- `scanDirectorySequential()` - Processes image files one at a time
- `scanDirectoryConcurrent()` - Processes image files in parallel
- `scanFileBuffered()` - Scans a single file and logs to a buffer
- `scanFile()` - Scans a file for steganography
- `filterFindings()` - Filters findings to reduce false positives
- `runLSBAnalysis()` - Runs LSB analysis on an image
- `runJSLSBDetection()` - Runs JavaScript LSB detection
- `runJPEGAnalysis()` - Runs JPEG-specific analysis
- `IsSuspiciousText()` - Checks if text is suspicious
- `ExtractBitsReverse()` - Extracts bits in reverse order
- `tryExtractChannelMask()` - Tries to extract data using specific channel mask
- `ExtractBitsDirectly()` - Extracts bits directly from image
- `updateResults()` - Updates scan results
- `printSummary()` - Prints summary of scan results
- `IsLikelyC2Traffic()` - Checks if text appears to be C2 traffic
- `runStatisticalAnalysis()` - Runs the new statistical analysis
- `assessTextConfidence()` - Assesses how likely a text is to be an intentional hidden message

### detect-stego/steghide_detector.go

Package: main

Purpose: Implements detection algorithms specific to the StegHide steganography tool.

#### Constants/Variables
- `StegHideSignatures` - Byte patterns that might indicate Steghide usage

#### Types
- `StegHideStatistics` - Contains statistical information about a potential Steghide payload

#### Functions
- `DetectStegHide()` - Analyzes a JPEG file for signs of Steghide modification
- `checkForStegHideSignatures()` - Searches for Steghide signatures in the file
- `containsPatternInDCTArea()` - Checks for Steghide patterns in the DCT coefficient area
- `analyzeCoefficients()` - Analyzes DCT coefficients for signs of Steghide
- `checkEntropyDistribution()` - Examines entropy distribution in the file
- `isSuspiciousComment()` - Checks if a JPEG comment might contain Steghide data
- `fileHasStegHideSizeCharacteristics()` - Checks if file size matches Steghide patterns
- `ExtractPotentialStegHidePayload()` - Attempts to extract the embedded payload

### detect-stego/stats_analysis.go

Package: main

Purpose: Provides statistical analysis functions for LSB steganography detection.

#### Functions
- `ChiSquareLSB()` - Calculates a chi-square statistic on the distribution of even/odd values for a specific color channel
- `IsSuspiciousChiSquare()` - Returns true if the chi-square value is below a threshold indicating a suspicious distribution

### detect-stego/statistical_analysis.go

Package: main

Purpose: Implements advanced statistical analysis methods for detecting steganography.

#### Types
- `LSBDistribution` - Represents the statistical distribution of LSB values
- `ChannelStatistics` - Holds statistical information about a specific channel's LSBs

#### Functions
- `AnalyzeLSBStatistics()` - Performs advanced statistical analysis on the LSBs of an image
- `analyzeChannel()` - Performs statistical analysis on a single channel's LSB values
- `uniformityScore()` - Measures how evenly distributed the LSB values are
- `calculatePatternScore()` - Measures how much patterns repeat in LSBs
- `DetectSteganoAnomaly()` - Combines statistical measures to determine if an image likely contains steganography
- `calculateAnomalyScore()` - Computes a score indicating how likely the image contains hidden data
- `calculatePatternRepetition()` - Measures how much patterns repeat in LSBs
- `calculateDifferenceDistribution()` - Analyzes differences between adjacent pixels
- `abs()` - Returns the absolute value of an integer

### detect-stego/scan_results.go

Package: main

Purpose: Defines types for storing and tracking scan results.

#### Types
- `ScanResults` - Tracks findings from the steganography scan

### detect-stego/progress_tracker.go

Package: main

Purpose: Implements a progress tracking system for displaying scan progress.

#### Types
- `ProgressTracker` - Manages multiple concurrent progress bars
- `ProgressBar` - Represents a single progress operation

#### Functions
- `NewProgressTracker()` - Creates a new progress tracker
- `Start()` - Begins tracking a new operation
- `Update()` - Updates the progress of an operation
- `Complete()` - Marks an operation as complete and removes it from display
- `render()` - Displays all progress bars
- `GetProgressCallback()` - Returns a callback function that updates the tracker

### detect-stego/metadata_analysis.go

Package: main

Purpose: Provides functions for analyzing metadata in files to distinguish normal metadata from hidden content.

#### Functions
- `IsMetadataString()` - Checks if a string is likely standard metadata rather than hidden text
- `isHexString()` - Checks if a string is primarily hexadecimal
- `FilterHiddenText()` - Takes detected text and filters out normal metadata

### detect-stego/lsb_detector.go

Package: main

Purpose: Implements LSB (Least Significant Bit) steganography detection algorithms.

#### Types
- `LSBDistributionL` - The LSB distribution for basic detection
- `ChannelLSBStats` - Holds LSB statistics for a specific channel

#### Functions
- `DetectJSLSB()` - Tries to determine if the image contains data hidden using JavaScript LSB algorithm
- `AnalyzeLSBDistribution()` - Analyzes the distribution of LSB values across channels
- `IsLikelyASCII()` - Checks if the data looks like ASCII text
- `GetLSBMask()` - Returns a ChannelMask suitable for the JavaScript LSB implementation
- `TryExtractJSLSB()` - Attempts to extract data using the JavaScript LSB algorithm
- `extractBitsDirectly()` - Extracts bits directly from the image without assuming a length prefix

### detect-stego/jpeg_utils.go

Package: main

Purpose: Provides utility types for JPEG steganography analysis.

#### Types
- `StegResults` - Contains steganography analysis results

### detect-stego/jpeg_stego_extractor.go

Package: main

Purpose: Implements functions for extracting steganographic content from JPEG files.

#### Types
- `StegoExtractionResult` - Holds the result of an extraction attempt

#### Functions
- `ExtractJPEGSteganography()` - Attempts to extract steganographic content from a JPEG file
- `extractUsingDCTAnalysis()` - Extracts data using DCT coefficient analysis
- `extractUsingStegHide()` - Tries to extract data using specialized StegHide detection
- `extractAppendedData()` - Creates a result from data appended after the JPEG EOF marker
- `extractUsingExternalTools()` - Attempts to use external steganography tools if available
- `hasStegHideCommand()` - Checks if the steghide command-line tool is available
- `tryExtractUsingStegHideCommand()` - Attempts extraction using the steghide command
- `hasOutguessCommand()` - Checks if the outguess command-line tool is available
- `tryExtractUsingOutguessCommand()` - Attempts extraction using the outguess command
- `analyzeExtractedData()` - Performs detailed analysis on extracted data to determine its nature
- `saveExtractedData()` - Saves the data to a file with an appropriate extension

### detect-stego/jpeg_steganalysis.go

Package: main

Purpose: Implements steganalysis algorithms for detecting steganography in JPEG files.

#### Constants
- `markerSOI = 0xFFD8` - Start of Image JPEG marker

#### Types
- `StegAnalysisResult` - Contains the detection results

#### Functions
- `AnalyzeJPEG()` - Performs comprehensive steganalysis on a JPEG file
- `detectJSteg()` - Implements the JSteg detection algorithm
- `calculateDifferenceEntropy()` - Analyzes the entropy of differences between adjacent values
- `detectSequentialPatterns()` - Looks for suspicious sequential patterns in coefficient values
- `analyzeFrequencyDistribution()` - Examines the frequency distribution of coefficient values
- `hasJStegLSBPattern()` - Checks for LSB patterns characteristic of JSteg
- `detectF5()` - Implements the F5 steganography detection algorithm
- `detectOutGuess()` - Implements the OutGuess detection algorithm
- `detectJPHide()` - Implements the JPHide detection algorithm
- Additional helper functions for analyzing JPEG data

### detect-stego/jpeg_dct_parser.go

Package: main

Purpose: Implements parsing and analysis of DCT (Discrete Cosine Transform) coefficients in JPEG files.

#### Types
- `DCTCoefficientBlock` - Represents an 8×8 block of DCT coefficients
- `JPEGDCTData` - Holds all DCT coefficient data from a JPEG file
- `HuffmanTable` - Represents a JPEG Huffman table

#### Functions
- `ParseJPEGDCTCoefficients()` - Extracts DCT coefficients from a JPEG file
- `extractEntropyCodedData()` - Extracts the bitstream between SOS and the next marker
- `decodeDCTCoefficients()` - Decodes the entropy-coded data to extract DCT coefficients
- `AnalyzeDCTCoefficientHistogram()` - Analyzes the histogram of DCT coefficients to detect anomalies
- `DetectStegoByCoefficientAnalysis()` - Analyzes DCT coefficients to detect steganography
- `ExtractSteganographicData()` - Attempts to extract hidden data from DCT coefficients
- `extractJStegData()` - Extracts data embedded using the JSteg algorithm
- `extractF5Data()` - Attempts to extract data embedded using the F5 algorithm
- `extractStegHideData()` - Attempts to extract data embedded using the StegHide algorithm
- `extractOutGuessData()` - Attempts to extract data embedded using the OutGuess algorithm
- `extractGenericLSB()` - Attempts a generic LSB extraction from coefficients
- `convertBitsToBytes()` - Converts a slice of bits to bytes
- `GetStegoCoefficientCount()` - Estimates how many coefficients were modified for steganography

### detect-stego/jpeg_analyzer.go

Package: main

Purpose: Provides functions for analyzing JPEG files to detect steganography.

#### Constants
- JPEG markers: `markerPrefix`, `jMarkerSOI`, `markerAPP0`, etc.

#### Types
- `JPEGMetadata` - Holds information extracted from a JPEG file's structure
- `segment` - Represents a JPEG segment

#### Functions
- `ExtractJPEGMetadata()` - Analyzes a JPEG file and extracts its metadata
- `DetectJPEGSteganography()` - Analyzes JPEG metadata to detect signs of steganography
- `containsEncodedText()` - Checks if the string might contain encoded data
- `textEntropy()` - Calculates the Shannon entropy of a text string
- `logBase2()` - Calculates log base 2 of a value
- `hasAbnormalMarkerSequence()` - Checks for unusual marker sequences
- `hasModifiedQuantizationValues()` - Checks if a quantization table has been manipulated
- `DetectJSteg()` - Checks for signs of JSteg steganography
- `DetectF5()` - Checks for signs of F5 steganography
- `DetectOutguess()` - Checks for signs of Outguess steganography
- `CheckAppendedData()` - Checks if there is data appended after the JPEG EOI marker
- `ExtractAppendedData()` - Extracts any data appended after the EOI marker
- `containsEncodedBytes()` - Checks if byte array might contain encoded data
- `ScanForPlaintextStego()` - Searches for plaintext hidden in various JPEG segments
- Multiple helper functions for JPEG analysis and text detection

### detect-stego/jpeg_analysis.go

Package: main

Purpose: Provides functions for analyzing JPEG files, particularly focusing on quantization tables.

#### Types
- `DQT` - Represents a JPEG quantization table

#### Functions
- `analyzeQuantizationTables()` - Examines quantization tables but no longer flags them as suspicious
- `getStandardQuantizationTables()` - Returns standard JPEG quantization tables for information

### detect-stego/false_positive_reduction.go

Package: main

Purpose: Implements methods to reduce false positives in steganography detection.

#### Types
- `FalsePositiveCheck` - Performs additional checks to reduce false positives

#### Functions
- `NewFalsePositiveCheck()` - Creates a new instance for false positive reduction
- `EvaluateDetection()` - Examines findings and returns true if likely a false positive
- `CalculateImageComplexity()` - Returns a measure of image complexity (0-1)
- `calculateLocalVariance()` - Computes intensity variance in a local region

### detect-stego/detection_config.go

Package: main

Purpose: Defines configuration and threshold settings for steganography detection.

#### Types
- `DetectionThresholds` - Contains configurable settings for steganography detection

#### Variables
- `CurrentConfig` - Holds the active detection configuration

#### Functions
- `DefaultDetectionConfig()` - Returns the default detection configuration
- `Initialize()` - Applies custom configuration settings

### detect-stego/convert.go

Package: main

Purpose: Provides functions for converting images between formats, particularly JPEG to PNG.

#### Functions
- `ConvertToPNG()` - Converts a JPEG file to PNG and saves it with the same name but .png extension
- `ConvertAllJPEGs()` - Finds all JPEG files in a directory and converts them to PNG
- `LoadAndConvertJPEG()` - Loads a JPEG file and returns it as an image.Image

### detect-stego/types.go (Empty file)

Package: main

Purpose: File exists but has no content. Likely intended for type definitions.

### detect-stego/utils.go (Empty file)

Package: main

Purpose: File exists but has no content. Likely intended for utility functions.

### DeSteGo/main.go

Package: main

Purpose: Simple Hello World program, possibly a placeholder for future development.

#### Functions
- `main()` - Prints "Hello, World!"

## Core Functionality

The SteGOC2 codebase is primarily focused on detecting various steganography techniques in image files. The main components are:

1. **LSB (Least Significant Bit) Detection** - Analyzes pixel data to detect hidden messages in the least significant bits of image data
2. **JPEG-specific Analysis** - Specialized analysis for JPEG files, including DCT coefficient analysis, quantization table examination, and detection of appended data
3. **Statistical Analysis** - Advanced statistical methods to detect anomalies that might indicate steganography
4. **Multiple Algorithm Detection** - Detection for specific steganography tools including StegHide, JSteg, F5, and OutGuess
5. **Command-line Interface** - User interface for scanning individual files or directories of images
6. **URL Processing** - Capability to download images from URLs or galleries and scan them
7. **False Positive Reduction** - Methods to minimize false positive detections

## Key Algorithms

The codebase implements several key algorithms for steganography detection:

1. **LSB Distribution Analysis** - Examines the distribution of least significant bits to detect hidden data
2. **Chi-Square Analysis** - Statistical tests to detect anomalies in pixel value distributions
3. **DCT Coefficient Analysis** - For JPEG files, analyzes the Discrete Cosine Transform coefficients
4. **Entropy Measurement** - Calculates Shannon entropy to detect unusual randomness
5. **Pattern Detection** - Looks for suspicious patterns that might indicate steganography
6. **Text Extraction** - Attempts to extract hidden plaintext from various parts of files
