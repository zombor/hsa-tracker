package scanning

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Gemini implements the Scanner interface using Google Gemini
type Gemini struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// NewGemini creates a new Gemini Scanner instance
func NewGemini(apiKey string, modelName string) (*Gemini, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}
	if modelName == "" {
		modelName = "gemini-2.5-pro"
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", err)
	}

	model := client.GenerativeModel(modelName)

	return &Gemini{
		client: client,
		model:  model,
	}, nil
}

// ScanReceipt analyzes a receipt and extracts metadata
func (g *Gemini) ScanReceipt(imageData []byte, contentType string) (*ReceiptData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Prepare image data (convert to PNG if needed)
	finalImageData, _, _, err := prepareImageData(imageData, contentType)
	if err != nil {
		return nil, err
	}

	// genai.ImageData expects just the format suffix (e.g., "png"), not the full MIME type (e.g., "image/png")
	// After prepareImageData, everything is PNG, so we always use "png"
	parts := []genai.Part{
		genai.ImageData("png", finalImageData),
		genai.Text(receiptScanPrompt),
	}

	// Generate response
	resp, err := g.model.GenerateContent(ctx, parts...)
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from gemini")
	}

	// Extract text response
	var responseText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		// Check if part is text
		if text, ok := part.(genai.Text); ok {
			responseText.WriteString(string(text))
		}
	}

	// Parse JSON response
	text := strings.TrimSpace(responseText.String())
	// Remove markdown code blocks if present
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSpace(text)

	data, err := parseReceiptJSON(text)
	if err != nil {
		return nil, fmt.Errorf("parsing receipt data: %w", err)
	}

	return data, nil
}

// Close closes the Gemini client
func (g *Gemini) Close() error {
	return g.client.Close()
}
