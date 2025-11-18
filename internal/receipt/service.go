package receipt

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/zombor/hsa-tracker/internal/scanning"
)

// IDGenerator generates unique IDs for receipts
type IDGenerator interface {
	Generate() string
}

// TimeSource provides the current time
type TimeSource interface {
	Now() time.Time
}

// defaultIDGenerator generates IDs using UnixNano timestamp
type defaultIDGenerator struct{}

func (g *defaultIDGenerator) Generate() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// defaultTimeSource provides the current time
type defaultTimeSource struct{}

func (t *defaultTimeSource) Now() time.Time {
	return time.Now()
}

// Service handles receipt operations
type Service struct {
	db          DB
	scanner     scanning.Scanner
	storage     Storage
	idGenerator IDGenerator
	timeSource  TimeSource
}

// NewService creates a new Service with default ID generator and time source
func NewService(db DB, scanner scanning.Scanner, storage Storage) *Service {
	return &Service{
		db:          db,
		scanner:     scanner,
		storage:     storage,
		idGenerator: &defaultIDGenerator{},
		timeSource:  &defaultTimeSource{},
	}
}

// NewServiceWithDeps creates a new Service with custom dependencies for testing
func NewServiceWithDeps(db DB, scanner scanning.Scanner, storage Storage, idGen IDGenerator, timeSrc TimeSource) *Service {
	return &Service{
		db:          db,
		scanner:     scanner,
		storage:     storage,
		idGenerator: idGen,
		timeSource:  timeSrc,
	}
}

// sanitizeFilename cleans up a filename by removing special characters and truncating length
func sanitizeFilename(filename string) string {
	// Get the extension
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	
	// Remove special characters, keep only alphanumeric, spaces, hyphens, and underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9\s\-_]`)
	base = reg.ReplaceAllString(base, "")
	
	// Replace multiple spaces with single space
	reg = regexp.MustCompile(`\s+`)
	base = reg.ReplaceAllString(base, " ")
	
	// Trim spaces
	base = strings.TrimSpace(base)
	
	// Truncate to reasonable length (50 chars for base, plus extension)
	maxLen := 50
	if len(base) > maxLen {
		base = base[:maxLen]
	}
	
	// If base is empty after sanitization, use a default
	if base == "" {
		base = "receipt"
	}
	
	return base + ext
}

// ProcessReceipt uploads a receipt, scans it, and saves it
func (s *Service) ProcessReceipt(filename string, data []byte, contentType string) (*Receipt, error) {
	// Generate unique ID
	id := s.idGenerator.Generate()
	now := s.timeSource.Now()

	// Sanitize filename to clean up phone-generated long filenames
	cleanFilename := sanitizeFilename(filename)
	
	// Save file to storage
	savedPath, err := s.storage.Save(fmt.Sprintf("%s_%s", id, cleanFilename), data)
	if err != nil {
		return nil, fmt.Errorf("saving file: %w", err)
	}

	// Scan receipt
	receiptData, err := s.scanner.ScanReceipt(data, contentType)
	if err != nil {
		// Log the scanning error with details
		slog.Error("Failed to scan receipt",
			"filename", filename,
			"content_type", contentType,
			"file_size", len(data),
			"error", err,
		)
		// Clean up the saved file since scanning failed
		s.storage.Delete(savedPath)
		return nil, fmt.Errorf("scanning receipt: %w", err)
	}

	// Parse date
	date, err := time.Parse("2006-01-02", receiptData.Date)
	if err != nil {
		date = now
	}

	// Convert amount from dollars (float) to cents (int)
	amountCents := int(receiptData.Amount * 100)

	// Create receipt model
	receipt := &Receipt{
		ID:          id,
		Title:       receiptData.Title,
		Date:        date,
		Amount:      amountCents,
		Filename:    savedPath,
		ContentType: contentType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save to database
	if err := s.db.SaveReceipt(receipt); err != nil {
		// Clean up file if database save fails
		s.storage.Delete(savedPath)
		return nil, fmt.Errorf("saving receipt to database: %w", err)
	}

	return receipt, nil
}

// GetReceipt retrieves a receipt by ID
func (s *Service) GetReceipt(id string) (*Receipt, error) {
	receipt, err := s.db.GetReceipt(id)
	if err != nil {
		return nil, fmt.Errorf("getting receipt: %w", err)
	}
	return receipt, nil
}

// ListReceipts returns all receipts
func (s *Service) ListReceipts() ([]*Receipt, error) {
	receipts, err := s.db.ListReceipts()
	if err != nil {
		return nil, fmt.Errorf("listing receipts: %w", err)
	}
	return receipts, nil
}

// DeleteReceipt removes a receipt and its file
func (s *Service) DeleteReceipt(id string) error {
	receipt, err := s.db.GetReceipt(id)
	if err != nil {
		return fmt.Errorf("getting receipt for deletion: %w", err)
	}

	// Delete file
	if err := s.storage.Delete(receipt.Filename); err != nil {
		// Log error but continue with database deletion
		slog.Warn("Failed to delete file", "filename", receipt.Filename, "error", err)
	}

	// Delete from database
	if err := s.db.DeleteReceipt(id); err != nil {
		return fmt.Errorf("deleting receipt from database: %w", err)
	}
	return nil
}

// GetReceiptFile retrieves the file data for a receipt
func (s *Service) GetReceiptFile(id string) ([]byte, string, error) {
	receipt, err := s.db.GetReceipt(id)
	if err != nil {
		return nil, "", fmt.Errorf("getting receipt: %w", err)
	}

	data, err := s.storage.Get(receipt.Filename)
	if err != nil {
		return nil, "", fmt.Errorf("getting receipt file: %w", err)
	}

	return data, receipt.ContentType, nil
}

// CreateReimbursement creates a new reimbursement and marks the specified receipts as reimbursed
func (s *Service) CreateReimbursement(receiptIDs []string) (*Reimbursement, error) {
	if len(receiptIDs) == 0 {
		return nil, fmt.Errorf("at least one receipt is required")
	}

	now := s.timeSource.Now()
	id := s.idGenerator.Generate()

	// Validate all receipts exist and calculate total
	var totalAmount int
	for _, receiptID := range receiptIDs {
		receipt, err := s.db.GetReceipt(receiptID)
		if err != nil {
			return nil, fmt.Errorf("getting receipt %s: %w", receiptID, err)
		}
		if receipt.ReimbursementID != "" {
			return nil, fmt.Errorf("receipt %s is already reimbursed", receiptID)
		}
		totalAmount += receipt.Amount
	}

	// Create reimbursement
	reimbursement := &Reimbursement{
		ID:          id,
		ReceiptIDs:  receiptIDs,
		TotalAmount: totalAmount,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save reimbursement
	if err := s.db.SaveReimbursement(reimbursement); err != nil {
		return nil, fmt.Errorf("saving reimbursement: %w", err)
	}

	// Mark receipts as reimbursed
	for _, receiptID := range receiptIDs {
		receipt, err := s.db.GetReceipt(receiptID)
		if err != nil {
			return nil, fmt.Errorf("getting receipt %s for update: %w", receiptID, err)
		}
		receipt.ReimbursementID = id
		receipt.UpdatedAt = now
		if err := s.db.SaveReceipt(receipt); err != nil {
			return nil, fmt.Errorf("updating receipt %s: %w", receiptID, err)
		}
	}

	return reimbursement, nil
}

// GetReimbursement retrieves a reimbursement by ID
func (s *Service) GetReimbursement(id string) (*Reimbursement, error) {
	reimbursement, err := s.db.GetReimbursement(id)
	if err != nil {
		return nil, fmt.Errorf("getting reimbursement: %w", err)
	}
	return reimbursement, nil
}

// GetReimbursementWithReceipts retrieves a reimbursement with its associated receipts
func (s *Service) GetReimbursementWithReceipts(id string) (*Reimbursement, []*Receipt, error) {
	reimbursement, err := s.db.GetReimbursement(id)
	if err != nil {
		return nil, nil, fmt.Errorf("getting reimbursement: %w", err)
	}

	// Get all receipts for this reimbursement
	receipts := make([]*Receipt, 0, len(reimbursement.ReceiptIDs))
	for _, receiptID := range reimbursement.ReceiptIDs {
		receipt, err := s.db.GetReceipt(receiptID)
		if err != nil {
			return nil, nil, fmt.Errorf("getting receipt %s: %w", receiptID, err)
		}
		receipts = append(receipts, receipt)
	}

	return reimbursement, receipts, nil
}

// ListReimbursements returns all reimbursements
func (s *Service) ListReimbursements() ([]*Reimbursement, error) {
	reimbursements, err := s.db.ListReimbursements()
	if err != nil {
		return nil, fmt.Errorf("listing reimbursements: %w", err)
	}
	return reimbursements, nil
}
