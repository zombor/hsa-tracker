package receipt

import (
	"errors"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BoltDB", func() {
	var (
		tmpDir string
		dbPath string
		db     *BoltDB
	)

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
		dbPath = filepath.Join(tmpDir, "test.db")
		var err error
		db, err = NewBoltDB(dbPath)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if db != nil {
			db.Close()
		}
	})

	Describe("SaveReceipt", func() {
		var (
			receipt *Receipt
			err     error
		)

		BeforeEach(func() {
			receipt = &Receipt{
				ID:          "test-id",
				Title:       "Test Receipt",
				Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				Amount:      2599,
				Filename:    "test.jpg",
				ContentType: "image/jpeg",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
		})

		JustBeforeEach(func() {
			err = db.SaveReceipt(receipt)
		})

		When("saving succeeds", func() {
			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should save the receipt to the database", func() {
				saved, getErr := db.GetReceipt("test-id")
				Expect(getErr).NotTo(HaveOccurred())
				Expect(saved.ID).To(Equal("test-id"))
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
			receipt, err = db.GetReceipt(receiptID)
		})

		When("receipt exists", func() {
			BeforeEach(func() {
				receiptID = "test-id"
				testReceipt := &Receipt{
					ID:          "test-id",
					Title:       "Test Receipt",
					Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
					Amount:      2599,
					Filename:    "test.jpg",
					ContentType: "image/jpeg",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				Expect(db.SaveReceipt(testReceipt)).NotTo(HaveOccurred())
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return the correct receipt ID", func() {
				Expect(receipt.ID).To(Equal("test-id"))
			})

			It("should return the correct receipt title", func() {
				Expect(receipt.Title).To(Equal("Test Receipt"))
			})

			It("should return the correct receipt amount", func() {
				Expect(receipt.Amount).To(Equal(2599))
			})
		})

		When("receipt does not exist", func() {
			var expectedErr error

			BeforeEach(func() {
				receiptID = "nonexistent"
				expectedErr = errors.New("receipt not found: nonexistent")
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(expectedErr))
			})
		})
	})

	Describe("ListReceipts", func() {
		var (
			receipts []*Receipt
			err      error
		)

		JustBeforeEach(func() {
			receipts, err = db.ListReceipts()
		})

		When("receipts exist", func() {
			BeforeEach(func() {
				receipt1 := &Receipt{
					ID:        "id1",
					Title:     "Receipt 1",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				receipt2 := &Receipt{
					ID:        "id2",
					Title:     "Receipt 2",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				Expect(db.SaveReceipt(receipt1)).NotTo(HaveOccurred())
				Expect(db.SaveReceipt(receipt2)).NotTo(HaveOccurred())
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return all receipts", func() {
				Expect(receipts).To(HaveLen(2))
			})
		})

		When("no receipts exist", func() {
			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an empty list", func() {
				Expect(receipts).To(BeEmpty())
			})
		})
	})

	Describe("DeleteReceipt", func() {
		var (
			receiptID string
			err       error
		)

		JustBeforeEach(func() {
			err = db.DeleteReceipt(receiptID)
		})

		When("receipt exists", func() {
			BeforeEach(func() {
				receiptID = "test-id"
				receipt := &Receipt{
					ID:        "test-id",
					Title:     "Test",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				Expect(db.SaveReceipt(receipt)).NotTo(HaveOccurred())
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should remove the receipt from the database", func() {
				_, getErr := db.GetReceipt("test-id")
				Expect(getErr).To(HaveOccurred())
			})
		})

		When("receipt does not exist", func() {
			BeforeEach(func() {
				receiptID = "nonexistent"
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Close", func() {
		It("should not return an error", func() {
			err := db.Close()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SaveReimbursement", func() {
		var (
			reimbursement *Reimbursement
			err           error
		)

		BeforeEach(func() {
			reimbursement = &Reimbursement{
				ID:          "reimb-1",
				ReceiptIDs:  []string{"receipt-1", "receipt-2"},
				TotalAmount: 5000,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
		})

		JustBeforeEach(func() {
			err = db.SaveReimbursement(reimbursement)
		})

		When("saving succeeds", func() {
			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should save the reimbursement to the database", func() {
				saved, getErr := db.GetReimbursement("reimb-1")
				Expect(getErr).NotTo(HaveOccurred())
				Expect(saved.ID).To(Equal("reimb-1"))
			})
		})
	})

	Describe("GetReimbursement", func() {
		var (
			reimbursementID string
			reimbursement   *Reimbursement
			err             error
		)

		JustBeforeEach(func() {
			reimbursement, err = db.GetReimbursement(reimbursementID)
		})

		When("reimbursement exists", func() {
			BeforeEach(func() {
				reimbursementID = "reimb-1"
				testReimbursement := &Reimbursement{
					ID:          "reimb-1",
					ReceiptIDs:  []string{"receipt-1"},
					TotalAmount: 2500,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				Expect(db.SaveReimbursement(testReimbursement)).NotTo(HaveOccurred())
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return the correct reimbursement ID", func() {
				Expect(reimbursement.ID).To(Equal("reimb-1"))
			})

			It("should return the correct receipt IDs", func() {
				Expect(reimbursement.ReceiptIDs).To(Equal([]string{"receipt-1"}))
			})
		})

		When("reimbursement does not exist", func() {
			var expectedErr error

			BeforeEach(func() {
				reimbursementID = "nonexistent"
				expectedErr = errors.New("reimbursement not found: nonexistent")
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(expectedErr))
			})
		})
	})

	Describe("ListReimbursements", func() {
		var (
			reimbursements []*Reimbursement
			err            error
		)

		JustBeforeEach(func() {
			reimbursements, err = db.ListReimbursements()
		})

		When("reimbursements exist", func() {
			BeforeEach(func() {
				reimb1 := &Reimbursement{
					ID:          "reimb-1",
					ReceiptIDs:  []string{"receipt-1"},
					TotalAmount: 2500,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				reimb2 := &Reimbursement{
					ID:          "reimb-2",
					ReceiptIDs:  []string{"receipt-2"},
					TotalAmount: 3000,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				Expect(db.SaveReimbursement(reimb1)).NotTo(HaveOccurred())
				Expect(db.SaveReimbursement(reimb2)).NotTo(HaveOccurred())
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return all reimbursements", func() {
				Expect(reimbursements).To(HaveLen(2))
			})
		})

		When("no reimbursements exist", func() {
			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an empty list", func() {
				Expect(reimbursements).To(BeEmpty())
			})
		})
	})
})
