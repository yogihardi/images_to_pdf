# Images to PDF Converter

A command-line tool written in Go that converts images from a folder into a single PDF document. Each image is placed on its own page with optimal scaling and compression.

## Features

- **Multiple Format Support**: Supports JPEG, PNG, GIF, BMP, TIFF, and WebP image formats
- **Automatic Image Scaling**: Scales images to 800px width while maintaining aspect ratio
- **High-Quality Output**: Uses 200 DPI for crisp, professional-quality PDFs
- **Smart Compression**: Applies efficient JPEG compression while maintaining readability
- **Full Page Layout**: Images are sized to 100% of the page for maximum visual impact
- **Batch Processing**: Processes entire directories of images at once
- **Memory Efficient**: Uses sequential low-memory mode for handling large image collections

## Installation

### Prerequisites

- Go 1.23 or later

### Build from Source

```bash
git clone https://github.com/yogihardi/images_to_pdf.git
cd images_to_pdf
go build
```

This will create an executable named `images_to_pdf` (or `images_to_pdf.exe` on Windows).

## Usage

### Basic Usage

```bash
./images_to_pdf -i /path/to/images
```

### Command-Line Options

```bash
Usage:
  images_to_pdf [flags]

Flags:
  -h, --help            help for images_to_pdf
  -i, --input string    Input directory containing images (required)
  -n, --name string     Name of the output PDF file (default: images.pdf)
  -o, --output string   Output directory for the PDF file (default: current directory)
```

### Examples

**Convert all images in a folder:**
```bash
./images_to_pdf -i ./my-photos
```

**Specify custom output directory and filename:**
```bash
./images_to_pdf -i ./photos -o ./output -n "vacation-photos.pdf"
```

**Convert images from multiple subdirectories:**
```bash
./images_to_pdf -i ./project-screenshots -o ./docs -n "project-documentation.pdf"
```

## Supported Image Formats

- JPEG (.jpg, .jpeg)
- PNG (.png)
- GIF (.gif)
- BMP (.bmp)
- TIFF (.tiff, .tif)
- WebP (.webp)

## How It Works

1. **Image Discovery**: Recursively scans the input directory for supported image files
2. **Sorting**: Sorts images alphabetically by filename for consistent ordering
3. **Scaling**: Automatically scales images to 800px width while preserving aspect ratio
4. **Optimization**: Converts images to optimized JPEG format for better PDF compression
5. **PDF Generation**: Creates a PDF with 200 DPI quality, placing each image on its own page
6. **Cleanup**: Removes temporary files after PDF generation

## Output Quality

- **DPI**: 200 DPI for high-quality output suitable for both screen viewing and printing
- **Compression**: Intelligent JPEG compression that maintains visual quality while optimizing file size
- **Page Layout**: Images are centered and scaled to use 100% of the available page space
- **File Size**: Automatically reports final PDF size and provides optimization suggestions if needed

## Performance

The tool includes several optimizations for handling large image collections:

- **Memory Management**: Sequential low-memory mode prevents memory issues with large batches
- **Temporary File Handling**: Automatic cleanup of intermediate files
- **Progress Reporting**: Real-time progress updates during processing
- **Error Recovery**: Continues processing even if individual images fail to convert

## Examples of Use Cases

- **Photo Albums**: Convert family photos into shareable PDF albums
- **Documentation**: Combine screenshots into technical documentation
- **Presentations**: Create PDF presentations from image slides
- **Archives**: Digitize and organize scanned documents
- **Portfolios**: Compile artwork or design samples into professional PDFs

## Troubleshooting

### Common Issues

**"input directory does not exist"**
- Verify the path to your image directory is correct
- Use absolute paths if relative paths aren't working

**"no image files found"**
- Check that your directory contains supported image formats
- Ensure file extensions match the supported formats list

**Large file sizes**
- The tool automatically suggests optimizations for files over 3MB
- Consider resizing very large images before processing
- Use JPEG format for photographs instead of PNG when possible

### Getting Help

If you encounter issues:

1. Check that your Go installation is up to date (Go 1.23+)
2. Verify that all image files are in supported formats
3. Ensure you have write permissions in the output directory

## Development

### Dependencies

This project uses the following key dependencies:

- `github.com/johnfercher/maroto/v2` - PDF generation
- `github.com/spf13/cobra` - CLI interface

### Building

```bash
# Install dependencies
go mod tidy

# Build the application
go build

# Run tests
go test ./...

# Format code
go fmt ./...

# Check for issues
go vet ./...
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is open source. Please check the license file for details.

---

**Note**: This tool is designed for legitimate use cases such as document compilation, photo organization, and content creation. Please respect copyright and intellectual property rights when processing images.