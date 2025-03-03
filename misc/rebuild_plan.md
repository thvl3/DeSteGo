# SteGOC2 Rebuild Plan

This document outlines the process of rebuilding the SteGOC2 steganography detection tool with better organization, modular architecture, and a modern UI using Wails.

## 1. Current State Analysis

The current codebase has several issues:

- **Code Organization**: Most functionality is in a single package with many files
- **Redundant Functions**: Multiple functions performing similar operations
- **Inconsistent Placement**: Related functions are scattered across different files
- **No Clear API**: Lack of well-defined interfaces between components
- **CLI-only Interface**: Limited to command-line usage
- **Limited File Type Support**: Primarily focused on PNG/JPG files

## 2. New Architecture

### 2.1. Package Structure

We'll reorganize the code into distinct packages with a modular approach to support multiple file types:

```
/DeSteGo/
├── cmd/                           # Command-line tools
│   └── stegoc2/                   # Main CLI app
├── internal/                      # Private implementation details
│   ├── common/                    # Shared utilities & types
│   └── analysis/                  # Analysis results & reporting
├── pkg/                           # Public packages for potential reuse
│   ├── analyzer/                  # Root analyzer package with common interfaces
│   │   ├── image/                # Base image analysis functionality
│   │   │   ├── lsb/             # LSB detection algorithms
│   │   │   ├── statistics/      # Statistical analysis tools
│   │   │   ├── jpeg/            # JPEG-specific detection
│   │   │   ├── png/             # PNG-specific detection
│   │   │   ├── gif/             # GIF-specific detection
│   │   │   ├── svg/             # SVG-specific detection
│   │   │   ├── webp/            # WEBP-specific detection
│   │   │   └── bmp/             # BMP-specific detection
│   │   └── document/            # Document file analysis (future expansion)
│   ├── extractor/                # Data extraction framework
│   │   ├── image/               # Image extraction base
│   │   │   ├── lsb/            # LSB extraction strategies
│   │   │   ├── jpeg/           # JPEG-specific extraction
│   │   │   ├── png/            # PNG-specific extraction
│   │   │   ├── gif/            # GIF-specific extraction
│   │   │   ├── svg/            # SVG-specific extraction
│   │   │   ├── webp/           # WEBP-specific extraction
│   │   │   └── bmp/            # BMP-specific extraction
│   │   └── document/           # Document extraction (future expansion)
│   ├── filehandler/              # File & URL operations
│   │   └── formatdetection/     # File format detection
│   └── models/                   # Shared data models
├── ui/                           # Wails UI codebase
│   ├── frontend/                 # React/Vue frontend 
│   └── backend/                  # Go backend API for Wails
└── wails.json                    # Wails configuration
```

### 2.2. Key Interfaces

We'll define flexible interfaces to accommodate different file types:

```go
// pkg/analyzer/analyzer.go
type FileAnalyzer interface {
    CanAnalyze(format string) bool
    Analyze(filePath string, options AnalysisOptions) (*models.AnalysisResult, error)
    Name() string
    Description() string
    SupportedFormats() []string
}

// More specific image analyzer
type ImageAnalyzer interface {
    FileAnalyzer
    AnalyzeImage(img image.Image, options AnalysisOptions) (*models.AnalysisResult, error)
}

// pkg/extractor/extractor.go
type DataExtractor interface {
    CanExtract(format string) bool
    Extract(filePath string, options ExtractionOptions) (*models.ExtractionResult, error)
    SupportedFormats() []string
}

// More specific image extractor
type ImageExtractor interface {
    DataExtractor
    ExtractFromImage(img image.Image, options ExtractionOptions) (*models.ExtractionResult, error)
}

// pkg/models/results.go
type AnalysisResult struct {
    DetectionScore     float64
    Confidence         float64
    PossibleAlgorithm  string
    FileType           string
    Details            map[string]interface{}
    Recommendations    []string
    ExtractionHints    []ExtractionHint
}

type ExtractionHint struct {
    Algorithm     string
    Confidence    float64
    Parameters    map[string]interface{}
}
```

### 2.3. Registry System

To support dynamic loading of analyzers and extractors for different file types:

```go
// pkg/analyzer/registry.go
type AnalyzerRegistry struct {
    analyzers map[string][]FileAnalyzer
}

func (r *AnalyzerRegistry) Register(analyzer FileAnalyzer) {
    for _, format := range analyzer.SupportedFormats() {
        r.analyzers[format] = append(r.analyzers[format], analyzer)
    }
}

func (r *AnalyzerRegistry) GetAnalyzersForFormat(format string) []FileAnalyzer {
    return r.analyzers[format]
}

// Similar registry for extractors
```

## 3. Module Reorganization

### 3.1. File Type Support Framework

Create a modular framework for file type support:

- Each file type gets its own package for analysis and extraction
- Common utilities shared across file types
- Registry system to dynamically load appropriate analyzers/extractors
- File type detection to automatically select the correct modules

### 3.2. LSB Detection

Create a unified LSB framework that works across multiple image formats:

- Generic implementation for PNG, BMP
- Format-specific adaptations for JPEG, GIF, WEBP
- Unified API with common interfaces
- File format specific optimizations

### 3.3. JPEG Analysis

Specialized JPEG package with algorithm-specific implementations:

- JSteg, F5, OutGuess, StegHide as separate strategies
- DCT coefficient analysis framework
- Quantization table analysis
- Metadata analysis

### 3.4. New File Format Support

Implementation plan for additional file formats:

#### GIF Support
- Frame analysis
- Color palette manipulation detection
- Frame timing analysis
- Disposal method steganography

#### SVG Support
- XML attribute analysis
- Embedded content detection
- Path data analysis
- Metadata inspection

#### WEBP Support
- Compression analysis
- Alpha channel analysis
- Conversion artifacts detection

### 3.5. Data Extraction

Modular extraction framework:

- Format-specific extractors
- Generic approach adapters
- Success probability estimation
- Post-extraction analysis and classification

### 3.6. False Positive Reduction

Comprehensive filter system:

- Format-specific false positive profiles
- Machine learning based classifier
- Context-aware filtering
- Multiple detection agreement
- Image complexity analysis

## 4. Wails UI Implementation

### 4.1. Backend API

The API will be extended to handle all supported file formats:

```go
// ui/backend/api.go
type StegoAPI struct {
    // Services
    analyzerRegistry  *analyzer.Registry
    extractorRegistry *extractor.Registry
    fileService       *services.FileService
}

func (api *StegoAPI) ScanFile(path string, options map[string]interface{}) (*models.AnalysisResult, error) {
    // Detect file format and run appropriate analyzers
}

func (api *StegoAPI) GetSupportedFormats() map[string][]string {
    // Return map of supported formats and their capabilities
}

func (api *StegoAPI) ExtractData(path string, algorithm string, options map[string]interface{}) (*models.ExtractionResult, error) {
    // Extract using the appropriate extractor
}
```

### 4.2. Frontend Features

Create a modern, intuitive UI with:

- File type filtering and selection
- Format-specific analysis options
- Visualization appropriate to file type
- Format-specific extraction tools
- File format conversion utilities
- Batch processing with format detection
- Detailed results view with format-specific information

### 4.3. Format-Specific UI Components

Add specialized UI components for each file format:

- JPEG: DCT coefficient visualizer
- GIF: Frame-by-frame analysis view
- SVG: XML structure view with highlighting
- General: Hex viewer with anomaly highlighting

## 5. Implementation Strategy

### 5.1. Phase 1: Core Framework

1. Set up the new project structure
2. Implement the file analyzer/extractor interfaces and registry
3. Create the file format detection system
4. Develop the base image analysis package

### 5.2. Phase 2: Port Existing Functionality

1. Migrate PNG/JPEG analysis to the new structure
2. Implement the LSB analysis package
3. Implement the JPEG analysis package
4. Port existing extractors to the new framework
5. Ensure compatibility and feature parity

### 5.3. Phase 3: New File Format Support

1. Implement GIF analysis and extraction
2. Implement SVG analysis and extraction
3. Implement WEBP analysis and extraction
4. Add BMP support
5. Create tests for each format

### 5.4. Phase 4: CLI Rebuild

1. Create a new CLI interface using the modular framework
2. Add format-specific command flags
3. Implement improved reporting
4. Add batch processing capabilities

### 5.5. Phase 5: Wails UI

1. Set up the Wails project
2. Implement the backend API with format support
3. Build the core frontend components
4. Add format-specific UI elements
5. Implement visualization components

### 5.6. Phase 6: Advanced Features

1. Implement machine learning-based detection
2. Add comprehensive report generation
3. Support for additional steganography algorithms
4. Integration with external tools

## 6. Code Quality Standards

Establish standards for the new codebase:

- **Documentation**: All public APIs fully documented
- **Testing**: Minimum 80% test coverage
- **Error Handling**: Consistent error wrapping and reporting
- **Logging**: Structured logging throughout
- **Configuration**: Environment-based configuration
- **Performance**: Benchmark tests for critical paths
- **Format Support**: Clear process for adding new file formats

## 7. Migration Strategy

To ensure a smooth transition while rebuilding:

1. Build the new framework without modifying existing code
2. Create adapters to wrap existing functionality
3. Implement format detection to route to appropriate analyzers
4. Create integration tests comparing old and new implementations
5. Phase in new file formats one at a time
6. Gradually replace old code with new implementations

## 8. Timeline and Milestones

- **Month 1**: Complete core framework and interfaces
- **Month 2**: Port existing PNG/JPEG functionality
- **Month 3**: Add GIF and SVG support
- **Month 4**: Implement WEBP and other formats
- **Month 5**: CLI rebuild with format support
- **Month 6**: Basic Wails UI implementation
- **Month 7**: Format-specific UI components
- **Month 8**: Advanced features and polish

## 9. Extension Points

The new architecture will have clear extension points for:

- **New File Formats**: Standard interface to implement for new formats
- **New Detection Algorithms**: Pluggable detection strategies
- **New Extraction Methods**: Modular extraction framework
- **Custom Visualizations**: Format-specific visualization plugins
- **External Tools**: Integration points for third-party tools

## 10. Potential Challenges and Solutions

### 10.1. Performance with Multiple Formats

**Challenge**: Supporting many file formats might impact performance
**Solution**: Lazy loading of analyzers, parallel processing, and format-specific optimizations

### 10.2. Complexity of Different Formats

**Challenge**: Each format has unique characteristics requiring specialized knowledge
**Solution**: Modular design, format-specific expert classes, comprehensive documentation

### 10.3. UI Consistency Across Formats

**Challenge**: Providing a consistent UI experience across diverse file formats
**Solution**: Common result model with format-specific extensions, adaptive UI components

## 11. Conclusion

This rebuild will transform SteGOC2 from a specialized PNG/JPEG steganography detector into a comprehensive multi-format steganography analysis suite with both CLI and GUI interfaces. The modular architecture will allow for:

- Easy addition of new file formats
- Consistent API across formats
- Format-specific optimizations
- Flexible UI adaptable to different format requirements
- Clear separation of concerns
- Improved maintainability

The end result will be a powerful, extensible steganography analysis tool capable of handling a wide variety of file formats while maintaining high performance and usability.
