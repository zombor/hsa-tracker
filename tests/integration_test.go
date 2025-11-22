package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/zombor/hsa-tracker/internal/receipt"
	"github.com/zombor/hsa-tracker/internal/scanning"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

// MockScanner for testing
type MockScanner struct {
	receiptData *scanning.ReceiptData
	scanErr     error
}

func (m *MockScanner) ScanReceipt(imageData []byte, contentType string) (*scanning.ReceiptData, error) {
	if m.scanErr != nil {
		return nil, m.scanErr
	}
	// Simulate processing time if needed, but for tests we want speed
	return m.receiptData, nil
}

func (m *MockScanner) Close() error {
	return nil
}

var _ = Describe("Integration", func() {
	var (
		tempDir     string
		dbPath      string
		storagePath string
		db          receipt.DB
		store       receipt.Storage
		scanner     *MockScanner
		service     *receipt.Service
		server      *receipt.Server
		ghServer    *ghttp.Server
		err         error
	)

	BeforeEach(func() {
		// Create temp directory for test artifacts
		tempDir, err = os.MkdirTemp("", "hsa-tracker-test-*")
		Expect(err).NotTo(HaveOccurred())

		dbPath = filepath.Join(tempDir, "test.db")
		storagePath = filepath.Join(tempDir, "receipts")

		// Initialize real dependencies
		db, err = receipt.NewBoltDB(dbPath)
		Expect(err).NotTo(HaveOccurred())

		store, err = receipt.NewLocalStorage(storagePath)
		Expect(err).NotTo(HaveOccurred())

		// Initialize mock scanner with expected data
		scanner = &MockScanner{
			receiptData: &scanning.ReceiptData{
				Title:  "Test Integration Receipt",
				Date:   "2024-03-20",
				Amount: 42.50,
			},
		}

		// Initialize service and server
		service = receipt.NewService(db, scanner, store)
		server = receipt.NewServer(service, receipt.BasicAuth{}) // No auth for testing convenience

		// Initialize ghttp server
		ghServer = ghttp.NewServer()
	})

	AfterEach(func() {
		// Clean up
		if ghServer != nil {
			ghServer.Close()
		}
		if db != nil {
			db.Close()
		}
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	})

	It("should successfully upload a receipt, scan it, and save it", func() {
		// Register the server handler twice because we make two requests
		ghServer.AppendHandlers(
			server.ServeHTTP, // For the scan request
			server.ServeHTTP, // For the create request
		)

		// --- Step 1: Scan Request ---

		// Create a sample "PDF"
		fileContent := []byte("%PDF-1.4 ... fake pdf content ...")
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "receipt.pdf")
		Expect(err).NotTo(HaveOccurred())
		_, err = part.Write(fileContent)
		Expect(err).NotTo(HaveOccurred())
		err = writer.Close()
		Expect(err).NotTo(HaveOccurred())

		// Create request
		req, err := http.NewRequest("POST", ghServer.URL()+"/api/receipts/scan", body)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", writer.FormDataContentType())

		// Perform request
		resp, err := http.DefaultClient.Do(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		// Verify response
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(resp.Header.Get("Content-Type")).To(ContainSubstring("application/json"))

		var receiptResp receipt.Receipt
		respBody, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal(respBody, &receiptResp)
		Expect(err).NotTo(HaveOccurred())

		// Check returned data matches mock scanner data
		Expect(receiptResp.Title).To(Equal("Test Integration Receipt"))
		Expect(receiptResp.Amount).To(Equal(4250)) // 42.50 * 100

		// Verify file is in storage
		// receiptResp.Filename contains the path relative to storage base
		_, err = store.Get(receiptResp.Filename)
		Expect(err).NotTo(HaveOccurred())

		// Verify receipt is NOT in DB yet
		_, err = db.GetReceipt(receiptResp.ID)
		Expect(err).To(HaveOccurred())

		// --- Step 2: Save Request ---

		// Now, let's save the receipt using POST /api/receipts
		saveReqBody, _ := json.Marshal(receiptResp)
		saveReq, err := http.NewRequest("POST", ghServer.URL()+"/api/receipts", bytes.NewBuffer(saveReqBody))
		Expect(err).NotTo(HaveOccurred())
		saveReq.Header.Set("Content-Type", "application/json")

		saveResp, err := http.DefaultClient.Do(saveReq)
		Expect(err).NotTo(HaveOccurred())
		defer saveResp.Body.Close()

		Expect(saveResp.StatusCode).To(Equal(http.StatusCreated))

		// Verify receipt is NOW in DB
		savedReceipt, err := db.GetReceipt(receiptResp.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(savedReceipt.Title).To(Equal("Test Integration Receipt"))
	})
})
