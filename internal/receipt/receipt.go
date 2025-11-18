package receipt

import "time"

// Receipt represents a receipt with metadata
type Receipt struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Date            time.Time `json:"date"`
	Amount          int       `json:"amount"` // Amount in cents
	Filename        string    `json:"filename"`
	ContentType     string    `json:"content_type"`
	ReimbursementID string    `json:"reimbursement_id,omitempty"` // ID of the reimbursement this receipt belongs to
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Reimbursement represents a reimbursement event with associated receipts
type Reimbursement struct {
	ID          string    `json:"id"`
	ReceiptIDs  []string  `json:"receipt_ids"` // IDs of receipts in this reimbursement
	TotalAmount int       `json:"total_amount"` // Total amount in cents
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

