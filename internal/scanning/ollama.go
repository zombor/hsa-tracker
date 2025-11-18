package scanning

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Ollama implements the Scanner interface using Ollama
type Ollama struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllama creates a new Ollama Scanner instance
// Recommended models for receipt scanning (in order of recommendation):
//   - llava:1.6 (best balance of accuracy and speed)
//   - llava:latest (general purpose vision model)
//   - qwen2-vl:7b (good OCR capabilities)
//   - bakllava (alternative vision model)
//   - llava-phi3 (smaller, faster, but less accurate)
//
// Note: Some models may struggle with PDFs - consider converting PDFs to images first
func NewOllama(baseURL string, modelName string) (*Ollama, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if modelName == "" {
		modelName = "llava" // Default to llava, a popular vision model
	}

	return &Ollama{
		baseURL: baseURL,
		model:   modelName,
		client: &http.Client{
			Timeout: 120 * time.Second, // Ollama can be slower, especially for vision models
		},
	}, nil
}

// ollamaChatRequest represents the request body for Ollama's chat API
type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Images   []string        `json:"images,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaChatResponse represents the response from Ollama's chat API
type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

// ScanReceipt analyzes a receipt and extracts metadata
func (o *Ollama) ScanReceipt(imageData []byte, contentType string) (*ReceiptData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Prepare image data (convert to PNG if needed)
	finalImageData, _, _, err := prepareImageData(imageData, contentType)
	if err != nil {
		return nil, err
	}

	// Encode image as base64
	imageBase64 := base64.StdEncoding.EncodeToString(finalImageData)

	// Prepare the request with system message for better context
	reqBody := ollamaChatRequest{
		Model:  o.model,
		Stream: false,
		Messages: []ollamaMessage{
			{
				Role:    "system",
				Content: "You are an expert at reading and extracting information from receipts and invoices. You must carefully read all text in images and extract accurate information.",
			},
			{
				Role:    "user",
				Content: receiptScanPrompt,
			},
		},
		Images: []string{imageBase64},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Make the request
	url := fmt.Sprintf("%s/api/chat", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Extract text response
	text := strings.TrimSpace(chatResp.Message.Content)
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

// Close closes the Ollama client (no-op for HTTP client)
func (o *Ollama) Close() error {
	return nil
}
