package receipt

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/zombor/hsa-tracker/internal/scanning"
)

func TestService(t *testing.T) {
	// Disable logging during tests
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	RegisterFailHandler(Fail)
	RunSpecs(t, "Receipt Suite")
}

// mockDB is a mock implementation of DB
type mockDB struct {
	receipts              map[string]*Receipt
	reimbursements        map[string]*Reimbursement
	saveErr               error
	getErr                error
	listErr               error
	deleteErr             error
	saveReimbursementErr  error
	getReimbursementErr   error
	listReimbursementsErr error
}

func newMockDB() *mockDB {
	return &mockDB{
		receipts:       make(map[string]*Receipt),
		reimbursements: make(map[string]*Reimbursement),
	}
}

func (m *mockDB) SaveReceipt(receipt *Receipt) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.receipts[receipt.ID] = receipt
	return nil
}

func (m *mockDB) GetReceipt(id string) (*Receipt, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	receipt, ok := m.receipts[id]
	if !ok {
		return nil, errors.New("receipt not found")
	}
	return receipt, nil
}

func (m *mockDB) ListReceipts() ([]*Receipt, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	receipts := make([]*Receipt, 0, len(m.receipts))
	for _, r := range m.receipts {
		receipts = append(receipts, r)
	}
	return receipts, nil
}

func (m *mockDB) DeleteReceipt(id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.receipts[id]; !ok {
		return errors.New("receipt not found")
	}
	delete(m.receipts, id)
	return nil
}

func (m *mockDB) SaveReimbursement(reimbursement *Reimbursement) error {
	if m.saveReimbursementErr != nil {
		return m.saveReimbursementErr
	}
	m.reimbursements[reimbursement.ID] = reimbursement
	return nil
}

func (m *mockDB) GetReimbursement(id string) (*Reimbursement, error) {
	if m.getReimbursementErr != nil {
		return nil, m.getReimbursementErr
	}
	reimbursement, ok := m.reimbursements[id]
	if !ok {
		return nil, errors.New("reimbursement not found")
	}
	return reimbursement, nil
}

func (m *mockDB) ListReimbursements() ([]*Reimbursement, error) {
	if m.listReimbursementsErr != nil {
		return nil, m.listReimbursementsErr
	}
	reimbursements := make([]*Reimbursement, 0, len(m.reimbursements))
	for _, r := range m.reimbursements {
		reimbursements = append(reimbursements, r)
	}
	return reimbursements, nil
}

func (m *mockDB) Close() error {
	return nil
}

// mockStorage is a mock implementation of Storage
type mockStorage struct {
	files     map[string][]byte
	saveErr   error
	getErr    error
	deleteErr error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		files: make(map[string][]byte),
	}
}

func (m *mockStorage) Save(filename string, data []byte) (string, error) {
	if m.saveErr != nil {
		return "", m.saveErr
	}
	m.files[filename] = data
	return filename, nil
}

func (m *mockStorage) Get(path string) ([]byte, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	data, ok := m.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return data, nil
}

func (m *mockStorage) Delete(path string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.files[path]; !ok {
		return errors.New("file not found")
	}
	delete(m.files, path)
	return nil
}

// mockScanner is a mock implementation of scanning.Scanner
type mockScanner struct {
	scanErr     error
	receiptData *scanning.ReceiptData
}

func newMockScanner() *mockScanner {
	return &mockScanner{
		receiptData: &scanning.ReceiptData{
			Title:  "Test Receipt",
			Date:   "2024-01-15",
			Amount: 25.99,
		},
	}
}

func (m *mockScanner) ScanReceipt(imageData []byte, contentType string) (*scanning.ReceiptData, error) {
	if m.scanErr != nil {
		return nil, m.scanErr
	}
	return m.receiptData, nil
}

func (m *mockScanner) Close() error {
	return nil
}

// mockIDGenerator is a mock implementation of IDGenerator
type mockIDGenerator struct {
	id string
}

func (m *mockIDGenerator) Generate() string {
	return m.id
}

// mockTimeSource is a mock implementation of TimeSource
type mockTimeSource struct {
	now time.Time
}

func (m *mockTimeSource) Now() time.Time {
	return m.now
}

var _ = Describe("Service", func() {
	var (
		db      *mockDB
		storage *mockStorage
		scanner *mockScanner
		idGen   *mockIDGenerator
		timeSrc *mockTimeSource
		service *Service
	)

	BeforeEach(func() {
		db = newMockDB()
		storage = newMockStorage()
		scanner = newMockScanner()
		idGen = &mockIDGenerator{id: "test-id-123"}
		timeSrc = &mockTimeSource{now: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)}
		service = NewServiceWithDeps(db, scanner, storage, idGen, timeSrc)
	})

	Describe("ScanReceipt", func() {
		var (
			filename    string
			data        []byte
			contentType string
			receipt     *Receipt
			err         error
		)

		BeforeEach(func() {
			filename = "receipt.jpg"
			data = []byte("fake image data")
			contentType = "image/jpeg"
		})

		JustBeforeEach(func() {
			receipt, err = service.ScanReceipt(filename, data, contentType)
		})

		When("processing succeeds", func() {
			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should set the receipt ID correctly", func() {
				Expect(receipt.ID).To(Equal("test-id-123"))
			})

			It("should set the receipt title from scanner", func() {
				Expect(receipt.Title).To(Equal("Test Receipt"))
			})

			It("should convert amount from dollars to cents", func() {
				Expect(receipt.Amount).To(Equal(2599))
			})

			It("should set the filename with ID prefix", func() {
				Expect(receipt.Filename).To(Equal("test-id-123_receipt.jpg"))
			})

			It("should NOT save the receipt to the database yet", func() {
				_, getErr := db.GetReceipt("test-id-123")
				Expect(getErr).To(HaveOccurred())
			})

			It("should save the file to storage", func() {
				Expect(storage.files).To(HaveKey("test-id-123_receipt.jpg"))
			})
		})

		When("storage save fails", func() {
			var setupErr error

			BeforeEach(func() {
				setupErr = errors.New("storage error")
				storage.saveErr = setupErr
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(setupErr))
			})
		})

		When("scanner fails", func() {
			var setupErr error

			BeforeEach(func() {
				setupErr = errors.New("scan error")
				scanner.scanErr = setupErr
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(setupErr))
			})

			It("cleans up the saved file", func() {
				Expect(storage.files).NotTo(HaveKey("test-id-123_receipt.jpg"))
			})
		})
	})

	Describe("CreateReceipt", func() {
		var (
			receipt *Receipt
			err     error
		)

		BeforeEach(func() {
			receipt = &Receipt{
				ID:    "test-id-123",
				Title: "Test Receipt",
			}
		})

		JustBeforeEach(func() {
			err = service.CreateReceipt(receipt)
		})

		When("save succeeds", func() {
			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should save the receipt to the database", func() {
				saved, getErr := db.GetReceipt("test-id-123")
				Expect(getErr).NotTo(HaveOccurred())
				Expect(saved.ID).To(Equal(receipt.ID))
			})

			It("should set CreatedAt and UpdatedAt", func() {
				saved, _ := db.GetReceipt("test-id-123")
				Expect(saved.CreatedAt).NotTo(BeZero())
				Expect(saved.UpdatedAt).NotTo(BeZero())
			})
		})

		When("database save fails", func() {
			var setupErr error

			BeforeEach(func() {
				setupErr = errors.New("database error")
				db.saveErr = setupErr
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(setupErr))
			})
		})
	})

	Describe("GetReceipt", func() {
		var (
			receiptID string
			receipt   *Receipt
			err       error
		)

		JustBeforeEach(func() {
			receipt, err = service.GetReceipt(receiptID)
		})

		When("receipt exists", func() {
			BeforeEach(func() {
				receiptID = "test-id"
				db.receipts["test-id"] = &Receipt{
					ID:    "test-id",
					Title: "Test",
				}
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return the correct receipt", func() {
				Expect(receipt.ID).To(Equal("test-id"))
			})
		})

		When("receipt does not exist", func() {
			var setupErr error

			BeforeEach(func() {
				receiptID = "nonexistent"
				setupErr = errors.New("receipt not found")
				db.getErr = setupErr
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(setupErr))
			})
		})
	})

	Describe("ListReceipts", func() {
		var (
			receipts []*Receipt
			err      error
		)

		JustBeforeEach(func() {
			receipts, err = service.ListReceipts()
		})

		When("receipts exist", func() {
			BeforeEach(func() {
				db.receipts["id1"] = &Receipt{ID: "id1"}
				db.receipts["id2"] = &Receipt{ID: "id2"}
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return all receipts", func() {
				Expect(receipts).To(HaveLen(2))
			})
		})
	})

	Describe("DeleteReceipt", func() {
		var (
			receiptID string
			err       error
		)

		JustBeforeEach(func() {
			err = service.DeleteReceipt(receiptID)
		})

		When("deletion succeeds", func() {
			BeforeEach(func() {
				receiptID = "test-id"
				db.receipts["test-id"] = &Receipt{
					ID:       "test-id",
					Filename: "test-file.jpg",
				}
				storage.files["test-file.jpg"] = []byte("data")
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should remove the receipt from the database", func() {
				Expect(db.receipts).NotTo(HaveKey("test-id"))
			})

			It("should remove the file from storage", func() {
				Expect(storage.files).NotTo(HaveKey("test-file.jpg"))
			})
		})

		When("storage delete fails", func() {
			BeforeEach(func() {
				receiptID = "test-id"
				storage.deleteErr = errors.New("storage delete error")
				db.receipts["test-id"] = &Receipt{
					ID:       "test-id",
					Filename: "test-file.jpg",
				}
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should still remove the receipt from the database", func() {
				Expect(db.receipts).NotTo(HaveKey("test-id"))
			})
		})
	})

	Describe("GetReceiptFile", func() {
		var (
			receiptID   string
			data        []byte
			contentType string
			err         error
		)

		JustBeforeEach(func() {
			data, contentType, err = service.GetReceiptFile(receiptID)
		})

		When("receipt and file exist", func() {
			BeforeEach(func() {
				receiptID = "test-id"
				db.receipts["test-id"] = &Receipt{
					ID:          "test-id",
					Filename:    "test-file.jpg",
					ContentType: "image/jpeg",
				}
				storage.files["test-file.jpg"] = []byte("file data")
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return the file data", func() {
				Expect(string(data)).To(Equal("file data"))
			})

			It("should return the content type", func() {
				Expect(contentType).To(Equal("image/jpeg"))
			})
		})

		When("receipt does not exist", func() {
			var setupErr error

			BeforeEach(func() {
				receiptID = "nonexistent"
				setupErr = errors.New("receipt not found")
				db.getErr = setupErr
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(setupErr))
			})
		})
	})
})
