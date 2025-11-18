package scanning

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"  // Register GIF decoder
	_ "image/jpeg" // Register JPEG decoder
	"image/png"
	"strings"

	"github.com/gen2brain/go-fitz"
	"github.com/gen2brain/heic"
)

// receiptScanPrompt is the shared prompt used by all LLM providers for scanning receipts
const receiptScanPrompt = `You are analyzing a receipt or invoice document. Carefully read all text in the image and extract the following information:

1. **Store/Business Name**: Look for the merchant name, store name, or business name at the top of the receipt. This is usually the largest text or in a header. Examples: "Walmart", "CVS Pharmacy", "Walgreens", "Target".

2. **Date**: Find the transaction date, purchase date, or invoice date on the receipt. Convert it to ISO 8601 format (YYYY-MM-DD). Look for dates near the top or bottom of the receipt. Common formats: MM/DD/YYYY, DD/MM/YYYY, or written dates.

3. **Total Amount**: Find the final total, grand total, or amount due. This is usually at the bottom of the receipt, often labeled as "TOTAL", "Amount Due", "Grand Total", or similar. Extract only the numeric value (e.g., 42.75 for $42.75).

Return ONLY valid JSON in this exact format:
{
  "title": "Store Name - Brief Description",
  "date": "YYYY-MM-DD",
  "amount": 0.00
}

Important:
- The title should start with the actual store/business name from the receipt
- The date must be in YYYY-MM-DD format
- The amount must be a number (not a string), representing dollars and cents
- If you cannot find a field, use null for that field
- Do not include any text before or after the JSON
- Do not use markdown code blocks`

// pdfToImage converts a PDF to a PNG image
func pdfToImage(pdfData []byte) ([]byte, error) {
	doc, err := fitz.NewFromMemory(pdfData)
	if err != nil {
		return nil, fmt.Errorf("opening PDF: %w", err)
	}
	defer doc.Close()

	// Render the first page (most receipts are single page)
	// Use a high DPI for better quality (300 DPI)
	img, err := doc.Image(0)
	if err != nil {
		return nil, fmt.Errorf("rendering PDF page: %w", err)
	}

	// Encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// imageToPNG converts any image format to PNG
func imageToPNG(imageData []byte, mimeType string) ([]byte, error) {
	var img image.Image
	var err error

	// Check for HEIC/HEIF format (common on iPhones) - Go's standard image package doesn't support it
	if isHEICFormat(imageData) || isHEICMimeType(mimeType) {
		// Use pure Go HEIC decoder
		img, err = heic.Decode(bytes.NewReader(imageData))
		if err != nil {
			return nil, fmt.Errorf("decoding HEIC/HEIF image: %w", err)
		}
	} else {
		// Decode standard image formats (JPEG, PNG, GIF)
		img, _, err = image.Decode(bytes.NewReader(imageData))
		if err != nil {
			// Provide more helpful error message for unsupported formats
			if strings.Contains(err.Error(), "unknown format") || strings.Contains(err.Error(), "unsupported") {
				return nil, fmt.Errorf("unsupported image format. Supported formats: JPEG, PNG, GIF, HEIC, HEIF, PDF. Error: %w", err)
			}
			return nil, fmt.Errorf("decoding image: %w", err)
		}
	}

	// Encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// isHEICFormat checks if the image data is in HEIC/HEIF format
// HEIC files typically start with specific magic bytes
func isHEICFormat(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	// HEIC files can start with various signatures:
	// - ftyp box with brand 'heic', 'heif', 'mif1', 'msf1'
	// Check for ftyp at offset 4
	if len(data) >= 12 && string(data[4:8]) == "ftyp" {
		// Check for HEIC-related brands
		brand := string(data[8:12])
		if brand == "heic" || brand == "heif" || brand == "mif1" || brand == "msf1" {
			return true
		}
	}
	return false
}

// isHEICMimeType checks if the MIME type indicates HEIC/HEIF format
func isHEICMimeType(mimeType string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	return mimeType == "image/heic" || mimeType == "image/heif" ||
		strings.Contains(mimeType, "heic") || strings.Contains(mimeType, "heif")
}

// convertToPNG converts PDFs and non-PNG images to PNG format
// Returns the PNG data and a boolean indicating if conversion occurred
func convertToPNG(imageData []byte, mimeType string) ([]byte, bool, error) {
	if mimeType == "application/pdf" {
		pngData, err := pdfToImage(imageData)
		if err != nil {
			return nil, false, fmt.Errorf("converting PDF to image: %w", err)
		}
		return pngData, true, nil
	} else if mimeType != "image/png" || isHEICFormat(imageData) || isHEICMimeType(mimeType) {
		// Convert all non-PNG images (including HEIC) to PNG
		pngData, err := imageToPNG(imageData, mimeType)
		if err != nil {
			return nil, false, fmt.Errorf("converting image to PNG: %w", err)
		}
		return pngData, true, nil
	}
	// Already PNG, return as-is
	return imageData, false, nil
}

// prepareImageData normalizes the MIME type and converts the image to PNG if needed
// Returns the final image data, the MIME type to use, and whether conversion occurred
func prepareImageData(imageData []byte, contentType string) ([]byte, string, bool, error) {
	// Normalize MIME type (lowercase, trim whitespace)
	mimeType := strings.ToLower(strings.TrimSpace(contentType))
	if mimeType == "" {
		mimeType = "image/jpeg" // default
	}

	// Convert to PNG if needed
	finalImageData, converted, err := convertToPNG(imageData, mimeType)
	if err != nil {
		return nil, "", false, err
	}

	// After prepareImageData, the data is always PNG (either converted or already PNG)
	// So we always return "image/png" as the MIME type
	finalMimeType := "image/png"

	return finalImageData, finalMimeType, converted, nil
}
