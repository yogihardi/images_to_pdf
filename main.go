package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"strings"

	v2 "github.com/johnfercher/maroto/v2"
	marotoimage "github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/spf13/cobra"
)

var (
	inputDir  string
	outputDir string
	pdfName   string
)

var rootCmd = &cobra.Command{
	Use:   "images-to-pdf",
	Short: "Convert images from a folder to a single PDF document",
	Long: `A CLI tool that reads all image files from an input folder,
sorts them by name, and combines them into a single PDF file with each image on its own page.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := convertImagesToPDF(inputDir, outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().StringVarP(&inputDir, "input", "i", "", "Input directory containing images (required)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for the PDF file (default: current directory)")
	rootCmd.Flags().StringVarP(&pdfName, "name", "n", "images.pdf", "Name of the output PDF file (default: images.pdf)")
	rootCmd.MarkFlagRequired("input")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func convertImagesToPDF(inputDir, outputDir string) error {
	// Validate input directory
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", inputDir)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Find all image files
	imageFiles, err := findImageFiles(inputDir)
	if err != nil {
		return fmt.Errorf("failed to find image files: %v", err)
	}

	if len(imageFiles) == 0 {
		return fmt.Errorf("no image files found in directory: %s", inputDir)
	}

	// Sort files by name
	sort.Strings(imageFiles)

	fmt.Printf("Found %d image files, converting to PDF...\n", len(imageFiles))

	// Step 0: Convert images to optimized JPEG
	convertedImageFiles, err := convertImagesToOptimizedJPEG(imageFiles, outputDir)
	if err != nil {
		return fmt.Errorf("failed to convert images to optimized JPEG: %v", err)
	}
	defer cleanupConvertedImages(convertedImageFiles)

	// Step 1: Calculate average image dimensions
	avgWidth, avgHeight, err := calculateAverageImageSize(convertedImageFiles)
	if err != nil {
		return fmt.Errorf("failed to calculate average image size: %v", err)
	}

	fmt.Printf("Average image dimensions: %.1fx%.1f pixels\n", avgWidth, avgHeight)

	dpiValue := float64(200)
	// Step 2: Create PDF document with DPI value and enhanced compression
	pageWidthPoints := avgWidth * 72 / dpiValue // Convert from given DPI to points
	pageHeightPoints := avgHeight * 72 / dpiValue

	// Enhanced PDF compression settings
	cfg := config.NewBuilder().
		WithDimensions(pageWidthPoints, pageHeightPoints).
		WithLeftMargin(0).
		WithTopMargin(0).
		WithRightMargin(0).
		WithBottomMargin(0).
		WithCompression(true). // Enable PDF compression
		WithSequentialLowMemoryMode(8). // More aggressive memory optimization
		Build()
	m := v2.New(cfg)

	fmt.Printf("%f DPI quality with 100%% page size (%.1fx%.1f points)\n", dpiValue, pageWidthPoints, pageHeightPoints)

	// Step 3: Add each converted image to fit full page
	for i, imagePath := range convertedImageFiles {
		fmt.Printf("Processing image %d/%d: %s\n", i+1, len(convertedImageFiles), filepath.Base(imagePath))

		// Add image that fits the full page
		imageCol := marotoimage.NewFromFileCol(12, imagePath, props.Rect{
			Center:  true,
			Percent: 100, // Use full available space
		})

		// Use the full page height for the row
		imageRow := row.New(pageHeightPoints).Add(imageCol)

		// Add the row to the document
		m.AddRows(imageRow)
	}

	// Generate output filename
	outputPath := filepath.Join(outputDir, pdfName)

	// Create PDF file
	document, err := m.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate PDF: %v", err)
	}

	// Save to file
	if err := document.Save(outputPath); err != nil {
		return fmt.Errorf("failed to save PDF to %s: %v", outputPath, err)
	}

	// Check file size and provide feedback
	if err := checkAndReportFileSize(outputPath); err != nil {
		return fmt.Errorf("failed to check file size: %v", err)
	}

	fmt.Printf("Successfully created PDF: %s\n", outputPath)
	return nil
}

func findImageFiles(dir string) ([]string, error) {
	var imageFiles []string
	supportedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".tiff": true,
		".tif":  true,
		".webp": true,
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if supportedExts[ext] {
			imageFiles = append(imageFiles, path)
		}

		return nil
	})

	return imageFiles, err
}

// calculateAverageImageSize calculates the average width and height of all images
func calculateAverageImageSize(imageFiles []string) (float64, float64, error) {
	if len(imageFiles) == 0 {
		return 0, 0, fmt.Errorf("no image files provided")
	}

	var totalWidth, totalHeight int
	var validImages int

	for _, imagePath := range imageFiles {
		file, err := os.Open(imagePath)
		if err != nil {
			fmt.Printf("Warning: Could not open image %s: %v\n", filepath.Base(imagePath), err)
			continue
		}

		imgConfig, _, err := image.DecodeConfig(file)
		file.Close()
		if err != nil {
			fmt.Printf("Warning: Could not decode image %s: %v\n", filepath.Base(imagePath), err)
			continue
		}

		totalWidth += imgConfig.Width
		totalHeight += imgConfig.Height
		validImages++
	}

	if validImages == 0 {
		return 0, 0, fmt.Errorf("no valid images found")
	}

	avgWidth := float64(totalWidth) / float64(validImages)
	avgHeight := float64(totalHeight) / float64(validImages)

	return avgWidth, avgHeight, nil
}

// checkAndReportFileSize checks the PDF file size and provides feedback
func checkAndReportFileSize(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	fileSizeBytes := fileInfo.Size()
	fileSizeMB := float64(fileSizeBytes) / (1024 * 1024)

	fmt.Printf("PDF file size: %.2f MB\n", fileSizeMB)

	const targetSizeMB = 3.0
	if fileSizeMB > targetSizeMB {
		fmt.Printf("⚠️  Warning: PDF size (%.2f MB) exceeds target of %.1f MB\n", fileSizeMB, targetSizeMB)
		fmt.Printf("Suggestions to reduce size:\n")
		fmt.Printf("  • Use JPEG images instead of PNG for photos\n")
		fmt.Printf("  • Reduce image resolution before processing\n")
		fmt.Printf("  • Consider processing fewer images per PDF\n")
	} else {
		fmt.Printf("✅ PDF size is within the %.1f MB target\n", targetSizeMB)
	}

	return nil
}

// convertToJPEG converts a single image to JPEG format with adaptive quality compression
func convertToJPEG(imagePath, outputDir string) (string, error) {
	// Open and decode the source image
	srcFile, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	img, _, err := image.Decode(srcFile)
	if err != nil {
		return "", err
	}

	// Scale image to 800px width with proportional height
	img = scaleImageToWidth(img, 800)

	// Generate output filename
	baseName := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
	outputPath := filepath.Join(outputDir, baseName+".jpg")

	// Get image dimensions for adaptive quality
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	totalPixels := width * height

	// Calculate adaptive quality based on image size
	quality := calculateAdaptiveQuality(totalPixels)

	// Try compression with iterative quality reduction if needed
	finalPath, err := compressImageWithTargetSize(img, outputPath, quality)
	if err != nil {
		return "", err
	}

	return finalPath, nil
}

// isJPEGFile checks if the file is already a JPEG
func isJPEGFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".jpg" || ext == ".jpeg"
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	return err
}

// cleanupConvertedImages removes the temporary converted image files
func cleanupConvertedImages(convertedFiles []string) {
	if len(convertedFiles) == 0 {
		return
	}

	// Get the temp directory from the first file
	tempDir := filepath.Dir(convertedFiles[0])

	// Remove the entire temp directory
	if err := os.RemoveAll(tempDir); err != nil {
		fmt.Printf("Warning: Failed to cleanup temp directory %s: %v\n", tempDir, err)
	} else {
		fmt.Printf("Cleaned up temporary converted images\n")
	}
}

// calculateAdaptiveQuality determines optimal JPEG quality based on image characteristics
func calculateAdaptiveQuality(totalPixels int) int {
	baseQuality := 85 // Start with high quality

	// Adjust quality based on image size
	if totalPixels > 4000000 { // Very large images (4MP+)
		baseQuality = 75 // More compression for large images
	} else if totalPixels > 2000000 { // Large images (2MP+)
		baseQuality = 80
	} else if totalPixels > 1000000 { // Medium images (1MP+)
		baseQuality = 85
	} else { // Small images
		baseQuality = 90 // Less compression to preserve detail
	}

	// Ensure quality is within valid range
	if baseQuality > 95 {
		baseQuality = 95
	} else if baseQuality < 60 {
		baseQuality = 60
	}

	return baseQuality
}

// compressImageWithTargetSize compresses image with iterative quality adjustment
func compressImageWithTargetSize(img image.Image, outputPath string, startQuality int) (string, error) {
	const maxFileSize = 500 * 1024 // 500KB per image target
	quality := startQuality

	for attempts := 0; attempts < 4; attempts++ {
		// Create output file
		outFile, err := os.Create(outputPath)
		if err != nil {
			return "", err
		}

		// Encode with current quality
		options := &jpeg.Options{Quality: quality}
		err = jpeg.Encode(outFile, img, options)
		outFile.Close()

		if err != nil {
			return "", err
		}

		// Check file size
		fileInfo, err := os.Stat(outputPath)
		if err != nil {
			return "", err
		}

		fileSize := fileInfo.Size()

		// If size is acceptable or quality is already very low, accept it
		if fileSize <= maxFileSize || quality <= 50 {
			if attempts > 0 {
				fmt.Printf("    → Compressed to %d KB (quality: %d)\n", fileSize/1024, quality)
			}
			return outputPath, nil
		}

		// Reduce quality for next attempt
		quality -= 15
		if quality < 50 {
			quality = 50
		}
	}

	return outputPath, nil
}

// convertImagesToOptimizedJPEG applies efficient compression while maintaining PDF readability
func convertImagesToOptimizedJPEG(imageFiles []string, outputDir string) ([]string, error) {
	var convertedFiles []string
	tempDir := filepath.Join(outputDir, "temp_optimized_images")

	// Create temporary directory for converted images
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}

	fmt.Printf("Applying efficient compression while maintaining PDF readability...\n")

	for i, imagePath := range imageFiles {
		fmt.Printf("Optimizing %d/%d: %s\n", i+1, len(imageFiles), filepath.Base(imagePath))

		convertedPath, err := convertToEfficientCompression(imagePath, tempDir)
		if err != nil {
			fmt.Printf("Warning: Failed to optimize image %s: %v\n", filepath.Base(imagePath), err)
			continue
		}
		convertedFiles = append(convertedFiles, convertedPath)
	}

	fmt.Printf("Successfully optimized %d images for PDF readability\n", len(convertedFiles))
	return convertedFiles, nil
}

// scaleImageToWidth scales an image to a specific width while maintaining aspect ratio
func scaleImageToWidth(img image.Image, targetWidth int) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Max.X - bounds.Min.X
	srcHeight := bounds.Max.Y - bounds.Min.Y

	// If image is already smaller than target width, keep original size
	if srcWidth <= targetWidth {
		return img
	}

	// Calculate proportional height
	scale := float64(targetWidth) / float64(srcWidth)
	targetHeight := int(float64(srcHeight) * scale)

	// Create new scaled image
	scaled := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Simple scaling using nearest neighbor
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)

			// Ensure we don't go out of bounds
			if srcX >= srcWidth {
				srcX = srcWidth - 1
			}
			if srcY >= srcHeight {
				srcY = srcHeight - 1
			}

			scaled.Set(x, y, img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}

	return scaled
}

// convertToEfficientCompression applies the most efficient compression for PDF readability
func convertToEfficientCompression(imagePath, outputDir string) (string, error) {
	// Open and analyze the source image
	srcFile, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	img, _, err := image.Decode(srcFile)
	if err != nil {
		return "", err
	}

	// Scale image to 800px width with proportional height
	img = scaleImageToWidth(img, 800)

	// Analyze image characteristics
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	totalPixels := width * height

	// Get original file info
	originalInfo, _ := os.Stat(imagePath)
	originalSize := originalInfo.Size()

	// Determine optimal compression strategy
	strategy := determineCompressionStrategy(totalPixels, originalSize, imagePath)

	baseName := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
	var outputPath string
	var finalSize int64

	switch strategy {
	case "keep_original":
		// Keep original if it's already optimal
		outputPath = filepath.Join(outputDir, filepath.Base(imagePath))
		err = copyFile(imagePath, outputPath)
		finalSize = originalSize

	case "optimize_jpeg":
		// Convert to optimized JPEG for better PDF compression
		outputPath = filepath.Join(outputDir, baseName+".jpg")
		err = compressToOptimalJPEG(img, outputPath, totalPixels)
		if fileInfo, statErr := os.Stat(outputPath); statErr == nil {
			finalSize = fileInfo.Size()
		}

	case "convert_png_to_jpeg":
		// Convert PNG photos to JPEG (better for PDF)
		outputPath = filepath.Join(outputDir, baseName+".jpg")
		err = convertPNGToOptimalJPEG(img, outputPath, totalPixels)
		if fileInfo, statErr := os.Stat(outputPath); statErr == nil {
			finalSize = fileInfo.Size()
		}

	default:
		// Fallback to original
		outputPath = filepath.Join(outputDir, filepath.Base(imagePath))
		err = copyFile(imagePath, outputPath)
		finalSize = originalSize
	}

	if err != nil {
		return "", err
	}

	// Report compression results
	compressionRatio := float64(originalSize-finalSize) / float64(originalSize) * 100
	if compressionRatio > 0 {
		fmt.Printf("    → %s: %d KB → %d KB (%.1f%% reduction)\n",
			strategy, originalSize/1024, finalSize/1024, compressionRatio)
	} else {
		fmt.Printf("    → %s: %d KB (kept original)\n", strategy, originalSize/1024)
	}

	return outputPath, nil
}

// determineCompressionStrategy analyzes image and determines best compression approach
func determineCompressionStrategy(totalPixels int, originalSize int64, imagePath string) string {
	ext := strings.ToLower(filepath.Ext(imagePath))

	// For very small files, keep original
	if originalSize < 50*1024 { // Less than 50KB
		return "keep_original"
	}

	// For already small JPEG files, keep them
	if (ext == ".jpg" || ext == ".jpeg") && originalSize < 200*1024 {
		return "keep_original"
	}

	// For PNG files that are likely photos (large with many pixels), convert to JPEG
	if ext == ".png" && totalPixels > 100000 && originalSize > 500*1024 {
		return "convert_png_to_jpeg"
	}

	// For large JPEG files, optimize them
	if (ext == ".jpg" || ext == ".jpeg") && originalSize > 300*1024 {
		return "optimize_jpeg"
	}

	// For other large files, convert to optimized JPEG
	if originalSize > 400*1024 {
		return "optimize_jpeg"
	}

	// Default: keep original for small/medium files
	return "keep_original"
}

// compressToOptimalJPEG compresses image to JPEG with optimal settings for PDF readability
func compressToOptimalJPEG(img image.Image, outputPath string, totalPixels int) error {
	// Determine optimal quality based on image characteristics
	quality := 75 // Start with high quality for readability

	// Adjust quality based on image size for optimal PDF readability
	if totalPixels > 2000000 { // Very large images (2MP+)
		quality = 65 // Acceptable compression for large images
	} else if totalPixels > 1000000 { // Large images (1MP+)
		quality = 70 // Good balance
	} else if totalPixels < 300000 { // Small images
		quality = 80 // Preserve quality for small images
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Encode with optimal settings
	options := &jpeg.Options{Quality: quality}
	return jpeg.Encode(outFile, img, options)
}

// convertPNGToOptimalJPEG converts PNG to JPEG with optimal settings for PDF
func convertPNGToOptimalJPEG(img image.Image, outputPath string, totalPixels int) error {
	bounds := img.Bounds()

	// Create a new image without alpha channel for JPEG conversion
	rgbImg := image.NewRGBA(bounds)

	// Fill with white background and draw original image
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgbImg.Set(x, y, color.RGBA{255, 255, 255, 255}) // White background
		}
	}

	// Draw original image on white background with alpha blending
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, a := originalColor.RGBA()

			// Alpha blending with white background
			if a > 0 {
				alpha := float64(a) / 65535.0
				newR := uint8(float64(r>>8)*alpha + 255*(1-alpha))
				newG := uint8(float64(g>>8)*alpha + 255*(1-alpha))
				newB := uint8(float64(b>>8)*alpha + 255*(1-alpha))
				rgbImg.Set(x, y, color.RGBA{newR, newG, newB, 255})
			}
		}
	}

	// Use higher quality for PNG conversions to maintain readability
	quality := 88
	if totalPixels > 2000000 {
		quality = 82
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Encode with optimal settings
	options := &jpeg.Options{Quality: quality}
	return jpeg.Encode(outFile, rgbImg, options)
}
