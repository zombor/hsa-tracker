import { Controller } from "https://cdn.skypack.dev/@hotwired/stimulus@3.2.2"

export default class extends Controller {
    static targets = ["container", "selectionBar", "selectionCount", "totalCount", "totalValue"]
    static values = { selectedIds: Array }

    connect() {
        this.selectedIdsValue = []
        this.load()
        
        // Listen for reload and clear selection events
        this.reloadHandler = () => this.load()
        this.clearSelectionHandler = () => this.clearSelection()
        window.addEventListener("receipts:reload", this.reloadHandler)
        window.addEventListener("receipts:clearSelection", this.clearSelectionHandler)
    }
    
    disconnect() {
        window.removeEventListener("receipts:reload", this.reloadHandler)
        window.removeEventListener("receipts:clearSelection", this.clearSelectionHandler)
    }

    async load() {
        try {
            const response = await fetch("/api/receipts")
            if (!response.ok) throw new Error("Failed to load receipts")

            const receipts = await response.json()
            // Ensure receipts is always an array, even if backend returns null
            const receiptsArray = receipts || []
            this.updateSummary(receiptsArray)
            this.display(receiptsArray)
        } catch (error) {
            this.containerTarget.innerHTML = 
                '<div class="empty-state">Error loading receipts: ' + error.message + "</div>"
        }
    }

    display(receipts) {
        // Handle null or undefined receipts
        if (!receipts || receipts.length === 0) {
            this.containerTarget.innerHTML = 
                '<div class="empty-state">No receipts yet. Upload one to get started!</div>'
            return
        }

        // Sort by receipt date (newest first)
        receipts.sort((a, b) => {
            const dateA = new Date(a.date)
            const dateB = new Date(b.date)
            return dateB - dateA
        })

        this.containerTarget.innerHTML = receipts.map(receipt => {
            const date = new Date(receipt.date).toLocaleDateString()
            const amount = receipt.amount ? "$" + (receipt.amount / 100).toFixed(2) : "N/A"
            const isReimbursed = receipt.reimbursement_id && receipt.reimbursement_id !== ""
            const isSelected = this.selectedIdsValue.includes(receipt.id)
            const checkbox = !isReimbursed ? 
                `<input type="checkbox" data-action="change->receipts#toggle" data-receipt-id="${this.escapeHtml(receipt.id)}" ${isSelected ? "checked" : ""}>` : ""
            const badge = isReimbursed ? '<span class="badge">Reimbursed</span>' : ""
            
            // Extract display filename (remove ID prefix if present)
            let displayFilename = receipt.filename
            const underscoreIndex = displayFilename.indexOf('_')
            if (underscoreIndex > 0 && underscoreIndex < 20) {
                // Likely has ID prefix, remove it
                displayFilename = displayFilename.substring(underscoreIndex + 1)
            }
            // Truncate if still too long
            if (displayFilename.length > 40) {
                displayFilename = displayFilename.substring(0, 37) + '...'
            }

            return `<div class="receipt-item">
                <div class="receipt-info">
                    ${checkbox}
                    <div class="receipt-info-content">
                        <div class="receipt-title">${this.escapeHtml(receipt.title)} ${badge}</div>
                        <div class="receipt-meta">${date} • ${this.escapeHtml(displayFilename)}</div>
                    </div>
                </div>
                <div class="receipt-right">
                    <span class="receipt-amount">${amount}</span>
                    <div class="receipt-actions">
                        <button class="btn-small" data-action="click->receipts#view" data-receipt-id="${this.escapeHtml(receipt.id)}">View</button>
                        ${!isReimbursed ? `<button class="btn-small btn-danger" data-action="click->receipts#delete" data-receipt-id="${this.escapeHtml(receipt.id)}">Delete</button>` : ""}
                    </div>
                </div>
            </div>`
        }).join("")
    }

    toggle(event) {
        const receiptId = event.currentTarget.dataset.receiptId
        if (this.selectedIdsValue.includes(receiptId)) {
            this.selectedIdsValue = this.selectedIdsValue.filter(id => id !== receiptId)
        } else {
            this.selectedIdsValue = [...this.selectedIdsValue, receiptId]
        }
        this.updateSelectionBar()
    }

    selectedIdsValueChanged() {
        this.updateSelectionBar()
    }

    updateSelectionBar() {
        if (!this.hasSelectionBarTarget || !this.hasSelectionCountTarget) {
            return
        }
        const count = this.selectedIdsValue.length
        if (count > 0) {
            this.selectionBarTarget.classList.add("active")
            this.selectionCountTarget.textContent = count + " receipt" + (count > 1 ? "s" : "") + " selected"
        } else {
            this.selectionBarTarget.classList.remove("active")
        }
    }

    async createReimbursement() {
        if (this.selectedIdsValue.length === 0) {
            alert("Please select at least one receipt")
            return
        }

        if (!confirm("Mark " + this.selectedIdsValue.length + " receipt(s) as reimbursed?")) {
            return
        }

        try {
            const response = await fetch("/api/reimbursements", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    receipt_ids: this.selectedIdsValue
                })
            })

            if (!response.ok) {
                let errorMessage = "Failed to create reimbursement"
                try {
                    const errorData = await response.json()
                    if (errorData.error) {
                        errorMessage = errorData.error
                    }
                } catch (e) {
                    errorMessage = response.statusText || "Failed to create reimbursement"
                }
                throw new Error(errorMessage)
            }

            this.selectedIdsValue = []
            this.load()
            
            // Reload reimbursements via custom event
            window.dispatchEvent(new CustomEvent("reimbursements:reload"))

            alert("Receipts marked as reimbursed successfully!")
        } catch (error) {
            alert("Error creating reimbursement: " + error.message)
        }
    }

    view(event) {
        const receiptId = event.currentTarget.dataset.receiptId
        window.open("/api/receipts/" + receiptId + "/file", "_blank")
    }

    async delete(event) {
        const receiptId = event.currentTarget.dataset.receiptId
        if (!confirm("Are you sure you want to delete this receipt?")) {
            return
        }

        try {
            const response = await fetch("/api/receipts/" + receiptId, {
                method: "DELETE"
            })

            if (!response.ok) throw new Error("Delete failed")

            this.selectedIdsValue = this.selectedIdsValue.filter(id => id !== receiptId)
            this.load()
        } catch (error) {
            alert("Error deleting receipt: " + error.message)
        }
    }

    clearSelection() {
        this.selectedIdsValue = []
    }

    updateSummary(receipts) {
        if (!this.hasTotalCountTarget || !this.hasTotalValueTarget) {
            return
        }

        // Handle null or undefined receipts
        const receiptsArray = receipts || []
        const totalCount = receiptsArray.length
        const totalValue = receiptsArray.reduce((sum, receipt) => {
            return sum + (receipt.amount || 0)
        }, 0)

        this.totalCountTarget.textContent = totalCount.toString()
        this.totalValueTarget.textContent = "$" + (totalValue / 100).toFixed(2)
    }

    escapeHtml(text) {
        const div = document.createElement("div")
        div.textContent = text
        return div.innerHTML
    }
}

