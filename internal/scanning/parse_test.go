package scanning

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestScanning(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanning Suite")
}

var _ = Describe("parseReceiptJSON", func() {
	var (
		jsonInput string
		data      *ReceiptData
		err       error
	)

	JustBeforeEach(func() {
		data, err = parseReceiptJSON(jsonInput)
	})

	When("parsing valid JSON", func() {
		BeforeEach(func() {
			jsonInput = `{"title": "CVS Pharmacy", "date": "2024-01-15", "amount": 25.99}`
		})

		It("should not return an error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should parse the title correctly", func() {
			Expect(data.Title).To(Equal("CVS Pharmacy"))
		})

		It("should parse the date correctly", func() {
			Expect(data.Date).To(Equal("2024-01-15"))
		})

		It("should parse the amount correctly", func() {
			Expect(data.Amount).To(Equal(25.99))
		})
	})

	When("parsing JSON with markdown code blocks", func() {
		BeforeEach(func() {
			jsonInput = "```json\n{\"title\": \"Test\", \"date\": \"2024-01-15\", \"amount\": 10.50}\n```"
		})

		It("should not return an error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should parse the title correctly", func() {
			Expect(data.Title).To(Equal("Test"))
		})

		It("should parse the date correctly", func() {
			Expect(data.Date).To(Equal("2024-01-15"))
		})
	})

	When("parsing JSON with invalid date", func() {
		BeforeEach(func() {
			jsonInput = `{"title": "Test", "date": "invalid-date", "amount": 10.50}`
		})

		It("should not return an error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should default to today's date", func() {
			expectedDate := time.Now().Format("2006-01-02")
			Expect(data.Date).To(Equal(expectedDate))
		})
	})

	When("parsing JSON with empty title", func() {
		BeforeEach(func() {
			jsonInput = `{"title": "", "date": "2024-01-15", "amount": 10.50}`
		})

		It("should not return an error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should default to Unknown Expense", func() {
			Expect(data.Title).To(Equal("Unknown Expense"))
		})
	})

	When("parsing JSON with whitespace-only title", func() {
		BeforeEach(func() {
			jsonInput = `{"title": "   ", "date": "2024-01-15", "amount": 10.50}`
		})

		It("should not return an error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should default to Unknown Expense", func() {
			Expect(data.Title).To(Equal("Unknown Expense"))
		})
	})

	When("parsing JSON with no date", func() {
		BeforeEach(func() {
			jsonInput = `{"title": "Test", "date": "", "amount": 10.50}`
		})

		It("should not return an error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should default to today's date", func() {
			expectedDate := time.Now().Format("2006-01-02")
			Expect(data.Date).To(Equal(expectedDate))
		})
	})

	When("parsing invalid JSON", func() {
		BeforeEach(func() {
			jsonInput = `invalid json`
		})

		It("returns the error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
})
