package receipt

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

const (
	bucketName         = "receipts"
	reimbursementBucketName = "reimbursements"
)

// DB defines the interface for database operations
type DB interface {
	// SaveReceipt saves a receipt to the database
	SaveReceipt(receipt *Receipt) error

	// GetReceipt retrieves a receipt by ID
	GetReceipt(id string) (*Receipt, error)

	// ListReceipts returns all receipts
	ListReceipts() ([]*Receipt, error)

	// DeleteReceipt removes a receipt from the database
	DeleteReceipt(id string) error

	// SaveReimbursement saves a reimbursement to the database
	SaveReimbursement(reimbursement *Reimbursement) error

	// GetReimbursement retrieves a reimbursement by ID
	GetReimbursement(id string) (*Reimbursement, error)

	// ListReimbursements returns all reimbursements
	ListReimbursements() ([]*Reimbursement, error)

	// Close closes the database connection
	Close() error
}

// BoltDB implements the DB interface using BoltDB
type BoltDB struct {
	db *bbolt.DB
}

// NewBoltDB creates a new BoltDB instance
func NewBoltDB(path string) (*BoltDB, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening boltdb: %w", err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketName)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(reimbursementBucketName)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("creating buckets: %w", err)
	}

	return &BoltDB{db: db}, nil
}

// SaveReceipt saves a receipt to the database
func (b *BoltDB) SaveReceipt(receipt *Receipt) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		data, err := json.Marshal(receipt)
		if err != nil {
			return fmt.Errorf("marshaling receipt: %w", err)
		}
		return bucket.Put([]byte(receipt.ID), data)
	})
}

// GetReceipt retrieves a receipt by ID
func (b *BoltDB) GetReceipt(id string) (*Receipt, error) {
	var receipt *Receipt
	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("receipt not found: %s", id)
		}
		return json.Unmarshal(data, &receipt)
	})
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// ListReceipts returns all receipts
func (b *BoltDB) ListReceipts() ([]*Receipt, error) {
	receipts := make([]*Receipt, 0)
	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		return bucket.ForEach(func(k, v []byte) error {
			var receipt Receipt
			if err := json.Unmarshal(v, &receipt); err != nil {
				return fmt.Errorf("unmarshaling receipt: %w", err)
			}
			receipts = append(receipts, &receipt)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return receipts, nil
}

// DeleteReceipt removes a receipt from the database
func (b *BoltDB) DeleteReceipt(id string) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		return bucket.Delete([]byte(id))
	})
}

// SaveReimbursement saves a reimbursement to the database
func (b *BoltDB) SaveReimbursement(reimbursement *Reimbursement) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(reimbursementBucketName))
		data, err := json.Marshal(reimbursement)
		if err != nil {
			return fmt.Errorf("marshaling reimbursement: %w", err)
		}
		return bucket.Put([]byte(reimbursement.ID), data)
	})
}

// GetReimbursement retrieves a reimbursement by ID
func (b *BoltDB) GetReimbursement(id string) (*Reimbursement, error) {
	var reimbursement *Reimbursement
	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(reimbursementBucketName))
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("reimbursement not found: %s", id)
		}
		return json.Unmarshal(data, &reimbursement)
	})
	if err != nil {
		return nil, err
	}
	return reimbursement, nil
}

// ListReimbursements returns all reimbursements
func (b *BoltDB) ListReimbursements() ([]*Reimbursement, error) {
	reimbursements := make([]*Reimbursement, 0)
	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(reimbursementBucketName))
		return bucket.ForEach(func(k, v []byte) error {
			var reimbursement Reimbursement
			if err := json.Unmarshal(v, &reimbursement); err != nil {
				return fmt.Errorf("unmarshaling reimbursement: %w", err)
			}
			reimbursements = append(reimbursements, &reimbursement)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return reimbursements, nil
}

// Close closes the database connection
func (b *BoltDB) Close() error {
	return b.db.Close()
}

