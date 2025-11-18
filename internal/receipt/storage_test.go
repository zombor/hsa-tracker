package receipt

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LocalStorage", func() {
	var (
		tmpDir  string
		storage Storage
	)

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
		var err error
		storage, err = NewLocalStorage(tmpDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Save", func() {
		var (
			filename  string
			data      []byte
			savedPath string
			err       error
		)

		BeforeEach(func() {
			filename = "test.jpg"
			data = []byte("test file content")
		})

		JustBeforeEach(func() {
			savedPath, err = storage.Save(filename, data)
		})

		When("saving succeeds", func() {
			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return the correct path", func() {
				Expect(savedPath).To(Equal(filename))
			})

			It("should save the file to disk", func() {
				filePath := filepath.Join(tmpDir, filename)
				Expect(filePath).To(BeAnExistingFile())
			})
		})
	})

	Describe("Get", func() {
		var (
			filename string
			data     []byte
			err      error
		)

		JustBeforeEach(func() {
			data, err = storage.Get(filename)
		})

		When("file exists", func() {
			BeforeEach(func() {
				filename = "test.jpg"
				testData := []byte("test file content")
				_, saveErr := storage.Save(filename, testData)
				Expect(saveErr).NotTo(HaveOccurred())
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return the correct file data", func() {
				Expect(string(data)).To(Equal("test file content"))
			})
		})

		When("file does not exist", func() {
			BeforeEach(func() {
				filename = "nonexistent.jpg"
			})

			It("returns the error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("reading file"))
			})
		})
	})

	Describe("Delete", func() {
		var (
			filename string
			err      error
		)

		JustBeforeEach(func() {
			err = storage.Delete(filename)
		})

		When("file exists", func() {
			BeforeEach(func() {
				filename = "test.jpg"
				testData := []byte("test content")
				_, saveErr := storage.Save(filename, testData)
				Expect(saveErr).NotTo(HaveOccurred())
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should remove the file from disk", func() {
				filePath := filepath.Join(tmpDir, filename)
				Expect(filePath).NotTo(BeAnExistingFile())
			})

			It("should make the file inaccessible via Get", func() {
				_, getErr := storage.Get(filename)
				Expect(getErr).To(HaveOccurred())
			})
		})

		When("file does not exist", func() {
			BeforeEach(func() {
				filename = "nonexistent.jpg"
			})

			It("returns the error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("deleting file"))
			})
		})
	})

	Describe("NewLocalStorage", func() {
		var (
			storagePath string
			storage     Storage
			err         error
		)

		JustBeforeEach(func() {
			storage, err = NewLocalStorage(storagePath)
		})

		When("directory does not exist", func() {
			BeforeEach(func() {
				baseDir := GinkgoT().TempDir()
				storagePath = filepath.Join(baseDir, "receipts")
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should create the directory", func() {
				Expect(storagePath).To(BeADirectory())
			})

			It("should allow saving files", func() {
				_, saveErr := storage.Save("test.jpg", []byte("data"))
				Expect(saveErr).NotTo(HaveOccurred())
			})
		})

		When("directory already exists", func() {
			BeforeEach(func() {
				storagePath = GinkgoT().TempDir()
			})

			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should allow saving files", func() {
				_, saveErr := storage.Save("test.jpg", []byte("data"))
				Expect(saveErr).NotTo(HaveOccurred())
			})
		})
	})
})
