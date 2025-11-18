package scanning

// ReceiptData contains extracted information from a receipt
type ReceiptData struct {
	Title  string  `json:"title"`
	Date   string  `json:"date"`   // ISO 8601 format
	Amount float64 `json:"amount"`
}

// Scanner defines the interface for receipt scanning operations
type Scanner interface {
	// ScanReceipt analyzes a receipt image/PDF and extracts metadata
	ScanReceipt(imageData []byte, contentType string) (*ReceiptData, error)
	// Close closes the scanner and releases resources
	Close() error
}

