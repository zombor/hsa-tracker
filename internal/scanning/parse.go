package scanning

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// parseReceiptJSON parses the JSON response from Gemini
func parseReceiptJSON(text string) (*ReceiptData, error) {
	// Remove markdown code blocks if present
	text = strings.TrimSpace(text)
	
	// Remove opening markdown code blocks
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSpace(text)
	
	// Find the JSON object boundaries - look for first { and last }
	startIdx := strings.Index(text, "{")
	if startIdx == -1 {
		return nil, fmt.Errorf("no JSON object found in response")
	}
	
	endIdx := strings.LastIndex(text, "}")
	if endIdx == -1 || endIdx < startIdx {
		return nil, fmt.Errorf("invalid JSON object in response")
	}
	
	// Extract just the JSON part
	text = text[startIdx : endIdx+1]

	var data ReceiptData
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return nil, fmt.Errorf("unmarshaling json: %w", err)
	}

	// Validate and parse date
	if data.Date != "" {
		// Try to parse the date
		parsedDate, err := time.Parse("2006-01-02", data.Date)
		if err != nil {
			// Try other common formats
			formats := []string{
				"2006/01/02",
				"01/02/2006",
				"02-01-2006",
			}
			parsed := false
			for _, format := range formats {
				if d, e := time.Parse(format, data.Date); e == nil {
					data.Date = d.Format("2006-01-02")
					parsed = true
					break
				}
			}
			if !parsed {
				// If we can't parse it, use today's date
				data.Date = time.Now().Format("2006-01-02")
			}
		} else {
			data.Date = parsedDate.Format("2006-01-02")
		}
	} else {
		// Default to today if no date found
		data.Date = time.Now().Format("2006-01-02")
	}

	// Clean up title
	data.Title = strings.TrimSpace(data.Title)
	if data.Title == "" {
		data.Title = "Unknown Expense"
	}

	// Note: Amount is kept as float64 here (for JSON unmarshaling from Gemini)
	// It will be converted to int cents in the service layer when creating the Receipt model

	return &data, nil
}

