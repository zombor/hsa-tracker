import { Controller } from "https://cdn.skypack.dev/@hotwired/stimulus@3.2.2"

export default class extends Controller {
    static targets = ["content"]

    connect() {
        console.log("Reimbursement detail controller connected")
        // Listen for show events
        this.showHandler = (event) => {
            console.log("Reimbursement detail show event received", event.detail)
            this.show(event.detail.id)
        }
        window.addEventListener("reimbursement-detail:show", this.showHandler)
    }
    
    disconnect() {
        window.removeEventListener("reimbursement-detail:show", this.showHandler)
    }

    async show(reimbursementId) {
        try {
            const response = await fetch("/api/reimbursements/" + reimbursementId)
            if (!response.ok) throw new Error("Failed to load reimbursement")

            const data = await response.json()
            this.display(data.reimbursement, data.receipts)
        } catch (error) {
            alert("Error loading reimbursement: " + error.message)
        }
    }

    display(reimbursement, receipts) {
        const date = new Date(reimbursement.created_at).toLocaleDateString()
        const amount = "$" + (reimbursement.total_amount / 100).toFixed(2)

        let html = `<div class="reimbursement-header">
            <h2>Reimbursement Details</h2>
            <div style="margin-top: 10px;">
                <div><strong>ID:</strong> ${this.escapeHtml(reimbursement.id)}</div>
                <div><strong>Date:</strong> ${date}</div>
                <div><strong>Total Amount:</strong> ${amount}</div>
                <div><strong>Receipts:</strong> ${receipts.length}</div>
            </div>
        </div>
        <h3 style="margin-bottom: 15px;">Receipts</h3>`

        receipts.forEach(receipt => {
            const receiptDate = new Date(receipt.date).toLocaleDateString()
            const receiptAmount = "$" + (receipt.amount / 100).toFixed(2)
            html += `<div class="receipt-item">
                <div class="receipt-info">
                    <div>
                        <div class="receipt-title">${this.escapeHtml(receipt.title)}</div>
                        <div class="receipt-meta">${receiptDate} • ${this.escapeHtml(receipt.filename)}</div>
                    </div>
                </div>
                <div style="display: flex; align-items: center;">
                    <span class="receipt-amount">${receiptAmount}</span>
                    <div class="receipt-actions">
                        <button class="btn-small" data-action="click->reimbursement-detail#viewReceipt" data-receipt-id="${this.escapeHtml(receipt.id)}">View</button>
                    </div>
                </div>
            </div>`
        })

        this.contentTarget.innerHTML = html
        this.element.style.display = "block"
        
        // Hide reimbursements tab content
        const reimbursementsTab = document.querySelector('[data-tab-name="reimbursements"]')
        if (reimbursementsTab) {
            reimbursementsTab.classList.remove("active")
        }
    }

    back() {
        this.element.style.display = "none"
        
        // Show reimbursements tab
        const reimbursementsTab = document.querySelector('[data-tab-name="reimbursements"]')
        if (reimbursementsTab) {
            reimbursementsTab.classList.add("active")
        }

        // Reload reimbursements via custom event
        window.dispatchEvent(new CustomEvent("reimbursements:reload"))
    }

    viewReceipt(event) {
        const receiptId = event.currentTarget.dataset.receiptId
        window.open("/api/receipts/" + receiptId + "/file", "_blank")
    }

    escapeHtml(text) {
        const div = document.createElement("div")
        div.textContent = text
        return div.innerHTML
    }
}

