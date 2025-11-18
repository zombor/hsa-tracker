package receipt

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Server", func() {
	var (
		service     *Service
		server      *Server
		auth        BasicAuth
		ghttpServer *ghttp.Server
	)

	setupServer := func() {
		if ghttpServer != nil {
			ghttpServer.Close()
		}
		ghttpServer = ghttp.NewServer()
		ghttpServer.AppendHandlers(server.ServeHTTP)
	}

	BeforeEach(func() {
		service = NewService(newMockDB(), newMockScanner(), newMockStorage())
		auth = BasicAuth{}
		server = NewServerWithMux(service, auth, http.NewServeMux())
		setupServer()
	})

	AfterEach(func() {
		if ghttpServer != nil {
			ghttpServer.Close()
		}
	})

	Describe("handleIndex", func() {
		When("request method is GET", func() {
			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should return HTML containing HSA Tracker", func() {
				resp, err := http.Get(ghttpServer.URL() + "/")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("HSA Tracker"))
			})
		})

		When("request method is not GET", func() {
			It("should return status Method Not Allowed", func() {
				req, err := http.NewRequest("POST", ghttpServer.URL()+"/", nil)
				Expect(err).NotTo(HaveOccurred())
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
				resp.Body.Close()
			})
		})
	})

	Describe("handleListReceipts", func() {
		When("receipts exist", func() {
			BeforeEach(func() {
				db := newMockDB()
				db.receipts["id1"] = &Receipt{ID: "id1", Title: "Test 1"}
				db.receipts["id2"] = &Receipt{ID: "id2", Title: "Test 2"}
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should return all receipts", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				var receipts []*Receipt
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(body, &receipts)).NotTo(HaveOccurred())
				Expect(receipts).To(HaveLen(2))
			})

			It("should set Content-Type to application/json", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))
			})
		})

		When("no receipts exist", func() {
			BeforeEach(func() {
				db := newMockDB()
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should return an empty array", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				var receipts []*Receipt
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(body, &receipts)).NotTo(HaveOccurred())
				Expect(receipts).To(BeEmpty())
			})
		})

		When("service returns an error", func() {
			BeforeEach(func() {
				setupErr := errors.New("service error")
				db := newMockDB()
				db.listErr = setupErr
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status Internal Server Error", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()
			})

			It("should return error message", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("Internal server error"))
			})
		})
	})

	Describe("handleUploadReceipt", func() {
		When("upload succeeds", func() {
			It("should return status Created", func() {
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				part, _ := writer.CreateFormFile("file", "test.jpg")
				part.Write([]byte("fake image data"))
				writer.Close()

				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", writer.FormDataContentType(), &b)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()
			})

			It("should return a receipt with an ID", func() {
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				part, _ := writer.CreateFormFile("file", "test.jpg")
				part.Write([]byte("fake image data"))
				writer.Close()

				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", writer.FormDataContentType(), &b)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				var receipt Receipt
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(body, &receipt)).NotTo(HaveOccurred())
				Expect(receipt.ID).NotTo(BeEmpty())
			})

			It("should set Content-Type to application/json", func() {
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				part, _ := writer.CreateFormFile("file", "test.jpg")
				part.Write([]byte("fake image data"))
				writer.Close()

				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", writer.FormDataContentType(), &b)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))
			})
		})

		When("upload succeeds with PNG file", func() {
			It("should return status Created", func() {
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				part, _ := writer.CreateFormFile("file", "test.png")
				part.Write([]byte("fake image data"))
				writer.Close()

				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", writer.FormDataContentType(), &b)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()
			})
		})

		When("upload succeeds with PDF file", func() {
			It("should return status Created", func() {
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				part, _ := writer.CreateFormFile("file", "test.pdf")
				part.Write([]byte("fake pdf data"))
				writer.Close()

				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", writer.FormDataContentType(), &b)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()
			})
		})

		When("no file is provided", func() {
			It("should return status Bad Request", func() {
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				writer.Close()

				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", writer.FormDataContentType(), &b)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()
			})

			It("should return error message", func() {
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				writer.Close()

				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", writer.FormDataContentType(), &b)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				// Error message should indicate no file was provided
				Expect(string(body)).To(ContainSubstring("file"))
			})
		})

		When("invalid multipart form", func() {
			It("should return status Bad Request", func() {
				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", "multipart/form-data", bytes.NewBufferString("invalid"))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()
			})

			It("should return error message", func() {
				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", "multipart/form-data", bytes.NewBufferString("invalid"))
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("Error parsing form"))
			})
		})

		When("service returns an error", func() {
			BeforeEach(func() {
				scanner := newMockScanner()
				scanner.scanErr = errors.New("scan error")
				service = NewService(newMockDB(), scanner, newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status Bad Request", func() {
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				part, _ := writer.CreateFormFile("file", "test.jpg")
				part.Write([]byte("fake image data"))
				writer.Close()

				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", writer.FormDataContentType(), &b)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()
			})

			It("should return error in JSON", func() {
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				part, _ := writer.CreateFormFile("file", "test.jpg")
				part.Write([]byte("fake image data"))
				writer.Close()

				resp, err := http.Post(ghttpServer.URL()+"/api/receipts", writer.FormDataContentType(), &b)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				var response map[string]string
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(body, &response)).NotTo(HaveOccurred())
				Expect(response["error"]).To(ContainSubstring("scan error"))
			})
		})
	})

	Describe("handleGetReceipt", func() {
		When("receipt exists", func() {
			BeforeEach(func() {
				db := newMockDB()
				receipt := &Receipt{ID: "test-id", Title: "Test Receipt"}
				db.receipts["test-id"] = receipt
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/test-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should return the correct receipt", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/test-id")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				var got Receipt
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(body, &got)).NotTo(HaveOccurred())
				Expect(got.ID).To(Equal("test-id"))
				Expect(got.Title).To(Equal("Test Receipt"))
			})

			It("should set Content-Type to application/json", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/test-id")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))
			})
		})

		When("receipt does not exist", func() {
			It("should return status Not Found", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/nonexistent")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})

			It("should return error message", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/nonexistent")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("Receipt not found"))
			})
		})

		When("service returns an error", func() {
			BeforeEach(func() {
				db := newMockDB()
				db.getErr = errors.New("database error")
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status Not Found", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/test-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})
		})
	})

	Describe("handleGetReceiptFile", func() {
		When("receipt and file exist", func() {
			BeforeEach(func() {
				db := newMockDB()
				storage := newMockStorage()
				receipt := &Receipt{
					ID:          "test-id",
					Filename:    "test-file.jpg",
					ContentType: "image/jpeg",
				}
				db.receipts["test-id"] = receipt
				storage.files["test-file.jpg"] = []byte("file content")
				service = NewService(db, newMockScanner(), storage)
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/test-id/file")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should return the file content", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/test-id/file")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal("file content"))
			})

			It("should set the correct Content-Type header", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/test-id/file")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("Content-Type")).To(Equal("image/jpeg"))
			})
		})

		When("receipt does not exist", func() {
			It("should return status Not Found", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/nonexistent/file")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})

			It("should return error message", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/nonexistent/file")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("File not found"))
			})
		})

		When("file does not exist in storage", func() {
			BeforeEach(func() {
				db := newMockDB()
				storage := newMockStorage()
				receipt := &Receipt{
					ID:          "test-id",
					Filename:    "missing-file.jpg",
					ContentType: "image/jpeg",
				}
				db.receipts["test-id"] = receipt
				service = NewService(db, newMockScanner(), storage)
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status Not Found", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/receipts/test-id/file")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})
		})
	})

	Describe("handleDeleteReceipt", func() {
		When("deletion succeeds", func() {
			BeforeEach(func() {
				db := newMockDB()
				storage := newMockStorage()
				receipt := &Receipt{
					ID:       "test-id",
					Filename: "test-file.jpg",
				}
				db.receipts["test-id"] = receipt
				storage.files["test-file.jpg"] = []byte("data")
				service = NewService(db, newMockScanner(), storage)
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status No Content", func() {
				req, err := http.NewRequest("DELETE", ghttpServer.URL()+"/api/receipts/test-id", nil)
				Expect(err).NotTo(HaveOccurred())
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
				resp.Body.Close()
			})

			It("should remove the receipt from the database", func() {
				req, err := http.NewRequest("DELETE", ghttpServer.URL()+"/api/receipts/test-id", nil)
				Expect(err).NotTo(HaveOccurred())
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()
				// Verify deletion by attempting to get the receipt
				_, getErr := service.GetReceipt("test-id")
				Expect(getErr).To(HaveOccurred())
			})
		})

		When("receipt does not exist", func() {
			It("should return status Internal Server Error", func() {
				req, err := http.NewRequest("DELETE", ghttpServer.URL()+"/api/receipts/nonexistent", nil)
				Expect(err).NotTo(HaveOccurred())
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()
			})

			It("should return error message", func() {
				req, err := http.NewRequest("DELETE", ghttpServer.URL()+"/api/receipts/nonexistent", nil)
				Expect(err).NotTo(HaveOccurred())
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("Error deleting receipt"))
			})
		})

		When("service returns an error", func() {
			BeforeEach(func() {
				db := newMockDB()
				db.deleteErr = errors.New("database error")
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status Internal Server Error", func() {
				req, err := http.NewRequest("DELETE", ghttpServer.URL()+"/api/receipts/test-id", nil)
				Expect(err).NotTo(HaveOccurred())
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()
			})
		})
	})

	Describe("authenticate", func() {
		var result bool

		When("no auth is configured", func() {
			It("should return true", func() {
				req, err := http.NewRequest("GET", ghttpServer.URL()+"/", nil)
				Expect(err).NotTo(HaveOccurred())
				result = server.authenticate(req)
				Expect(result).To(BeTrue())
			})
		})

		When("valid credentials are provided", func() {
			BeforeEach(func() {
				auth = BasicAuth{Username: "user", Password: "pass"}
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return true", func() {
				req, err := http.NewRequest("GET", ghttpServer.URL()+"/", nil)
				Expect(err).NotTo(HaveOccurred())
				credentials := base64.StdEncoding.EncodeToString([]byte("user:pass"))
				req.Header.Set("Authorization", "Basic "+credentials)
				result = server.authenticate(req)
				Expect(result).To(BeTrue())
			})
		})

		When("invalid credentials are provided", func() {
			BeforeEach(func() {
				auth = BasicAuth{Username: "user", Password: "pass"}
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return false", func() {
				req, err := http.NewRequest("GET", ghttpServer.URL()+"/", nil)
				Expect(err).NotTo(HaveOccurred())
				credentials := base64.StdEncoding.EncodeToString([]byte("user:wrong"))
				req.Header.Set("Authorization", "Basic "+credentials)
				result = server.authenticate(req)
				Expect(result).To(BeFalse())
			})
		})

		When("no authorization header is provided", func() {
			BeforeEach(func() {
				auth = BasicAuth{Username: "user", Password: "pass"}
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return false", func() {
				req, err := http.NewRequest("GET", ghttpServer.URL()+"/", nil)
				Expect(err).NotTo(HaveOccurred())
				result = server.authenticate(req)
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("requireAuth", func() {
		When("request is unauthorized", func() {
			BeforeEach(func() {
				auth = BasicAuth{Username: "user", Password: "pass"}
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status Unauthorized", func() {
				resp, err := http.Get(ghttpServer.URL() + "/")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
				resp.Body.Close()
			})

			It("should set WWW-Authenticate header", func() {
				resp, err := http.Get(ghttpServer.URL() + "/")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("WWW-Authenticate")).NotTo(BeEmpty())
			})
		})
	})

	Describe("handleListReimbursements", func() {
		When("reimbursements exist", func() {
			BeforeEach(func() {
				db := newMockDB()
				db.reimbursements["reimb1"] = &Reimbursement{ID: "reimb1"}
				db.reimbursements["reimb2"] = &Reimbursement{ID: "reimb2"}
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should return all reimbursements", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				var reimbursements []*Reimbursement
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(body, &reimbursements)).NotTo(HaveOccurred())
				Expect(reimbursements).To(HaveLen(2))
			})

			It("should set Content-Type to application/json", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))
			})
		})

		When("no reimbursements exist", func() {
			BeforeEach(func() {
				db := newMockDB()
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should return an empty array", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				var reimbursements []*Reimbursement
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(body, &reimbursements)).NotTo(HaveOccurred())
				Expect(reimbursements).To(BeEmpty())
			})
		})

		When("service returns an error", func() {
			BeforeEach(func() {
				setupErr := errors.New("service error")
				db := newMockDB()
				db.listReimbursementsErr = setupErr
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status Internal Server Error", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()
			})

			It("should return error message", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("Internal server error"))
			})
		})
	})

	Describe("handleCreateReimbursement", func() {
		When("creation succeeds", func() {
			BeforeEach(func() {
				db := newMockDB()
				db.receipts["receipt1"] = &Receipt{ID: "receipt1"}
				db.receipts["receipt2"] = &Receipt{ID: "receipt2"}
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status Created", func() {
				body := map[string][]string{
					"receipt_ids": {"receipt1", "receipt2"},
				}
				bodyBytes, _ := json.Marshal(body)
				resp, err := http.Post(ghttpServer.URL()+"/api/reimbursements", "application/json", bytes.NewBuffer(bodyBytes))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()
			})

			It("should return a reimbursement with an ID", func() {
				body := map[string][]string{
					"receipt_ids": {"receipt1", "receipt2"},
				}
				bodyBytes, _ := json.Marshal(body)
				resp, err := http.Post(ghttpServer.URL()+"/api/reimbursements", "application/json", bytes.NewBuffer(bodyBytes))
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				var reimbursement Reimbursement
				respBody, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(respBody, &reimbursement)).NotTo(HaveOccurred())
				Expect(reimbursement.ID).NotTo(BeEmpty())
			})

			It("should set Content-Type to application/json", func() {
				body := map[string][]string{
					"receipt_ids": {"receipt1", "receipt2"},
				}
				bodyBytes, _ := json.Marshal(body)
				resp, err := http.Post(ghttpServer.URL()+"/api/reimbursements", "application/json", bytes.NewBuffer(bodyBytes))
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))
			})
		})

		When("invalid JSON body", func() {
			It("should return status Bad Request", func() {
				resp, err := http.Post(ghttpServer.URL()+"/api/reimbursements", "application/json", bytes.NewBufferString("invalid json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()
			})

			It("should return error message", func() {
				resp, err := http.Post(ghttpServer.URL()+"/api/reimbursements", "application/json", bytes.NewBufferString("invalid json"))
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("Invalid request body"))
			})
		})

		When("empty receipt_ids", func() {
			It("should return status Bad Request", func() {
				body := map[string][]string{
					"receipt_ids": {},
				}
				bodyBytes, _ := json.Marshal(body)
				resp, err := http.Post(ghttpServer.URL()+"/api/reimbursements", "application/json", bytes.NewBuffer(bodyBytes))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()
			})
		})

		When("service returns an error", func() {
			BeforeEach(func() {
				db := newMockDB()
				// Set up receipt so it passes validation, then fail on save
				db.receipts["receipt1"] = &Receipt{ID: "receipt1"}
				db.saveReimbursementErr = errors.New("database error")
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status Bad Request", func() {
				body := map[string][]string{
					"receipt_ids": {"receipt1"},
				}
				bodyBytes, _ := json.Marshal(body)
				resp, err := http.Post(ghttpServer.URL()+"/api/reimbursements", "application/json", bytes.NewBuffer(bodyBytes))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()
			})

			It("should return error in JSON", func() {
				body := map[string][]string{
					"receipt_ids": {"receipt1"},
				}
				bodyBytes, _ := json.Marshal(body)
				resp, err := http.Post(ghttpServer.URL()+"/api/reimbursements", "application/json", bytes.NewBuffer(bodyBytes))
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				var response map[string]string
				respBody, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(respBody, &response)).NotTo(HaveOccurred())
				Expect(response["error"]).To(ContainSubstring("database error"))
			})
		})
	})

	Describe("handleGetReimbursement", func() {
		When("reimbursement exists", func() {
			BeforeEach(func() {
				db := newMockDB()
				reimbursement := &Reimbursement{ID: "test-id"}
				receipt := &Receipt{ID: "receipt1", ReimbursementID: "test-id"}
				db.reimbursements["test-id"] = reimbursement
				db.receipts["receipt1"] = receipt
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements/test-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should return reimbursement and receipts", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements/test-id")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				var response map[string]interface{}
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(body, &response)).NotTo(HaveOccurred())
				Expect(response).To(HaveKey("reimbursement"))
				Expect(response).To(HaveKey("receipts"))
			})

			It("should set Content-Type to application/json", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements/test-id")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))
			})
		})

		When("reimbursement does not exist", func() {
			It("should return status Not Found", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements/nonexistent")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})

			It("should return error message", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements/nonexistent")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("Reimbursement not found"))
			})
		})

		When("service returns an error", func() {
			BeforeEach(func() {
				db := newMockDB()
				db.getReimbursementErr = errors.New("database error")
				service = NewService(db, newMockScanner(), newMockStorage())
				server = NewServerWithMux(service, auth, http.NewServeMux())
				setupServer()
			})

			It("should return status Not Found", func() {
				resp, err := http.Get(ghttpServer.URL() + "/api/reimbursements/test-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})
		})
	})

	Describe("handleStaticCSS", func() {
		When("request is GET", func() {
			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/static/app.css")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should set Content-Type to text/css", func() {
				resp, err := http.Get(ghttpServer.URL() + "/static/app.css")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("Content-Type")).To(Equal("text/css"))
			})

			It("should return CSS content", func() {
				resp, err := http.Get(ghttpServer.URL() + "/static/app.css")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(body)).To(BeNumerically(">", 0))
			})
		})
	})

	Describe("handleStaticJS", func() {
		When("request is GET", func() {
			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/static/app.js")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should set Content-Type to application/javascript", func() {
				resp, err := http.Get(ghttpServer.URL() + "/static/app.js")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("Content-Type")).To(Equal("application/javascript"))
			})

			It("should return JavaScript content", func() {
				resp, err := http.Get(ghttpServer.URL() + "/static/app.js")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(body)).To(BeNumerically(">", 0))
			})
		})
	})

	Describe("handleControllers", func() {
		When("requesting a controller file", func() {
			It("should return status OK", func() {
				resp, err := http.Get(ghttpServer.URL() + "/static/controllers/receipts_controller.js")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should set Content-Type to application/javascript", func() {
				resp, err := http.Get(ghttpServer.URL() + "/static/controllers/receipts_controller.js")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.Header.Get("Content-Type")).To(Equal("application/javascript"))
			})
		})
	})
})
